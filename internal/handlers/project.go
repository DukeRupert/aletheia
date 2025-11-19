package handlers

import (
	"log/slog"
	"net/http"

	"github.com/dukerupert/aletheia/internal/database"
	"github.com/dukerupert/aletheia/internal/session"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

type ProjectHandler struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewProjectHandler(pool *pgxpool.Pool, logger *slog.Logger) *ProjectHandler {
	return &ProjectHandler{
		pool:   pool,
		logger: logger,
	}
}

// CreateProjectRequest is the request payload for creating a project
type CreateProjectRequest struct {
	OrganizationID string `json:"organization_id" validate:"required"`
	Name           string `json:"name" validate:"required,min=1,max=255"`
}

// CreateProjectResponse is the response payload for project creation
type CreateProjectResponse struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	CreatedAt      string `json:"created_at"`
}

// CreateProject creates a new project within an organization (owner/admin only)
func (h *ProjectHandler) CreateProject(c echo.Context) error {
	// Get authenticated user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	// Parse request
	var req CreateProjectRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.OrganizationID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "organization_id is required")
	}

	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "project name is required")
	}

	queries := database.New(h.pool)

	// Parse organization ID
	orgUUID, err := parseUUID(req.OrganizationID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid organization_id")
	}

	// Verify user is owner or admin of the organization
	membership, err := queries.GetOrganizationMemberByUserAndOrg(c.Request().Context(), database.GetOrganizationMemberByUserAndOrgParams{
		OrganizationID: orgUUID,
		UserID:         uuidToPgUUID(userID),
	})
	if err != nil {
		h.logger.Warn("user not authorized to create project in organization",
			slog.String("user_id", userID.String()),
			slog.String("org_id", req.OrganizationID))
		return echo.NewHTTPError(http.StatusForbidden, "you are not a member of this organization")
	}

	if membership.Role != database.OrganizationRoleOwner && membership.Role != database.OrganizationRoleAdmin {
		return echo.NewHTTPError(http.StatusForbidden, "only owners and admins can create projects")
	}

	// Create project
	project, err := queries.CreateProject(c.Request().Context(), database.CreateProjectParams{
		OrganizationID: orgUUID,
		Name:           req.Name,
	})
	if err != nil {
		h.logger.Error("failed to create project", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
	}

	h.logger.Info("project created",
		slog.String("project_id", project.ID.String()),
		slog.String("org_id", req.OrganizationID),
		slog.String("user_id", userID.String()))

	return c.JSON(http.StatusCreated, CreateProjectResponse{
		ID:             project.ID.String(),
		OrganizationID: project.OrganizationID.String(),
		Name:           project.Name,
		CreatedAt:      project.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// GetProjectResponse is the response payload for project retrieval
type GetProjectResponse struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// GetProject retrieves a project by ID (organization members only)
func (h *ProjectHandler) GetProject(c echo.Context) error {
	// Get authenticated user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	projectID := c.Param("id")
	if projectID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "project id is required")
	}

	queries := database.New(h.pool)

	// Parse project ID
	projectUUID, err := parseUUID(projectID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid project id")
	}

	// Get project
	project, err := queries.GetProject(c.Request().Context(), projectUUID)
	if err != nil {
		h.logger.Error("failed to get project", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusNotFound, "project not found")
	}

	// Verify user is a member of the organization that owns this project
	_, err = queries.GetOrganizationMemberByUserAndOrg(c.Request().Context(), database.GetOrganizationMemberByUserAndOrgParams{
		OrganizationID: project.OrganizationID,
		UserID:         uuidToPgUUID(userID),
	})
	if err != nil {
		h.logger.Warn("user not authorized to access project",
			slog.String("user_id", userID.String()),
			slog.String("project_id", projectID))
		return echo.NewHTTPError(http.StatusForbidden, "you are not a member of this project's organization")
	}

	return c.JSON(http.StatusOK, GetProjectResponse{
		ID:             project.ID.String(),
		OrganizationID: project.OrganizationID.String(),
		Name:           project.Name,
		CreatedAt:      project.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      project.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// ListProjectsResponse is the response payload for listing projects
type ProjectSummary struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	CreatedAt      string `json:"created_at"`
}

type ListProjectsResponse struct {
	Projects []ProjectSummary `json:"projects"`
}

// ListProjects lists all projects in an organization (organization members only)
func (h *ProjectHandler) ListProjects(c echo.Context) error {
	// Get authenticated user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	orgID := c.Param("orgId")
	if orgID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "organization id is required")
	}

	queries := database.New(h.pool)

	// Parse organization ID
	orgUUID, err := parseUUID(orgID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid organization id")
	}

	// Verify user is a member of the organization
	_, err = queries.GetOrganizationMemberByUserAndOrg(c.Request().Context(), database.GetOrganizationMemberByUserAndOrgParams{
		OrganizationID: orgUUID,
		UserID:         uuidToPgUUID(userID),
	})
	if err != nil {
		h.logger.Warn("user not authorized to list projects in organization",
			slog.String("user_id", userID.String()),
			slog.String("org_id", orgID))
		return echo.NewHTTPError(http.StatusForbidden, "you are not a member of this organization")
	}

	// Get all projects for the organization
	projects, err := queries.ListProjects(c.Request().Context(), orgUUID)
	if err != nil {
		h.logger.Error("failed to list projects", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list projects")
	}

	projectSummaries := make([]ProjectSummary, len(projects))
	for i, project := range projects {
		projectSummaries[i] = ProjectSummary{
			ID:             project.ID.String(),
			OrganizationID: project.OrganizationID.String(),
			Name:           project.Name,
			CreatedAt:      project.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return c.JSON(http.StatusOK, ListProjectsResponse{
		Projects: projectSummaries,
	})
}

// UpdateProjectRequest is the request payload for updating a project
type UpdateProjectRequest struct {
	Name *string `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
}

// UpdateProjectResponse is the response payload for project update
type UpdateProjectResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	UpdatedAt string `json:"updated_at"`
}

// UpdateProject updates a project (owner/admin only)
func (h *ProjectHandler) UpdateProject(c echo.Context) error {
	// Get authenticated user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	projectID := c.Param("id")
	if projectID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "project id is required")
	}

	// Parse request
	var req UpdateProjectRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	queries := database.New(h.pool)

	// Parse project ID
	projectUUID, err := parseUUID(projectID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid project id")
	}

	// Get project to find its organization
	project, err := queries.GetProject(c.Request().Context(), projectUUID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "project not found")
	}

	// Verify user is owner or admin of the organization
	membership, err := queries.GetOrganizationMemberByUserAndOrg(c.Request().Context(), database.GetOrganizationMemberByUserAndOrgParams{
		OrganizationID: project.OrganizationID,
		UserID:         uuidToPgUUID(userID),
	})
	if err != nil {
		h.logger.Warn("user not authorized to update project",
			slog.String("user_id", userID.String()),
			slog.String("project_id", projectID))
		return echo.NewHTTPError(http.StatusForbidden, "you are not a member of this project's organization")
	}

	if membership.Role != database.OrganizationRoleOwner && membership.Role != database.OrganizationRoleAdmin {
		return echo.NewHTTPError(http.StatusForbidden, "only owners and admins can update projects")
	}

	// Update project
	params := database.UpdateProjectParams{
		ID: projectUUID,
	}

	if req.Name != nil {
		params.Name = *req.Name
	}

	updatedProject, err := queries.UpdateProject(c.Request().Context(), params)
	if err != nil {
		h.logger.Error("failed to update project", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update project")
	}

	h.logger.Info("project updated", slog.String("project_id", projectID))

	return c.JSON(http.StatusOK, UpdateProjectResponse{
		ID:        updatedProject.ID.String(),
		Name:      updatedProject.Name,
		UpdatedAt: updatedProject.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// DeleteProject deletes a project (owner/admin only)
func (h *ProjectHandler) DeleteProject(c echo.Context) error {
	// Get authenticated user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	projectID := c.Param("id")
	if projectID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "project id is required")
	}

	queries := database.New(h.pool)

	// Parse project ID
	projectUUID, err := parseUUID(projectID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid project id")
	}

	// Get project to find its organization
	project, err := queries.GetProject(c.Request().Context(), projectUUID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "project not found")
	}

	// Verify user is owner or admin of the organization
	membership, err := queries.GetOrganizationMemberByUserAndOrg(c.Request().Context(), database.GetOrganizationMemberByUserAndOrgParams{
		OrganizationID: project.OrganizationID,
		UserID:         uuidToPgUUID(userID),
	})
	if err != nil {
		h.logger.Warn("user not authorized to delete project",
			slog.String("user_id", userID.String()),
			slog.String("project_id", projectID))
		return echo.NewHTTPError(http.StatusForbidden, "you are not a member of this project's organization")
	}

	if membership.Role != database.OrganizationRoleOwner && membership.Role != database.OrganizationRoleAdmin {
		return echo.NewHTTPError(http.StatusForbidden, "only owners and admins can delete projects")
	}

	// Delete project
	err = queries.DeleteProject(c.Request().Context(), projectUUID)
	if err != nil {
		h.logger.Error("failed to delete project", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete project")
	}

	h.logger.Info("project deleted", slog.String("project_id", projectID))

	return c.NoContent(http.StatusNoContent)
}
