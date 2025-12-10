package http

import (
	"log/slog"
	"net/http"

	"github.com/dukerupert/aletheia"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// CreateProjectRequest is the request payload for creating a project.
type CreateProjectRequest struct {
	OrganizationID string `json:"organization_id" form:"organization_id" validate:"required,uuid"`
	Name           string `json:"name" form:"name" validate:"required,min=2,max=100"`
	Description    string `json:"description" form:"description" validate:"omitempty,max=500"`
	ProjectType    string `json:"project_type" form:"project_type" validate:"omitempty,max=50"`
	Address        string `json:"address" form:"address" validate:"omitempty,max=200"`
	City           string `json:"city" form:"city" validate:"omitempty,max=100"`
	State          string `json:"state" form:"state" validate:"omitempty,max=100"`
	ZipCode        string `json:"zip_code" form:"zip_code" validate:"omitempty,max=20"`
	Country        string `json:"country" form:"country" validate:"omitempty,max=100"`
}

func (s *Server) handleCreateProject(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	var req CreateProjectRequest
	if err := bind(c, &req); err != nil {
		return err
	}

	orgID, err := parseUUID(req.OrganizationID)
	if err != nil {
		return err
	}

	project := &aletheia.Project{
		OrganizationID: orgID,
		Name:           req.Name,
		Description:    req.Description,
		ProjectType:    req.ProjectType,
		Status:         "active",
		Address:        req.Address,
		City:           req.City,
		State:          req.State,
		ZipCode:        req.ZipCode,
		Country:        req.Country,
	}

	if err := s.projectService.CreateProject(ctx, project); err != nil {
		return err
	}

	s.log(c).Info("project created",
		slog.String("project_id", project.ID.String()),
		slog.String("org_id", orgID.String()),
	)

	return RespondCreated(c, project)
}

func (s *Server) handleGetProject(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	projectID, err := requireUUIDParam(c, "id")
	if err != nil {
		return err
	}

	project, err := s.projectService.FindProjectByID(ctx, projectID)
	if err != nil {
		return err
	}

	return RespondOK(c, project)
}

func (s *Server) handleListProjects(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	orgID, err := requireUUIDParam(c, "orgId")
	if err != nil {
		return err
	}

	// Parse optional query parameters
	status := c.QueryParam("status")
	search := c.QueryParam("search")

	filter := aletheia.ProjectFilter{
		OrganizationID: &orgID,
		Limit:          100,
	}

	if status != "" {
		filter.Status = &status
	}
	if search != "" {
		filter.Search = &search
	}

	projects, total, err := s.projectService.FindProjects(ctx, filter)
	if err != nil {
		return err
	}

	return RespondOK(c, map[string]interface{}{
		"projects": projects,
		"total":    total,
	})
}

// UpdateProjectRequest is the request payload for updating a project.
type UpdateProjectRequest struct {
	Name        *string `json:"name" form:"name" validate:"omitempty,min=2,max=100"`
	Description *string `json:"description" form:"description" validate:"omitempty,max=500"`
	ProjectType *string `json:"project_type" form:"project_type" validate:"omitempty,max=50"`
	Status      *string `json:"status" form:"status" validate:"omitempty,oneof=active completed archived"`
	Address     *string `json:"address" form:"address" validate:"omitempty,max=200"`
	City        *string `json:"city" form:"city" validate:"omitempty,max=100"`
	State       *string `json:"state" form:"state" validate:"omitempty,max=100"`
	ZipCode     *string `json:"zip_code" form:"zip_code" validate:"omitempty,max=20"`
	Country     *string `json:"country" form:"country" validate:"omitempty,max=100"`
}

func (s *Server) handleUpdateProject(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	projectID, err := requireUUIDParam(c, "id")
	if err != nil {
		return err
	}

	var req UpdateProjectRequest
	if err := bind(c, &req); err != nil {
		return err
	}

	project, err := s.projectService.UpdateProject(ctx, projectID, aletheia.ProjectUpdate{
		Name:        req.Name,
		Description: req.Description,
		ProjectType: req.ProjectType,
		Status:      req.Status,
		Address:     req.Address,
		City:        req.City,
		State:       req.State,
		ZipCode:     req.ZipCode,
		Country:     req.Country,
	})
	if err != nil {
		return err
	}

	s.log(c).Info("project updated", slog.String("project_id", project.ID.String()))

	return RespondOK(c, project)
}

func (s *Server) handleDeleteProject(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	projectID, err := requireUUIDParam(c, "id")
	if err != nil {
		return err
	}

	if err := s.projectService.DeleteProject(ctx, projectID); err != nil {
		return err
	}

	s.log(c).Info("project deleted", slog.String("project_id", projectID.String()))

	return c.NoContent(http.StatusNoContent)
}

// Helper to get project and verify organization access
func (s *Server) getProjectWithOrgCheck(c echo.Context, projectID uuid.UUID) (*aletheia.Project, error) {
	ctx := c.Request().Context()

	project, err := s.projectService.FindProjectByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Verify user has access to the project's organization
	user, err := requireUser(c)
	if err != nil {
		return nil, err
	}

	_, err = s.organizationService.RequireMembership(ctx, project.OrganizationID, user.ID)
	if err != nil {
		return nil, err
	}

	return project, nil
}
