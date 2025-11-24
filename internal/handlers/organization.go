package handlers

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/dukerupert/aletheia/internal/database"
	"github.com/dukerupert/aletheia/internal/session"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

type OrganizationHandler struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewOrganizationHandler(pool *pgxpool.Pool, logger *slog.Logger) *OrganizationHandler {
	return &OrganizationHandler{
		pool:   pool,
		logger: logger,
	}
}


// CreateOrganizationRequest is the request payload for creating an organization
type CreateOrganizationRequest struct {
	Name string `json:"name" form:"name" validate:"required,min=1,max=255"`
}

// CreateOrganizationResponse is the response payload for organization creation
type CreateOrganizationResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

// CreateOrganization creates a new organization and adds the creating user as owner
func (h *OrganizationHandler) CreateOrganization(c echo.Context) error {
	// Get authenticated user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	// Parse request
	var req CreateOrganizationRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// Sanitize and validate name
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "organization name is required")
	}
	if len(req.Name) > 255 {
		return echo.NewHTTPError(http.StatusBadRequest, "organization name too long (max 255 characters)")
	}

	// Begin transaction to ensure atomicity
	tx, err := h.pool.Begin(c.Request().Context())
	if err != nil {
		h.logger.Error("failed to begin transaction", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create organization")
	}
	defer tx.Rollback(c.Request().Context()) // Rollback if not committed

	queries := database.New(tx)

	// Create organization
	org, err := queries.CreateOrganization(c.Request().Context(), req.Name)
	if err != nil {
		h.logger.Error("failed to create organization", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create organization")
	}

	// Add creator as owner
	_, err = queries.AddOrganizationMember(c.Request().Context(), database.AddOrganizationMemberParams{
		OrganizationID: org.ID,
		UserID:         uuidToPgUUID(userID),
		Role:           database.OrganizationRoleOwner,
	})
	if err != nil {
		h.logger.Error("failed to add organization owner", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create organization")
	}

	// Commit transaction
	if err := tx.Commit(c.Request().Context()); err != nil {
		h.logger.Error("failed to commit transaction", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create organization")
	}

	h.logger.Info("organization created", slog.String("org_id", org.ID.String()), slog.String("user_id", userID.String()))

	// Check if this is an HTMX request
	if c.Request().Header.Get("HX-Request") == "true" {
		// HTMX request - redirect to organizations list
		c.Response().Header().Set("HX-Redirect", "/organizations")
		return c.NoContent(http.StatusOK)
	}

	// Regular API request - return JSON
	return c.JSON(http.StatusCreated, CreateOrganizationResponse{
		ID:        org.ID.String(),
		Name:      org.Name,
		CreatedAt: org.CreatedAt.Time.Format(RFC3339Format),
	})
}

// GetOrganizationResponse is the response payload for organization retrieval
type GetOrganizationResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// GetOrganization retrieves an organization by ID
func (h *OrganizationHandler) GetOrganization(c echo.Context) error {
	// Get authenticated user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	orgID := c.Param("id")
	if orgID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "organization id is required")
	}

	// Parse organization ID
	orgUUID, err := parseUUID(orgID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid organization id")
	}

	// Check if user is a member of the organization
	_, err = requireOrganizationMembership(c.Request().Context(), h.pool, h.logger, userID, orgUUID)
	if err != nil {
		return err
	}

	// Get organization
	queries := database.New(h.pool)
	org, err := queries.GetOrganization(c.Request().Context(), orgUUID)
	if err != nil {
		h.logger.Error("failed to get organization", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusNotFound, "organization not found")
	}

	return c.JSON(http.StatusOK, GetOrganizationResponse{
		ID:        org.ID.String(),
		Name:      org.Name,
		CreatedAt: org.CreatedAt.Time.Format(RFC3339Format),
		UpdatedAt: org.UpdatedAt.Time.Format(RFC3339Format),
	})
}

// ListOrganizationsResponse is the response payload for listing user's organizations
type OrganizationSummary struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

type ListOrganizationsResponse struct {
	Organizations []OrganizationSummary `json:"organizations"`
}

// ListOrganizations lists all organizations the authenticated user is a member of
func (h *OrganizationHandler) ListOrganizations(c echo.Context) error {
	// Get authenticated user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	queries := database.New(h.pool)

	// Get all organizations with user membership details in a single query (avoids N+1 problem)
	orgs, err := queries.ListUserOrganizationsWithDetails(c.Request().Context(), uuidToPgUUID(userID))
	if err != nil {
		h.logger.Error("failed to list user organizations", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list organizations")
	}

	// Build response
	organizations := make([]OrganizationSummary, len(orgs))
	for i, org := range orgs {
		organizations[i] = OrganizationSummary{
			ID:        org.ID.String(),
			Name:      org.Name,
			Role:      string(org.Role),
			CreatedAt: org.CreatedAt.Time.Format(RFC3339Format),
		}
	}

	return c.JSON(http.StatusOK, ListOrganizationsResponse{
		Organizations: organizations,
	})
}

// UpdateOrganizationRequest is the request payload for updating an organization
type UpdateOrganizationRequest struct {
	Name *string `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
}

// UpdateOrganizationResponse is the response payload for organization update
type UpdateOrganizationResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	UpdatedAt string `json:"updated_at"`
}

// UpdateOrganization updates an organization (owner/admin only)
func (h *OrganizationHandler) UpdateOrganization(c echo.Context) error {
	// Get authenticated user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	orgID := c.Param("id")
	if orgID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "organization id is required")
	}

	// Parse request
	var req UpdateOrganizationRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// Parse organization ID
	orgUUID, err := parseUUID(orgID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid organization id")
	}

	// Check if user is owner or admin
	_, err = requireOrganizationMembership(c.Request().Context(), h.pool, h.logger, userID, orgUUID, database.OrganizationRoleOwner, database.OrganizationRoleAdmin)
	if err != nil {
		return err
	}

	// Validate that at least one field is being updated
	if req.Name == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "no fields to update")
	}

	// Sanitize and validate name
	sanitizedName := strings.TrimSpace(*req.Name)
	if sanitizedName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "organization name cannot be empty")
	}
	if len(sanitizedName) > 255 {
		return echo.NewHTTPError(http.StatusBadRequest, "organization name too long (max 255 characters)")
	}

	// Update organization
	params := database.UpdateOrganizationParams{
		ID:   orgUUID,
		Name: sanitizedName,
	}

	queries := database.New(h.pool)
	org, err := queries.UpdateOrganization(c.Request().Context(), params)
	if err != nil {
		h.logger.Error("failed to update organization", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update organization")
	}

	h.logger.Info("organization updated", slog.String("org_id", org.ID.String()))

	return c.JSON(http.StatusOK, UpdateOrganizationResponse{
		ID:        org.ID.String(),
		Name:      org.Name,
		UpdatedAt: org.UpdatedAt.Time.Format(RFC3339Format),
	})
}

// DeleteOrganization deletes an organization (owner only)
func (h *OrganizationHandler) DeleteOrganization(c echo.Context) error {
	// Get authenticated user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	orgID := c.Param("id")
	if orgID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "organization id is required")
	}

	// Parse organization ID
	orgUUID, err := parseUUID(orgID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid organization id")
	}

	// Check if user is owner (only owners can delete)
	_, err = requireOrganizationMembership(c.Request().Context(), h.pool, h.logger, userID, orgUUID, database.OrganizationRoleOwner)
	if err != nil {
		return err
	}

	// Delete organization
	queries := database.New(h.pool)
	err = queries.DeleteOrganization(c.Request().Context(), orgUUID)
	if err != nil {
		h.logger.Error("failed to delete organization", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete organization")
	}

	h.logger.Info("organization deleted", slog.String("org_id", orgID))

	return c.NoContent(http.StatusNoContent)
}

// ListOrganizationMembersResponse is the response payload for listing organization members
type MemberSummary struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

type ListOrganizationMembersResponse struct {
	Members []MemberSummary `json:"members"`
}

// ListOrganizationMembers lists all members of an organization
func (h *OrganizationHandler) ListOrganizationMembers(c echo.Context) error {
	// Get authenticated user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	orgID := c.Param("id")
	if orgID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "organization id is required")
	}

	// Parse organization ID
	orgUUID, err := parseUUID(orgID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid organization id")
	}

	// Check if user is a member of the organization
	_, err = requireOrganizationMembership(c.Request().Context(), h.pool, h.logger, userID, orgUUID)
	if err != nil {
		return err
	}

	// Get organization members
	queries := database.New(h.pool)
	members, err := queries.ListOrganizationMembers(c.Request().Context(), orgUUID)
	if err != nil {
		h.logger.Error("failed to list organization members", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list organization members")
	}

	memberSummaries := make([]MemberSummary, len(members))
	for i, member := range members {
		memberSummaries[i] = MemberSummary{
			ID:        member.ID.String(),
			UserID:    member.UserID.String(),
			Role:      string(member.Role),
			CreatedAt: member.CreatedAt.Time.Format(RFC3339Format),
		}
	}

	return c.JSON(http.StatusOK, ListOrganizationMembersResponse{
		Members: memberSummaries,
	})
}

// AddOrganizationMemberRequest is the request payload for adding a member
type AddOrganizationMemberRequest struct {
	Email string `json:"email" validate:"required,email"`
	Role  string `json:"role" validate:"required,oneof=admin member"`
}

// AddOrganizationMemberResponse is the response payload for adding a member
type AddOrganizationMemberResponse struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

// AddOrganizationMember adds a new member to the organization (owner/admin only)
func (h *OrganizationHandler) AddOrganizationMember(c echo.Context) error {
	// Get authenticated user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	orgID := c.Param("id")
	if orgID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "organization id is required")
	}

	// Parse request
	var req AddOrganizationMemberRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Email == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "email is required")
	}

	if req.Role == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "role is required")
	}

	// Parse organization ID
	orgUUID, err := parseUUID(orgID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid organization id")
	}

	// Check if requester is owner or admin
	_, err = requireOrganizationMembership(c.Request().Context(), h.pool, h.logger, userID, orgUUID, database.OrganizationRoleOwner, database.OrganizationRoleAdmin)
	if err != nil {
		return err
	}

	// Find user by email
	queries := database.New(h.pool)
	targetUser, err := queries.GetUserByEmail(c.Request().Context(), req.Email)
	if err != nil {
		h.logger.Warn("user not found for organization invite", slog.String("email", req.Email))
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	// Check if user is already a member
	_, err = queries.GetOrganizationMemberByUserAndOrg(c.Request().Context(), database.GetOrganizationMemberByUserAndOrgParams{
		OrganizationID: orgUUID,
		UserID:         targetUser.ID,
	})
	if err == nil {
		return echo.NewHTTPError(http.StatusConflict, "user is already a member of this organization")
	}

	// Parse role
	var role database.OrganizationRole
	switch req.Role {
	case "admin":
		role = database.OrganizationRoleAdmin
	case "member":
		role = database.OrganizationRoleMember
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "invalid role")
	}

	// Add member
	newMember, err := queries.AddOrganizationMember(c.Request().Context(), database.AddOrganizationMemberParams{
		OrganizationID: orgUUID,
		UserID:         targetUser.ID,
		Role:           role,
	})
	if err != nil {
		h.logger.Error("failed to add organization member", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to add organization member")
	}

	h.logger.Info("organization member added",
		slog.String("org_id", orgID),
		slog.String("user_id", targetUser.ID.String()),
		slog.String("role", req.Role))

	return c.JSON(http.StatusCreated, AddOrganizationMemberResponse{
		ID:        newMember.ID.String(),
		UserID:    newMember.UserID.String(),
		Role:      string(newMember.Role),
		CreatedAt: newMember.CreatedAt.Time.Format(RFC3339Format),
	})
}

// UpdateOrganizationMemberRequest is the request payload for updating a member role
type UpdateOrganizationMemberRequest struct {
	Role string `json:"role" validate:"required,oneof=admin member"`
}

// UpdateOrganizationMemberResponse is the response payload for updating a member
type UpdateOrganizationMemberResponse struct {
	ID   string `json:"id"`
	Role string `json:"role"`
}

// UpdateOrganizationMember updates a member's role (owner/admin only)
func (h *OrganizationHandler) UpdateOrganizationMember(c echo.Context) error {
	// Get authenticated user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	orgID := c.Param("id")
	memberID := c.Param("memberId")
	if orgID == "" || memberID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "organization id and member id are required")
	}

	// Parse request
	var req UpdateOrganizationMemberRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Role == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "role is required")
	}

	// Parse IDs
	orgUUID, err := parseUUID(orgID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid organization id")
	}

	memberUUID, err := parseUUID(memberID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid member id")
	}

	// Check if requester is owner or admin
	_, err = requireOrganizationMembership(c.Request().Context(), h.pool, h.logger, userID, orgUUID, database.OrganizationRoleOwner, database.OrganizationRoleAdmin)
	if err != nil {
		return err
	}

	// Get the member being updated
	queries := database.New(h.pool)
	targetMember, err := queries.GetOrganizationMember(c.Request().Context(), memberUUID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "member not found")
	}

	// Verify member belongs to this organization
	if targetMember.OrganizationID != orgUUID {
		return echo.NewHTTPError(http.StatusBadRequest, "member does not belong to this organization")
	}

	// Prevent changing owner role
	if targetMember.Role == database.OrganizationRoleOwner {
		return echo.NewHTTPError(http.StatusForbidden, "cannot change owner role")
	}

	// Parse new role
	var role database.OrganizationRole
	switch req.Role {
	case "admin":
		role = database.OrganizationRoleAdmin
	case "member":
		role = database.OrganizationRoleMember
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "invalid role")
	}

	// Update member role
	updatedMember, err := queries.UpdateOrganizationMemberRole(c.Request().Context(), database.UpdateOrganizationMemberRoleParams{
		ID:   memberUUID,
		Role: role,
	})
	if err != nil {
		h.logger.Error("failed to update organization member role", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update member role")
	}

	h.logger.Info("organization member role updated",
		slog.String("org_id", orgID),
		slog.String("member_id", memberID),
		slog.String("new_role", req.Role))

	return c.JSON(http.StatusOK, UpdateOrganizationMemberResponse{
		ID:   updatedMember.ID.String(),
		Role: string(updatedMember.Role),
	})
}

// RemoveOrganizationMember removes a member from the organization (owner/admin only)
func (h *OrganizationHandler) RemoveOrganizationMember(c echo.Context) error {
	// Get authenticated user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	orgID := c.Param("id")
	memberID := c.Param("memberId")
	if orgID == "" || memberID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "organization id and member id are required")
	}

	// Parse IDs
	orgUUID, err := parseUUID(orgID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid organization id")
	}

	memberUUID, err := parseUUID(memberID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid member id")
	}

	// Check if requester is owner or admin
	_, err = requireOrganizationMembership(c.Request().Context(), h.pool, h.logger, userID, orgUUID, database.OrganizationRoleOwner, database.OrganizationRoleAdmin)
	if err != nil {
		return err
	}

	// Get the member being removed
	queries := database.New(h.pool)
	targetMember, err := queries.GetOrganizationMember(c.Request().Context(), memberUUID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "member not found")
	}

	// Verify member belongs to this organization
	if targetMember.OrganizationID != orgUUID {
		return echo.NewHTTPError(http.StatusBadRequest, "member does not belong to this organization")
	}

	// Prevent removing owner
	if targetMember.Role == database.OrganizationRoleOwner {
		return echo.NewHTTPError(http.StatusForbidden, "cannot remove owner")
	}

	// Remove member
	err = queries.RemoveOrganizationMember(c.Request().Context(), memberUUID)
	if err != nil {
		h.logger.Error("failed to remove organization member", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to remove member")
	}

	h.logger.Info("organization member removed",
		slog.String("org_id", orgID),
		slog.String("member_id", memberID))

	return c.NoContent(http.StatusNoContent)
}
