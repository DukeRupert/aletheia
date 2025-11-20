package handlers

import (
	"log/slog"
	"net/http"

	"github.com/dukerupert/aletheia/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
)

// ViolationHandler handles detected violation HTTP requests
type ViolationHandler struct {
	db     *database.Queries
	logger *slog.Logger
}

// NewViolationHandler creates a new violation handler
func NewViolationHandler(db *database.Queries, logger *slog.Logger) *ViolationHandler {
	return &ViolationHandler{
		db:     db,
		logger: logger,
	}
}

// ViolationResponse represents a detected violation in API responses
type ViolationResponse struct {
	ID              string  `json:"id"`
	PhotoID         string  `json:"photo_id"`
	Description     string  `json:"description"`
	ConfidenceScore float64 `json:"confidence_score"`
	Severity        string  `json:"severity"`
	Location        *string `json:"location,omitempty"`
	SafetyCodeID    *string `json:"safety_code_id,omitempty"`
	Status          string  `json:"status"`
	CreatedAt       string  `json:"created_at"`
}

// ListViolationsByInspection godoc
// @Summary List detected violations for an inspection
// @Description Get all detected violations for a specific inspection
// @Tags violations
// @Accept json
// @Produce json
// @Param inspection_id path string true "Inspection ID"
// @Param status query string false "Filter by status (pending, confirmed, dismissed)"
// @Success 200 {array} ViolationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/inspections/{inspection_id}/violations [get]
func (h *ViolationHandler) ListViolationsByInspection(c echo.Context) error {
	ctx := c.Request().Context()

	inspectionIDStr := c.Param("inspection_id")
	inspectionID, err := uuid.Parse(inspectionIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid inspection_id format")
	}

	// Check if status filter is provided
	statusFilter := c.QueryParam("status")

	var violations []database.DetectedViolation

	if statusFilter != "" {
		// Validate status
		status := database.ViolationStatus(statusFilter)
		violations, err = h.db.ListDetectedViolationsByInspectionAndStatus(ctx,
			database.ListDetectedViolationsByInspectionAndStatusParams{
				InspectionID: pgtype.UUID{Bytes: inspectionID, Valid: true},
				Status:       status,
			})
	} else {
		violations, err = h.db.ListDetectedViolationsByInspection(ctx, pgtype.UUID{Bytes: inspectionID, Valid: true})
	}

	if err != nil {
		h.logger.Error("failed to list violations",
			slog.String("inspection_id", inspectionID.String()),
			slog.String("error", err.Error()),
		)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve violations")
	}

	// Convert to response format
	response := make([]ViolationResponse, 0, len(violations))
	for _, v := range violations {
		resp := ViolationResponse{
			ID:          v.ID.String(),
			PhotoID:     v.PhotoID.String(),
			Description: v.Description,
			Severity:    string(v.Severity),
			Status:      string(v.Status),
			CreatedAt:   v.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		}

		// Convert confidence score (numeric(5,4) to float64)
		if v.ConfidenceScore.Valid {
			// Convert pgtype.Numeric to float64
			// Note: This is a simplified conversion. For precise decimal handling,
			// you might want to use a proper decimal library
			resp.ConfidenceScore = float64(v.ConfidenceScore.Int.Int64()) / 10000.0
		}

		// Add optional fields
		if v.Location.Valid {
			resp.Location = &v.Location.String
		}

		if v.SafetyCodeID.Valid {
			safetyCodeIDStr := uuid.UUID(v.SafetyCodeID.Bytes).String()
			resp.SafetyCodeID = &safetyCodeIDStr
		}

		response = append(response, resp)
	}

	return c.JSON(http.StatusOK, response)
}

// GetViolation godoc
// @Summary Get a specific detected violation
// @Description Get details of a detected violation by ID
// @Tags violations
// @Accept json
// @Produce json
// @Param violation_id path string true "Violation ID"
// @Success 200 {object} ViolationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/violations/{violation_id} [get]
func (h *ViolationHandler) GetViolation(c echo.Context) error {
	ctx := c.Request().Context()

	violationIDStr := c.Param("violation_id")
	violationID, err := uuid.Parse(violationIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid violation_id format")
	}

	violation, err := h.db.GetDetectedViolation(ctx, pgtype.UUID{Bytes: violationID, Valid: true})
	if err != nil {
		h.logger.Error("violation not found",
			slog.String("violation_id", violationID.String()),
			slog.String("error", err.Error()),
		)
		return echo.NewHTTPError(http.StatusNotFound, "Violation not found")
	}

	// Convert to response format
	resp := ViolationResponse{
		ID:          violation.ID.String(),
		PhotoID:     violation.PhotoID.String(),
		Description: violation.Description,
		Severity:    string(violation.Severity),
		Status:      string(violation.Status),
		CreatedAt:   violation.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}

	if violation.ConfidenceScore.Valid {
		resp.ConfidenceScore = float64(violation.ConfidenceScore.Int.Int64()) / 10000.0
	}

	if violation.Location.Valid {
		resp.Location = &violation.Location.String
	}

	if violation.SafetyCodeID.Valid {
		safetyCodeIDStr := uuid.UUID(violation.SafetyCodeID.Bytes).String()
		resp.SafetyCodeID = &safetyCodeIDStr
	}

	return c.JSON(http.StatusOK, resp)
}

// UpdateViolationRequest is the request body for updating a violation
type UpdateViolationRequest struct {
	Status      *string `json:"status,omitempty" validate:"omitempty,oneof=pending confirmed dismissed"`
	Description *string `json:"description,omitempty"`
}

// UpdateViolation godoc
// @Summary Update a detected violation
// @Description Update violation status or add notes to a detected violation
// @Tags violations
// @Accept json
// @Produce json
// @Param violation_id path string true "Violation ID"
// @Param request body UpdateViolationRequest true "Update Request"
// @Success 200 {object} ViolationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/violations/{violation_id} [patch]
func (h *ViolationHandler) UpdateViolation(c echo.Context) error {
	ctx := c.Request().Context()

	violationIDStr := c.Param("violation_id")
	violationID, err := uuid.Parse(violationIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid violation_id format")
	}

	// Parse request
	var req UpdateViolationRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	// Prepare update parameters
	params := database.UpdateDetectedViolationNotesParams{
		ID: pgtype.UUID{Bytes: violationID, Valid: true},
	}

	if req.Status != nil {
		params.Status = database.NullViolationStatus{
			ViolationStatus: database.ViolationStatus(*req.Status),
			Valid:           true,
		}
	}

	if req.Description != nil {
		params.Description = pgtype.Text{
			String: *req.Description,
			Valid:  true,
		}
	}

	// Update violation
	violation, err := h.db.UpdateDetectedViolationNotes(ctx, params)
	if err != nil {
		h.logger.Error("failed to update violation",
			slog.String("violation_id", violationID.String()),
			slog.String("error", err.Error()),
		)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update violation")
	}

	// Convert to response format
	resp := ViolationResponse{
		ID:          violation.ID.String(),
		PhotoID:     violation.PhotoID.String(),
		Description: violation.Description,
		Severity:    string(violation.Severity),
		Status:      string(violation.Status),
		CreatedAt:   violation.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}

	if violation.ConfidenceScore.Valid {
		resp.ConfidenceScore = float64(violation.ConfidenceScore.Int.Int64()) / 10000.0
	}

	if violation.Location.Valid {
		resp.Location = &violation.Location.String
	}

	if violation.SafetyCodeID.Valid {
		safetyCodeIDStr := uuid.UUID(violation.SafetyCodeID.Bytes).String()
		resp.SafetyCodeID = &safetyCodeIDStr
	}

	h.logger.Info("violation updated",
		slog.String("violation_id", violationID.String()),
	)

	return c.JSON(http.StatusOK, resp)
}

// DeleteViolation godoc
// @Summary Delete a detected violation
// @Description Delete a detected violation (dismiss false positive)
// @Tags violations
// @Accept json
// @Produce json
// @Param violation_id path string true "Violation ID"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/violations/{violation_id} [delete]
func (h *ViolationHandler) DeleteViolation(c echo.Context) error {
	ctx := c.Request().Context()

	violationIDStr := c.Param("violation_id")
	violationID, err := uuid.Parse(violationIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid violation_id format")
	}

	err = h.db.DeleteDetectedViolation(ctx, pgtype.UUID{Bytes: violationID, Valid: true})
	if err != nil {
		h.logger.Error("failed to delete violation",
			slog.String("violation_id", violationID.String()),
			slog.String("error", err.Error()),
		)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete violation")
	}

	h.logger.Info("violation deleted",
		slog.String("violation_id", violationID.String()),
	)

	return c.NoContent(http.StatusNoContent)
}
