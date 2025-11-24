package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/dukerupert/aletheia/internal/database"
	"github.com/dukerupert/aletheia/internal/session"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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
	OrganizationID string `json:"organization_id" form:"organization_id" validate:"required"`
	Name           string `json:"name" form:"name" validate:"required,min=1,max=255"`
	Description    string `json:"description" form:"description"`
	ProjectType    string `json:"project_type" form:"project_type"`
	Address        string `json:"address" form:"address"`
	City           string `json:"city" form:"city"`
	State          string `json:"state" form:"state" validate:"omitempty,len=2"`
	ZipCode        string `json:"zip_code" form:"zip_code"`
	Country        string `json:"country" form:"country" validate:"omitempty,len=2"`
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

	if err := c.Validate(&req); err != nil {
		return err
	}

	// Sanitize input
	req.OrganizationID = strings.TrimSpace(req.OrganizationID)
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	req.ProjectType = strings.TrimSpace(req.ProjectType)
	req.Address = strings.TrimSpace(req.Address)
	req.City = strings.TrimSpace(req.City)
	req.State = strings.TrimSpace(req.State)
	req.ZipCode = strings.TrimSpace(req.ZipCode)
	req.Country = strings.TrimSpace(req.Country)

	queries := database.New(h.pool)

	// Parse organization ID
	orgUUID, err := parseUUID(req.OrganizationID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid organization_id")
	}

	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	// Verify user is owner or admin of the organization
	_, err = requireOrganizationMembership(ctx, h.pool, h.logger, userID, orgUUID, database.OrganizationRoleOwner, database.OrganizationRoleAdmin)
	if err != nil {
		return err
	}

	// Set default country if not provided
	country := req.Country
	if country == "" {
		country = "US"
	}

	// Create project
	project, err := queries.CreateProject(ctx, database.CreateProjectParams{
		OrganizationID: orgUUID,
		Name:           req.Name,
		Description:    pgtype.Text{String: req.Description, Valid: req.Description != ""},
		ProjectType:    pgtype.Text{String: req.ProjectType, Valid: req.ProjectType != ""},
		Address:        pgtype.Text{String: req.Address, Valid: req.Address != ""},
		City:           pgtype.Text{String: req.City, Valid: req.City != ""},
		State:          pgtype.Text{String: req.State, Valid: req.State != ""},
		ZipCode:        pgtype.Text{String: req.ZipCode, Valid: req.ZipCode != ""},
		Country:        pgtype.Text{String: country, Valid: true},
	})
	if err != nil {
		h.logger.Error("failed to create project", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create project")
	}

	h.logger.Info("project created",
		slog.String("project_id", project.ID.String()),
		slog.String("org_id", req.OrganizationID),
		slog.String("user_id", userID.String()))

	// Check if this is an HTMX request
	if c.Request().Header.Get("HX-Request") == "true" {
		// HTMX request - redirect to projects list
		c.Response().Header().Set("HX-Redirect", "/projects")
		return c.NoContent(http.StatusOK)
	}

	// Regular API request - return JSON
	return c.JSON(http.StatusCreated, CreateProjectResponse{
		ID:             project.ID.String(),
		OrganizationID: project.OrganizationID.String(),
		Name:           project.Name,
		CreatedAt:      project.CreatedAt.Time.Format(RFC3339Format),
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

	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	// Get project
	project, err := queries.GetProject(ctx, projectUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "project not found")
		}
		h.logger.Error("failed to get project", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get project")
	}

	// Verify user is a member of the organization that owns this project
	_, err = requireOrganizationMembership(ctx, h.pool, h.logger, userID, project.OrganizationID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, GetProjectResponse{
		ID:             project.ID.String(),
		OrganizationID: project.OrganizationID.String(),
		Name:           project.Name,
		CreatedAt:      project.CreatedAt.Time.Format(RFC3339Format),
		UpdatedAt:      project.UpdatedAt.Time.Format(RFC3339Format),
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

	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	// Verify user is a member of the organization
	_, err = requireOrganizationMembership(ctx, h.pool, h.logger, userID, orgUUID)
	if err != nil {
		return err
	}

	// Get all projects for the organization
	projects, err := queries.ListProjects(ctx, orgUUID)
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
			CreatedAt:      project.CreatedAt.Time.Format(RFC3339Format),
		}
	}

	return c.JSON(http.StatusOK, ListProjectsResponse{
		Projects: projectSummaries,
	})
}

// UpdateProjectRequest is the request payload for updating a project
type UpdateProjectRequest struct {
	Name        *string `json:"name,omitempty" form:"name" validate:"omitempty,min=1,max=255"`
	Description *string `json:"description,omitempty" form:"description"`
	ProjectType *string `json:"project_type,omitempty" form:"project_type"`
	Status      *string `json:"status,omitempty" form:"status"`
	Address     *string `json:"address,omitempty" form:"address"`
	City        *string `json:"city,omitempty" form:"city"`
	State       *string `json:"state,omitempty" form:"state" validate:"omitempty,len=2"`
	ZipCode     *string `json:"zip_code,omitempty" form:"zip_code"`
	Country     *string `json:"country,omitempty" form:"country" validate:"omitempty,len=2"`
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

	if err := c.Validate(&req); err != nil {
		return err
	}

	// Sanitize input
	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		req.Name = &trimmed
	}
	if req.Description != nil {
		trimmed := strings.TrimSpace(*req.Description)
		req.Description = &trimmed
	}
	if req.ProjectType != nil {
		trimmed := strings.TrimSpace(*req.ProjectType)
		req.ProjectType = &trimmed
	}
	if req.Status != nil {
		trimmed := strings.TrimSpace(*req.Status)
		req.Status = &trimmed
	}
	if req.Address != nil {
		trimmed := strings.TrimSpace(*req.Address)
		req.Address = &trimmed
	}
	if req.City != nil {
		trimmed := strings.TrimSpace(*req.City)
		req.City = &trimmed
	}
	if req.State != nil {
		trimmed := strings.TrimSpace(*req.State)
		req.State = &trimmed
	}
	if req.ZipCode != nil {
		trimmed := strings.TrimSpace(*req.ZipCode)
		req.ZipCode = &trimmed
	}
	if req.Country != nil {
		trimmed := strings.TrimSpace(*req.Country)
		req.Country = &trimmed
	}

	queries := database.New(h.pool)

	// Parse project ID
	projectUUID, err := parseUUID(projectID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid project id")
	}

	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	// Get project to find its organization
	project, err := queries.GetProject(ctx, projectUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "project not found")
		}
		h.logger.Error("failed to get project", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get project")
	}

	// Verify user is owner or admin of the organization
	_, err = requireOrganizationMembership(ctx, h.pool, h.logger, userID, project.OrganizationID, database.OrganizationRoleOwner, database.OrganizationRoleAdmin)
	if err != nil {
		return err
	}

	// Update project
	params := database.UpdateProjectParams{
		ID: projectUUID,
	}

	if req.Name != nil {
		params.Name = pgtype.Text{String: *req.Name, Valid: true}
	}
	if req.Description != nil {
		params.Description = pgtype.Text{String: *req.Description, Valid: true}
	}
	if req.ProjectType != nil {
		params.ProjectType = pgtype.Text{String: *req.ProjectType, Valid: true}
	}
	if req.Status != nil {
		params.Status = pgtype.Text{String: *req.Status, Valid: true}
	}
	if req.Address != nil {
		params.Address = pgtype.Text{String: *req.Address, Valid: true}
	}
	if req.City != nil {
		params.City = pgtype.Text{String: *req.City, Valid: true}
	}
	if req.State != nil {
		params.State = pgtype.Text{String: *req.State, Valid: true}
	}
	if req.ZipCode != nil {
		params.ZipCode = pgtype.Text{String: *req.ZipCode, Valid: true}
	}
	if req.Country != nil {
		params.Country = pgtype.Text{String: *req.Country, Valid: true}
	}

	updatedProject, err := queries.UpdateProject(ctx, params)
	if err != nil {
		h.logger.Error("failed to update project", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update project")
	}

	h.logger.Info("project updated", slog.String("project_id", projectID))

	return c.JSON(http.StatusOK, UpdateProjectResponse{
		ID:        updatedProject.ID.String(),
		Name:      updatedProject.Name,
		UpdatedAt: updatedProject.UpdatedAt.Time.Format(RFC3339Format),
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

	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	// Get project to find its organization
	project, err := queries.GetProject(ctx, projectUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "project not found")
		}
		h.logger.Error("failed to get project", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get project")
	}

	// Verify user is owner or admin of the organization
	_, err = requireOrganizationMembership(ctx, h.pool, h.logger, userID, project.OrganizationID, database.OrganizationRoleOwner, database.OrganizationRoleAdmin)
	if err != nil {
		return err
	}

	// Delete project
	err = queries.DeleteProject(ctx, projectUUID)
	if err != nil {
		h.logger.Error("failed to delete project", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete project")
	}

	h.logger.Info("project deleted", slog.String("project_id", projectID))

	return c.NoContent(http.StatusNoContent)
}
