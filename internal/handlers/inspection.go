package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/dukerupert/aletheia/internal/database"
	"github.com/dukerupert/aletheia/internal/session"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

type InspectionHandler struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewInspectionHandler(pool *pgxpool.Pool, logger *slog.Logger) *InspectionHandler {
	return &InspectionHandler{
		pool:   pool,
		logger: logger,
	}
}

// CreateInspectionRequest is the request payload for creating an inspection
type CreateInspectionRequest struct {
	ProjectID string `json:"project_id" form:"project_id" validate:"required"`
}

// CreateInspectionResponse is the response payload for inspection creation
type CreateInspectionResponse struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	InspectorID string `json:"inspector_id"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

// CreateInspection creates a new inspection for a project
func (h *InspectionHandler) CreateInspection(c echo.Context) error {
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	// Get authenticated user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session", slog.String("endpoint", "CreateInspection"))
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	// Parse request
	var req CreateInspectionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := c.Validate(&req); err != nil {
		return err
	}

	queries := database.New(h.pool)

	// Parse project ID
	projectUUID, err := parseUUID(req.ProjectID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid project_id")
	}

	// Get project to find its organization
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

	// Create inspection with default status 'draft'
	inspection, err := queries.CreateInspection(ctx, database.CreateInspectionParams{
		ProjectID:    projectUUID,
		InspectorID:  uuidToPgUUID(userID),
		Status:       database.InspectionStatusDraft,
	})
	if err != nil {
		h.logger.Error("failed to create inspection", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create inspection")
	}

	h.logger.Info("inspection created",
		slog.String("inspection_id", inspection.ID.String()),
		slog.String("project_id", req.ProjectID),
		slog.String("inspector_id", userID.String()))

	// Check if this is an HTMX request
	if c.Request().Header.Get("HX-Request") == "true" {
		// HTMX request - redirect to inspection detail
		c.Response().Header().Set("HX-Redirect", "/inspections/"+inspection.ID.String())
		return c.NoContent(http.StatusOK)
	}

	// Regular API request - return JSON
	return c.JSON(http.StatusCreated, CreateInspectionResponse{
		ID:          inspection.ID.String(),
		ProjectID:   inspection.ProjectID.String(),
		InspectorID: inspection.InspectorID.String(),
		Status:      string(inspection.Status),
		CreatedAt:   inspection.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// GetInspectionResponse is the response payload for inspection retrieval
type GetInspectionResponse struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	InspectorID string `json:"inspector_id"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// GetInspection retrieves an inspection by ID
func (h *InspectionHandler) GetInspection(c echo.Context) error {
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	// Get authenticated user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session", slog.String("endpoint", "GetInspection"))
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
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
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get inspection")
	}

	// Get project to find its organization
	project, err := queries.GetProject(ctx, inspection.ProjectID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "project not found")
		}
		h.logger.Error("failed to get project for inspection", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get inspection details")
	}

	// Verify user is a member of the organization that owns this project
	_, err = requireOrganizationMembership(ctx, h.pool, h.logger, userID, project.OrganizationID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, GetInspectionResponse{
		ID:          inspection.ID.String(),
		ProjectID:   inspection.ProjectID.String(),
		InspectorID: inspection.InspectorID.String(),
		Status:      string(inspection.Status),
		CreatedAt:   inspection.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   inspection.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// ListInspectionsResponse is the response payload for listing inspections
type InspectionSummary struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	InspectorID string `json:"inspector_id"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

type ListInspectionsResponse struct {
	Inspections []InspectionSummary `json:"inspections"`
}

// ListInspections lists all inspections for a project
func (h *InspectionHandler) ListInspections(c echo.Context) error {
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	// Get authenticated user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session", slog.String("endpoint", "ListInspections"))
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
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

	// Get project to find its organization
	project, err := queries.GetProject(ctx, projectUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "project not found")
		}
		h.logger.Error("failed to get project", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get project")
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
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list inspections")
	}

	inspectionSummaries := make([]InspectionSummary, len(inspections))
	for i, inspection := range inspections {
		inspectionSummaries[i] = InspectionSummary{
			ID:          inspection.ID.String(),
			ProjectID:   inspection.ProjectID.String(),
			InspectorID: inspection.InspectorID.String(),
			Status:      string(inspection.Status),
			CreatedAt:   inspection.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return c.JSON(http.StatusOK, ListInspectionsResponse{
		Inspections: inspectionSummaries,
	})
}

// UpdateInspectionStatusRequest is the request payload for updating inspection status
type UpdateInspectionStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=draft in_progress completed"`
}

// UpdateInspectionStatusResponse is the response payload for status update
type UpdateInspectionStatusResponse struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updated_at"`
}

// UpdateInspectionStatus updates an inspection's status
func (h *InspectionHandler) UpdateInspectionStatus(c echo.Context) error {
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	// Get authenticated user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session", slog.String("endpoint", "UpdateInspectionStatus"))
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	inspectionID := c.Param("id")
	if inspectionID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "inspection id is required")
	}

	// Parse request
	var req UpdateInspectionStatusRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := c.Validate(&req); err != nil {
		return err
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
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get inspection")
	}

	// Get project to find its organization
	project, err := queries.GetProject(ctx, inspection.ProjectID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "project not found")
		}
		h.logger.Error("failed to get project for inspection", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get inspection details")
	}

	// Verify user is a member of the organization and get membership for role checking
	membership, err := requireOrganizationMembership(ctx, h.pool, h.logger, userID, project.OrganizationID)
	if err != nil {
		return err
	}

	// Check if user is the inspector or an owner/admin
	isInspector := inspection.InspectorID.Bytes == userID
	isOwnerOrAdmin := membership.Role == database.OrganizationRoleOwner || membership.Role == database.OrganizationRoleAdmin

	if !isInspector && !isOwnerOrAdmin {
		return echo.NewHTTPError(http.StatusForbidden, "only the inspector or organization owners/admins can update inspection status")
	}

	// Parse and validate new status
	var newStatus database.InspectionStatus
	switch req.Status {
	case "draft":
		newStatus = database.InspectionStatusDraft
	case "in_progress":
		newStatus = database.InspectionStatusInProgress
	case "completed":
		newStatus = database.InspectionStatusCompleted
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "invalid status")
	}

	// Update inspection status
	updatedInspection, err := queries.UpdateInspectionStatus(ctx, database.UpdateInspectionStatusParams{
		ID:     inspectionUUID,
		Status: newStatus,
	})
	if err != nil {
		h.logger.Error("failed to update inspection status", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update inspection status")
	}

	h.logger.Info("inspection status updated",
		slog.String("inspection_id", inspectionID),
		slog.String("new_status", req.Status))

	return c.JSON(http.StatusOK, UpdateInspectionStatusResponse{
		ID:        updatedInspection.ID.String(),
		Status:    string(updatedInspection.Status),
		UpdatedAt: updatedInspection.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	})
}
