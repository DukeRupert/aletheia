package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/dukerupert/aletheia/internal/database"
	"github.com/dukerupert/aletheia/internal/session"
	"github.com/jackc/pgx/v5"
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
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	queries := database.New(h.pool)
	user, err := queries.GetUser(ctx, uuidToPgUUID(userID))
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
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	userID, ok := session.GetUserID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
	}

	queries := database.New(h.pool)

	// Get user's organizations
	orgMemberships, err := queries.ListUserOrganizations(ctx, uuidToPgUUID(userID))
	if err != nil || len(orgMemberships) == 0 {
		h.logger.Error("failed to get user organizations",
			slog.String("user_id", userID.String()),
			slog.String("err", err.Error()))

		// Show empty state for users with no organizations
		data := map[string]interface{}{
			"IsAuthenticated": true,
			"User":            h.getUserDisplayInfo(c, userID),
			"HasOrganization": false,
		}
		return c.Render(http.StatusOK, "dashboard.html", data)
	}

	// Use first organization (TODO: add organization switcher)
	orgID := orgMemberships[0].OrganizationID

	// Calculate date ranges
	nowTime := time.Now()
	weekStartTime := nowTime.AddDate(0, 0, -7)
	monthStartTime := nowTime.AddDate(0, -1, 0)

	now := pgtype.Timestamptz{Time: nowTime, Valid: true}
	weekStart := pgtype.Timestamptz{Time: weekStartTime, Valid: true}
	monthStart := pgtype.Timestamptz{Time: monthStartTime, Valid: true}

	// Get stats for this week
	inspectionsThisWeek, err := queries.GetInspectionCountByOrganizationAndDateRange(ctx,
		database.GetInspectionCountByOrganizationAndDateRangeParams{
			OrganizationID: orgID,
			CreatedAt:      weekStart,
			CreatedAt_2:    now,
		})
	if err != nil {
		h.logger.Error("failed to get inspection count", slog.String("err", err.Error()))
		inspectionsThisWeek = 0
	}

	violationsThisWeek, err := queries.GetViolationCountByOrganizationAndDateRange(ctx,
		database.GetViolationCountByOrganizationAndDateRangeParams{
			OrganizationID: orgID,
			CreatedAt:      weekStart,
			CreatedAt_2:    now,
		})
	if err != nil {
		h.logger.Error("failed to get violation count", slog.String("err", err.Error()))
		violationsThisWeek = 0
	}

	photosThisWeek, err := queries.GetPhotoCountByOrganizationAndDateRange(ctx,
		database.GetPhotoCountByOrganizationAndDateRangeParams{
			OrganizationID: orgID,
			CreatedAt:      weekStart,
			CreatedAt_2:    now,
		})
	if err != nil {
		h.logger.Error("failed to get photo count", slog.String("err", err.Error()))
		photosThisWeek = 0
	}

	reportsThisMonth, err := queries.GetReportCountByOrganizationAndDateRange(ctx,
		database.GetReportCountByOrganizationAndDateRangeParams{
			OrganizationID: orgID,
			CreatedAt:      monthStart,
			CreatedAt_2:    now,
		})
	if err != nil {
		h.logger.Error("failed to get report count", slog.String("err", err.Error()))
		reportsThisMonth = 0
	}

	// Get violation breakdown by severity
	violationsBySeverity, err := queries.GetViolationCountBySeverityAndOrganization(ctx,
		database.GetViolationCountBySeverityAndOrganizationParams{
			OrganizationID: orgID,
			CreatedAt:      weekStart,
			CreatedAt_2:    now,
		})
	if err != nil {
		h.logger.Error("failed to get violations by severity", slog.String("err", err.Error()))
		violationsBySeverity = []database.GetViolationCountBySeverityAndOrganizationRow{}
	}

	// Build severity sub-value string
	var criticalCount int64
	for _, v := range violationsBySeverity {
		if v.Severity == database.ViolationSeverityCritical {
			criticalCount = v.Count
			break
		}
	}
	violationsSubValue := ""
	if criticalCount > 0 {
		violationsSubValue = fmt.Sprintf("%d critical", criticalCount)
	}

	// Get recent inspections
	recentInspections, err := queries.GetRecentInspectionsByOrganization(ctx,
		database.GetRecentInspectionsByOrganizationParams{
			OrganizationID: orgID,
			Limit:          int32(10),
		})
	if err != nil {
		h.logger.Error("failed to get recent inspections", slog.String("err", err.Error()))
		recentInspections = []database.GetRecentInspectionsByOrganizationRow{}
	}

	// Build stats cards
	statsCards := []map[string]interface{}{
		{
			"Label":   "Inspections This Week",
			"Value":   inspectionsThisWeek,
			"Icon":    "clipboard",
			"Color":   "blue",
			"URL":     "/inspections",
		},
		{
			"Label":    "Violations Detected",
			"Value":    violationsThisWeek,
			"SubValue": violationsSubValue,
			"Icon":     "alert",
			"Color":    "red",
		},
		{
			"Label": "Reports Generated",
			"Value": reportsThisMonth,
			"Icon":  "document",
			"Color": "green",
		},
		{
			"Label": "Photos Analyzed",
			"Value": photosThisWeek,
			"Icon":  "photo",
			"Color": "orange",
		},
	}

	data := map[string]interface{}{
		"IsAuthenticated":   true,
		"User":              h.getUserDisplayInfo(c, userID),
		"HasOrganization":   true,
		"StatsCards":        statsCards,
		"RecentInspections": recentInspections,
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
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	// Get user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
	}

	queries := database.New(h.pool)

	// Get user's organizations to get their projects
	memberships, err := queries.ListUserOrganizations(ctx, uuidToPgUUID(userID))
	if err != nil {
		h.logger.Error("failed to get user organizations", slog.String("err", err.Error()))
		memberships = []database.OrganizationMember{}
	}

	// Get all projects from user's organizations
	var allProjects []database.Project
	for _, membership := range memberships {
		projects, err := queries.ListProjects(ctx, membership.OrganizationID)
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
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	// Get user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
	}

	queries := database.New(h.pool)

	// Get user's organizations where they can create projects (owner/admin)
	memberships, err := queries.ListUserOrganizations(ctx, uuidToPgUUID(userID))
	if err != nil {
		h.logger.Error("failed to get user organizations", slog.String("err", err.Error()))
		memberships = []database.OrganizationMember{}
	}

	// Filter to only orgs where user is owner or admin
	var organizations []database.Organization
	for _, membership := range memberships {
		if membership.Role == database.OrganizationRoleOwner || membership.Role == database.OrganizationRoleAdmin {
			org, err := queries.GetOrganization(ctx, membership.OrganizationID)
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
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	// Get user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
	}

	queries := database.New(h.pool)

	// Get user's organizations
	memberships, err := queries.ListUserOrganizations(ctx, uuidToPgUUID(userID))
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
		org, err := queries.GetOrganization(ctx, membership.OrganizationID)
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
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

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
	project, err := queries.GetProject(ctx, projectUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "project not found")
		}
		h.logger.Error("failed to get project", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load project")
	}

	// Verify user is a member of the organization that owns this project
	_, err = requireOrganizationMembership(ctx, h.pool, h.logger, userID, project.OrganizationID)
	if err != nil {
		return err
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
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	// Get user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
	}

	queries := database.New(h.pool)

	// Get user details
	user, err := queries.GetUser(ctx, uuidToPgUUID(userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		h.logger.Error("failed to get user", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load user")
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
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

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
	project, err := queries.GetProject(ctx, projectUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "project not found")
		}
		h.logger.Error("failed to get project", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load project")
	}

	// Verify user is a member of the organization
	_, err = requireOrganizationMembership(ctx, h.pool, h.logger, userID, project.OrganizationID)
	if err != nil {
		return err
	}

	// Get all inspections for the project
	inspections, err := queries.ListInspections(ctx, projectUUID)
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
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

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
	project, err := queries.GetProject(ctx, projectUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "project not found")
		}
		h.logger.Error("failed to get project", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load project")
	}

	// Verify user is a member of the organization
	_, err = requireOrganizationMembership(ctx, h.pool, h.logger, userID, project.OrganizationID)
	if err != nil {
		return err
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
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	// Get user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
	}

	queries := database.New(h.pool)

	// Get user's organizations
	memberships, err := queries.ListUserOrganizations(ctx, uuidToPgUUID(userID))
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
		projects, err := queries.ListProjects(ctx, membership.OrganizationID)
		if err != nil {
			h.logger.Warn("failed to list projects", slog.String("err", err.Error()))
			continue
		}

		// Get inspections for each project
		for _, project := range projects {
			inspections, err := queries.ListInspections(ctx, project.ID)
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

// InspectionDetailPage displays a single inspection with its photos
func (h *PageHandler) InspectionDetailPage(c echo.Context) error {
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	// Get authenticated user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		return c.Redirect(http.StatusSeeOther, "/login")
	}

	inspectionID := c.Param("id")
	if inspectionID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "inspection id is required")
	}

	queries := database.New(h.pool)

	// Parse inspection ID
	inspectionUUID, err := parseUUID(inspectionID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid inspection id")
	}

	// Get inspection
	inspection, err := queries.GetInspection(ctx, inspectionUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "inspection not found")
		}
		h.logger.Error("failed to get inspection", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load inspection")
	}

	// Get project for inspection
	project, err := queries.GetProject(ctx, inspection.ProjectID)
	if err != nil {
		h.logger.Error("failed to get project", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load inspection details")
	}

	// Verify user is a member of the organization
	_, err = requireOrganizationMembership(ctx, h.pool, h.logger, userID, project.OrganizationID)
	if err != nil {
		return err
	}

	// Get photos for inspection
	photos, err := queries.ListPhotos(ctx, inspectionUUID)
	if err != nil {
		h.logger.Error("failed to list photos", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load photos")
	}

	// Get violations for the inspection
	violations, err := queries.ListDetectedViolationsByInspection(ctx, inspectionUUID)
	if err != nil {
		h.logger.Error("failed to list violations", slog.String("err", err.Error()))
		// Don't fail the page load if violations can't be fetched, just log it
		violations = []database.DetectedViolation{}
	}

	// Create a map of photo ID -> violations for easy lookup in template
	// Filter out dismissed violations (soft delete - hidden from UI but preserved until next analysis)
	violationsByPhoto := make(map[string][]database.DetectedViolation)
	for _, violation := range violations {
		// Skip dismissed violations - they're hidden from the UI
		if violation.Status == database.ViolationStatusDismissed {
			continue
		}
		photoIDStr := violation.PhotoID.String()
		violationsByPhoto[photoIDStr] = append(violationsByPhoto[photoIDStr], violation)
	}

	// Create a map of safety code ID -> safety code info
	safetyCodeMap := make(map[string]string)
	for _, violation := range violations {
		if violation.SafetyCodeID.Valid {
			// Only fetch if we haven't already
			if _, exists := safetyCodeMap[violation.SafetyCodeID.String()]; !exists {
				safetyCode, err := queries.GetSafetyCode(ctx, violation.SafetyCodeID)
				if err == nil {
					safetyCodeMap[violation.SafetyCodeID.String()] = safetyCode.Code + " - " + safetyCode.Description
				}
			}
		}
	}

	// Build project location string
	var projectLocation string
	if project.Address.Valid {
		projectLocation = project.Address.String
		if project.City.Valid {
			projectLocation += ", " + project.City.String
		}
		if project.State.Valid {
			projectLocation += ", " + project.State.String
		}
	}

	// Build photo cards data with violation counts
	photoCards := make([]map[string]interface{}, 0, len(photos))
	for _, photo := range photos {
		photoIDStr := photo.ID.String()
		photoViolations := violationsByPhoto[photoIDStr]

		// Count violations by status
		confirmedCount := 0
		pendingCount := 0
		for _, v := range photoViolations {
			if v.Status == database.ViolationStatusConfirmed {
				confirmedCount++
			} else if v.Status == database.ViolationStatusPending {
				pendingCount++
			}
		}

		// Determine analysis status based on violation counts
		analysisStatus := "uploaded"
		if len(photoViolations) > 0 {
			analysisStatus = "analyzed"
		}

		// Get first 2 violations for preview
		violationsPreview := photoViolations
		if len(violationsPreview) > 2 {
			violationsPreview = violationsPreview[:2]
		}

		// Build thumbnail URL
		thumbnailURL := photo.StorageUrl
		if photo.ThumbnailUrl.Valid {
			thumbnailURL = photo.ThumbnailUrl.String
		}

		photoCards = append(photoCards, map[string]interface{}{
			"PhotoID":           photoIDStr,
			"ThumbnailURL":      thumbnailURL,
			"AnalysisStatus":    analysisStatus,
			"ConfirmedCount":    confirmedCount,
			"PendingCount":      pendingCount,
			"TotalCount":        len(photoViolations),
			"ViolationsPreview": violationsPreview,
		})
	}

	// Count total violations by status
	confirmedTotal := 0
	pendingTotal := 0
	for _, violation := range violations {
		if violation.Status == database.ViolationStatusConfirmed {
			confirmedTotal++
		} else if violation.Status == database.ViolationStatusPending {
			pendingTotal++
		}
	}

	data := map[string]interface{}{
		"IsAuthenticated":   true,
		"User":              h.getUserDisplayInfo(c, userID),
		"Inspection":        inspection,
		"ProjectID":         project.ID.String(),
		"ProjectName":       project.Name,
		"ProjectLocation":   projectLocation,
		"Photos":            photoCards,
		"ConfirmedTotal":    confirmedTotal,
		"PendingTotal":      pendingTotal,
		"PhotoCount":        len(photos),
		"ViolationsByPhoto": violationsByPhoto,
		"SafetyCodeMap":     safetyCodeMap,
	}
	return c.Render(http.StatusOK, "inspection-detail.html", data)
}

// PhotoDetailPage renders the photo detail page with violation review
func (h *PageHandler) PhotoDetailPage(c echo.Context) error {
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	// Get authenticated user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		return c.Redirect(http.StatusSeeOther, "/login")
	}

	photoID := c.Param("id")
	if photoID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "photo id is required")
	}

	queries := database.New(h.pool)

	// Parse photo ID
	photoUUID, err := parseUUID(photoID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid photo id")
	}

	// Get photo
	photo, err := queries.GetPhoto(ctx, photoUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "photo not found")
		}
		h.logger.Error("failed to get photo", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load photo")
	}

	// Get inspection
	inspection, err := queries.GetInspection(ctx, photo.InspectionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "inspection not found")
		}
		h.logger.Error("failed to get inspection", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load photo details")
	}

	// Get project
	project, err := queries.GetProject(ctx, inspection.ProjectID)
	if err != nil {
		h.logger.Error("failed to get project", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load photo details")
	}

	// Verify user is a member of the organization
	_, err = requireOrganizationMembership(ctx, h.pool, h.logger, userID, project.OrganizationID)
	if err != nil {
		return err
	}

	// Get violations for this photo
	allViolations, err := queries.ListDetectedViolations(ctx, photo.ID)
	if err != nil {
		h.logger.Error("failed to list violations", slog.String("err", err.Error()))
		allViolations = []database.DetectedViolation{}
	}

	// Filter out dismissed violations (soft delete - hidden from UI but preserved until next analysis)
	violations := make([]database.DetectedViolation, 0)
	for _, violation := range allViolations {
		if violation.Status != database.ViolationStatusDismissed {
			violations = append(violations, violation)
		}
	}

	// Get safety codes for violations
	safetyCodeMap := make(map[string]string)
	for _, violation := range violations {
		if violation.SafetyCodeID.Valid {
			safetyCode, err := queries.GetSafetyCode(ctx, violation.SafetyCodeID)
			if err == nil {
				safetyCodeMap[violation.SafetyCodeID.String()] = safetyCode.Code + " - " + safetyCode.Description
			}
		}
	}

	data := map[string]interface{}{
		"IsAuthenticated": true,
		"User":            h.getUserDisplayInfo(c, userID),
		"Photo":           photo,
		"Inspection":      inspection,
		"ProjectName":     project.Name,
		"Violations":      violations,
		"SafetyCodeMap":   safetyCodeMap,
	}
	return c.Render(http.StatusOK, "photo-detail.html", data)
}
