package handlers

import (
	"log/slog"
	"net/http"

	"github.com/dukerupert/aletheia/internal/database"
	"github.com/dukerupert/aletheia/internal/session"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

// PageHandler handles template rendering for HTML pages
type PageHandler struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewPageHandler creates a new page handler
func NewPageHandler(pool *pgxpool.Pool, logger *slog.Logger) *PageHandler {
	return &PageHandler{
		pool:   pool,
		logger: logger,
	}
}

// getUserDisplayInfo fetches user from DB and returns display name for nav
func (h *PageHandler) getUserDisplayInfo(c echo.Context, userID [16]byte) map[string]interface{} {
	queries := database.New(h.pool)
	user, err := queries.GetUser(c.Request().Context(), uuidToPgUUID(userID))
	if err != nil {
		// Fallback to generic name if user fetch fails
		return map[string]interface{}{"Name": "User"}
	}

	// Build display name: FirstName LastName, or FirstName, or Username
	displayName := user.Username
	if user.FirstName.Valid && user.LastName.Valid {
		displayName = user.FirstName.String + " " + user.LastName.String
	} else if user.FirstName.Valid {
		displayName = user.FirstName.String
	}

	return map[string]interface{}{"Name": displayName}
}

// HomePage renders the home page
func (h *PageHandler) HomePage(c echo.Context) error {
	data := map[string]interface{}{
		"IsAuthenticated": false, // TODO: Check session
	}
	return c.Render(http.StatusOK, "home.html", data)
}

// NotFoundPage renders the 404 error page
func (h *PageHandler) NotFoundPage(c echo.Context) error {
	data := map[string]interface{}{
		"IsAuthenticated": false,
	}
	return c.Render(http.StatusNotFound, "404.html", data)
}

// ErrorPage renders the 500 error page
func (h *PageHandler) ErrorPage(c echo.Context) error {
	data := map[string]interface{}{
		"IsAuthenticated": false,
	}
	return c.Render(http.StatusInternalServerError, "500.html", data)
}

// LoginPage renders the login page
func (h *PageHandler) LoginPage(c echo.Context) error {
	data := map[string]interface{}{
		"IsAuthenticated": false,
	}
	return c.Render(http.StatusOK, "login.html", data)
}

// DashboardPage renders the dashboard page
func (h *PageHandler) DashboardPage(c echo.Context) error {
	userID, ok := session.GetUserID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
	}

	data := map[string]interface{}{
		"IsAuthenticated": true,
		"User":            h.getUserDisplayInfo(c, userID),
	}
	return c.Render(http.StatusOK, "dashboard.html", data)
}

// RegisterPage renders the registration page
func (h *PageHandler) RegisterPage(c echo.Context) error {
	data := map[string]interface{}{
		"IsAuthenticated": false,
	}
	return c.Render(http.StatusOK, "register.html", data)
}

// VerifyEmailPage renders the email verification page
func (h *PageHandler) VerifyEmailPage(c echo.Context) error {
	// Check if token is provided in query string
	token := c.QueryParam("token")

	data := map[string]interface{}{
		"IsAuthenticated": false,
		"Token":           token,
	}
	return c.Render(http.StatusOK, "verify.html", data)
}

// ForgotPasswordPage renders the forgot password page
func (h *PageHandler) ForgotPasswordPage(c echo.Context) error {
	data := map[string]interface{}{
		"IsAuthenticated": false,
	}
	return c.Render(http.StatusOK, "forgot-password.html", data)
}

// ProjectsPage renders the projects list page
func (h *PageHandler) ProjectsPage(c echo.Context) error {
	// Get user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
	}

	queries := database.New(h.pool)

	// Get user's organizations to get their projects
	memberships, err := queries.ListUserOrganizations(c.Request().Context(), uuidToPgUUID(userID))
	if err != nil {
		h.logger.Error("failed to get user organizations", slog.String("err", err.Error()))
		memberships = []database.OrganizationMember{}
	}

	// Get all projects from user's organizations
	var allProjects []database.Project
	for _, membership := range memberships {
		projects, err := queries.ListProjects(c.Request().Context(), membership.OrganizationID)
		if err != nil {
			h.logger.Error("failed to list projects", slog.String("err", err.Error()))
			continue
		}
		allProjects = append(allProjects, projects...)
	}

	data := map[string]interface{}{
		"IsAuthenticated": true,
		"User":            h.getUserDisplayInfo(c, userID),
		"Projects":        allProjects,
	}
	return c.Render(http.StatusOK, "projects.html", data)
}

// NewProjectPage renders the new project form page
func (h *PageHandler) NewProjectPage(c echo.Context) error {
	// Get user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
	}

	queries := database.New(h.pool)

	// Get user's organizations where they can create projects (owner/admin)
	memberships, err := queries.ListUserOrganizations(c.Request().Context(), uuidToPgUUID(userID))
	if err != nil {
		h.logger.Error("failed to get user organizations", slog.String("err", err.Error()))
		memberships = []database.OrganizationMember{}
	}

	// Filter to only orgs where user is owner or admin
	var organizations []database.Organization
	for _, membership := range memberships {
		if membership.Role == database.OrganizationRoleOwner || membership.Role == database.OrganizationRoleAdmin {
			org, err := queries.GetOrganization(c.Request().Context(), membership.OrganizationID)
			if err != nil {
				continue
			}
			organizations = append(organizations, org)
		}
	}

	data := map[string]interface{}{
		"IsAuthenticated": true,
		"User":            h.getUserDisplayInfo(c, userID),
		"Organizations":   organizations,
	}
	return c.Render(http.StatusOK, "new-project.html", data)
}

// OrganizationsPage renders the organizations list page
func (h *PageHandler) OrganizationsPage(c echo.Context) error {
	// Get user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
	}

	queries := database.New(h.pool)

	// Get user's organizations
	memberships, err := queries.ListUserOrganizations(c.Request().Context(), uuidToPgUUID(userID))
	if err != nil {
		h.logger.Error("failed to get user organizations", slog.String("err", err.Error()))
		memberships = []database.OrganizationMember{}
	}

	// Fetch organization details for each membership
	type OrgWithRole struct {
		ID        pgtype.UUID
		Name      string
		Role      database.OrganizationRole
		CreatedAt pgtype.Timestamptz
	}

	var organizations []OrgWithRole
	for _, membership := range memberships {
		org, err := queries.GetOrganization(c.Request().Context(), membership.OrganizationID)
		if err != nil {
			h.logger.Warn("failed to get organization", slog.String("err", err.Error()))
			continue
		}
		organizations = append(organizations, OrgWithRole{
			ID:        org.ID,
			Name:      org.Name,
			Role:      membership.Role,
			CreatedAt: org.CreatedAt,
		})
	}

	data := map[string]interface{}{
		"IsAuthenticated": true,
		"User":            h.getUserDisplayInfo(c, userID),
		"Organizations":   organizations,
	}
	return c.Render(http.StatusOK, "organizations.html", data)
}

// NewOrganizationPage renders the new organization form page
func (h *PageHandler) NewOrganizationPage(c echo.Context) error {
	// Get user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
	}

	data := map[string]interface{}{
		"IsAuthenticated": true,
		"User":            h.getUserDisplayInfo(c, userID),
	}
	return c.Render(http.StatusOK, "new-organization.html", data)
}

// ProjectDetailPage renders the project detail/edit page
func (h *PageHandler) ProjectDetailPage(c echo.Context) error {
	// Get user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
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

	data := map[string]interface{}{
		"IsAuthenticated": true,
		"User":            h.getUserDisplayInfo(c, userID),
		"Project":         project,
	}
	return c.Render(http.StatusOK, "project-detail.html", data)
}

// ProfilePage renders the user profile page
func (h *PageHandler) ProfilePage(c echo.Context) error {
	// Get user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
	}

	queries := database.New(h.pool)

	// Get user details
	user, err := queries.GetUser(c.Request().Context(), uuidToPgUUID(userID))
	if err != nil {
		h.logger.Error("failed to get user", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	// Build display name
	displayName := user.Username
	if user.FirstName.Valid && user.LastName.Valid {
		displayName = user.FirstName.String + " " + user.LastName.String
	} else if user.FirstName.Valid {
		displayName = user.FirstName.String
	}

	data := map[string]interface{}{
		"IsAuthenticated": true,
		"User": map[string]interface{}{
			"Name": displayName,
		},
		"Profile": user,
	}
	return c.Render(http.StatusOK, "profile.html", data)
}

// InspectionsPage renders the inspections list page for a project
func (h *PageHandler) InspectionsPage(c echo.Context) error {
	// Get user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
	}

	projectID := c.Param("projectId")
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

	// Verify user is a member of the organization
	_, err = queries.GetOrganizationMemberByUserAndOrg(c.Request().Context(), database.GetOrganizationMemberByUserAndOrgParams{
		OrganizationID: project.OrganizationID,
		UserID:         uuidToPgUUID(userID),
	})
	if err != nil {
		h.logger.Warn("user not authorized to access project inspections",
			slog.String("user_id", userID.String()),
			slog.String("project_id", projectID))
		return echo.NewHTTPError(http.StatusForbidden, "you are not a member of this project's organization")
	}

	// Get all inspections for the project
	inspections, err := queries.ListInspections(c.Request().Context(), projectUUID)
	if err != nil {
		h.logger.Error("failed to list inspections", slog.String("err", err.Error()))
		inspections = []database.Inspection{}
	}

	data := map[string]interface{}{
		"IsAuthenticated": true,
		"User":            h.getUserDisplayInfo(c, userID),
		"Project":         project,
		"Inspections":     inspections,
	}
	return c.Render(http.StatusOK, "inspections.html", data)
}

// NewInspectionPage renders the new inspection form page
func (h *PageHandler) NewInspectionPage(c echo.Context) error {
	// Get user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
	}

	projectID := c.Param("projectId")
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

	// Verify user is a member of the organization
	_, err = queries.GetOrganizationMemberByUserAndOrg(c.Request().Context(), database.GetOrganizationMemberByUserAndOrgParams{
		OrganizationID: project.OrganizationID,
		UserID:         uuidToPgUUID(userID),
	})
	if err != nil {
		h.logger.Warn("user not authorized to create inspection",
			slog.String("user_id", userID.String()),
			slog.String("project_id", projectID))
		return echo.NewHTTPError(http.StatusForbidden, "you are not a member of this project's organization")
	}

	data := map[string]interface{}{
		"IsAuthenticated": true,
		"User":            h.getUserDisplayInfo(c, userID),
		"Project":         project,
	}
	return c.Render(http.StatusOK, "new-inspection.html", data)
}

// AllInspectionsPage renders a global view of all inspections across all projects
func (h *PageHandler) AllInspectionsPage(c echo.Context) error {
	// Get user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
	}

	queries := database.New(h.pool)

	// Get user's organizations
	memberships, err := queries.ListUserOrganizations(c.Request().Context(), uuidToPgUUID(userID))
	if err != nil {
		h.logger.Error("failed to get user organizations", slog.String("err", err.Error()))
		memberships = []database.OrganizationMember{}
	}

	// Collect all inspections with project context
	type InspectionWithContext struct {
		Inspection      database.Inspection
		ProjectID       pgtype.UUID
		ProjectName     string
		ProjectLocation string
	}

	var allInspections []InspectionWithContext

	for _, membership := range memberships {
		// Get all projects in this organization
		projects, err := queries.ListProjects(c.Request().Context(), membership.OrganizationID)
		if err != nil {
			h.logger.Warn("failed to list projects", slog.String("err", err.Error()))
			continue
		}

		// Get inspections for each project
		for _, project := range projects {
			inspections, err := queries.ListInspections(c.Request().Context(), project.ID)
			if err != nil {
				h.logger.Warn("failed to list inspections", slog.String("err", err.Error()))
				continue
			}

			// Build location string
			location := ""
			if project.City.Valid && project.State.Valid {
				location = project.City.String + ", " + project.State.String
			} else if project.State.Valid {
				location = project.State.String
			} else if project.Address.Valid {
				location = project.Address.String
			}

			for _, inspection := range inspections {
				allInspections = append(allInspections, InspectionWithContext{
					Inspection:      inspection,
					ProjectID:       project.ID,
					ProjectName:     project.Name,
					ProjectLocation: location,
				})
			}
		}
	}

	data := map[string]interface{}{
		"IsAuthenticated": true,
		"User":            h.getUserDisplayInfo(c, userID),
		"Inspections":     allInspections,
	}
	return c.Render(http.StatusOK, "all-inspections.html", data)
}
