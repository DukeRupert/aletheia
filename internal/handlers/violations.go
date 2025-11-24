package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"

	"github.com/dukerupert/aletheia/internal/database"
	"github.com/dukerupert/aletheia/internal/session"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

// ViolationHandler handles detected violation HTTP requests
type ViolationHandler struct {
	db     *database.Queries
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewViolationHandler creates a new violation handler
func NewViolationHandler(pool *pgxpool.Pool, db *database.Queries, logger *slog.Logger) *ViolationHandler {
	return &ViolationHandler{
		db:     db,
		pool:   pool,
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
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	inspectionIDStr := c.Param("inspection_id")
	inspectionID, err := uuid.Parse(inspectionIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid inspection_id format")
	}

	// Authorization: verify user has access to this inspection's organization
	userID, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	orgID, err := getOrganizationIDFromInspection(ctx, h.db, pgtype.UUID{Bytes: inspectionID, Valid: true})
	if err != nil {
		return err
	}

	_, err = requireOrganizationMembership(ctx, h.pool, h.logger, userID, orgID)
	if err != nil {
		return err
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
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

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

	// Authorization: verify user has access to this violation's organization
	userID, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	orgID, err := getOrganizationIDFromPhoto(ctx, h.db, violation.PhotoID)
	if err != nil {
		return err
	}

	_, err = requireOrganizationMembership(ctx, h.pool, h.logger, userID, orgID)
	if err != nil {
		return err
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
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	violationIDStr := c.Param("violation_id")
	violationID, err := uuid.Parse(violationIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid violation_id format")
	}

	// Get violation first to check authorization
	violation, err := h.db.GetDetectedViolation(ctx, pgtype.UUID{Bytes: violationID, Valid: true})
	if err != nil {
		h.logger.Error("violation not found",
			slog.String("violation_id", violationID.String()),
			slog.String("error", err.Error()),
		)
		return echo.NewHTTPError(http.StatusNotFound, "Violation not found")
	}

	// Authorization: verify user has access to this violation's organization
	userID, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	orgID, err := getOrganizationIDFromPhoto(ctx, h.db, violation.PhotoID)
	if err != nil {
		return err
	}

	_, err = requireOrganizationMembership(ctx, h.pool, h.logger, userID, orgID)
	if err != nil {
		return err
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
	violation, err = h.db.UpdateDetectedViolationNotes(ctx, params)
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
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	violationIDStr := c.Param("violation_id")
	violationID, err := uuid.Parse(violationIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid violation_id format")
	}

	// Get violation first to check authorization
	violation, err := h.db.GetDetectedViolation(ctx, pgtype.UUID{Bytes: violationID, Valid: true})
	if err != nil {
		h.logger.Error("violation not found",
			slog.String("violation_id", violationID.String()),
			slog.String("error", err.Error()),
		)
		return echo.NewHTTPError(http.StatusNotFound, "Violation not found")
	}

	// Authorization: verify user has access to this violation's organization
	userID, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	orgID, err := getOrganizationIDFromPhoto(ctx, h.db, violation.PhotoID)
	if err != nil {
		return err
	}

	_, err = requireOrganizationMembership(ctx, h.pool, h.logger, userID, orgID)
	if err != nil {
		return err
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

// ConfirmViolation marks a violation as confirmed
func (h *ViolationHandler) ConfirmViolation(c echo.Context) error {
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	violationIDStr := c.Param("id")
	violationID, err := uuid.Parse(violationIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid violation_id format")
	}

	// Get violation first to check authorization
	existingViolation, err := h.db.GetDetectedViolation(ctx, pgtype.UUID{Bytes: violationID, Valid: true})
	if err != nil {
		h.logger.Error("violation not found",
			slog.String("violation_id", violationID.String()),
			slog.String("error", err.Error()),
		)
		return echo.NewHTTPError(http.StatusNotFound, "Violation not found")
	}

	// Authorization: verify user has access to this violation's organization
	userID, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	orgID, err := getOrganizationIDFromPhoto(ctx, h.db, existingViolation.PhotoID)
	if err != nil {
		return err
	}

	_, err = requireOrganizationMembership(ctx, h.pool, h.logger, userID, orgID)
	if err != nil {
		return err
	}

	// Update violation status to confirmed
	violation, err := h.db.UpdateDetectedViolationStatus(ctx, database.UpdateDetectedViolationStatusParams{
		ID:     pgtype.UUID{Bytes: violationID, Valid: true},
		Status: database.ViolationStatusConfirmed,
	})
	if err != nil {
		h.logger.Error("failed to confirm violation",
			slog.String("violation_id", violationIDStr),
			slog.String("error", err.Error()),
		)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to confirm violation")
	}

	h.logger.Info("violation confirmed",
		slog.String("violation_id", violationIDStr),
	)

	// If HTMX request, return updated HTML
	if c.Request().Header.Get("HX-Request") == "true" {
		return h.renderViolationCard(c, violation)
	}

	// Otherwise return JSON
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

// DismissViolation marks a violation as dismissed
func (h *ViolationHandler) DismissViolation(c echo.Context) error {
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	violationIDStr := c.Param("id")
	violationID, err := uuid.Parse(violationIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid violation_id format")
	}

	// Get violation first to check authorization
	existingViolation, err := h.db.GetDetectedViolation(ctx, pgtype.UUID{Bytes: violationID, Valid: true})
	if err != nil {
		h.logger.Error("violation not found",
			slog.String("violation_id", violationID.String()),
			slog.String("error", err.Error()),
		)
		return echo.NewHTTPError(http.StatusNotFound, "Violation not found")
	}

	// Authorization: verify user has access to this violation's organization
	userID, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	orgID, err := getOrganizationIDFromPhoto(ctx, h.db, existingViolation.PhotoID)
	if err != nil {
		return err
	}

	_, err = requireOrganizationMembership(ctx, h.pool, h.logger, userID, orgID)
	if err != nil {
		return err
	}

	// Update violation status to dismissed
	violation, err := h.db.UpdateDetectedViolationStatus(ctx, database.UpdateDetectedViolationStatusParams{
		ID:     pgtype.UUID{Bytes: violationID, Valid: true},
		Status: database.ViolationStatusDismissed,
	})
	if err != nil {
		h.logger.Error("failed to dismiss violation",
			slog.String("violation_id", violationIDStr),
			slog.String("error", err.Error()),
		)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to dismiss violation")
	}

	h.logger.Info("violation dismissed",
		slog.String("violation_id", violationIDStr),
	)

	// If HTMX request, return updated HTML
	if c.Request().Header.Get("HX-Request") == "true" {
		return h.renderViolationCard(c, violation)
	}

	// Otherwise return JSON
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

// renderViolationCard renders a violation card HTML for HTMX updates
func (h *ViolationHandler) renderViolationCard(c echo.Context, violation database.DetectedViolation) error {
	ctx := c.Request().Context()

	// Get safety code if available
	safetyCodeText := ""
	if violation.SafetyCodeID.Valid {
		safetyCode, err := h.db.GetSafetyCode(ctx, violation.SafetyCodeID)
		if err == nil {
			safetyCodeText = safetyCode.Code + " - " + safetyCode.Description
		}
	}

	// Determine background and border colors based on status and severity
	bgColor := "#fef2f2" // default pending
	borderColor := "#dc2626" // default critical
	if violation.Status == database.ViolationStatusConfirmed {
		bgColor = "#d1fae5"
		borderColor = "#059669"
	} else if violation.Status == database.ViolationStatusDismissed {
		bgColor = "#f3f4f6"
		borderColor = "#9ca3af"
	} else {
		// Use severity for border color when pending
		switch violation.Severity {
		case database.ViolationSeverityCritical:
			borderColor = "#dc2626"
		case database.ViolationSeverityHigh:
			borderColor = "#f97316"
		case database.ViolationSeverityMedium:
			borderColor = "#fbbf24"
		default:
			borderColor = "#94a3b8"
		}
	}

	// Severity badge colors
	severityBg := "#94a3b8"
	severityText := "white"
	switch violation.Severity {
	case database.ViolationSeverityCritical:
		severityBg = "#dc2626"
		severityText = "white"
	case database.ViolationSeverityHigh:
		severityBg = "#f97316"
		severityText = "white"
	case database.ViolationSeverityMedium:
		severityBg = "#fbbf24"
		severityText = "#78350f"
	}

	// Status badge colors
	statusBg := "#3b82f6" // pending
	statusText := "white"
	if violation.Status == database.ViolationStatusConfirmed {
		statusBg = "#059669"
	} else if violation.Status == database.ViolationStatusDismissed {
		statusBg = "#6b7280"
	}

	// Get confidence percentage
	confidenceFloat, _ := violation.ConfidenceScore.Float64Value()
	confidence := fmt.Sprintf("%.0f", confidenceFloat.Float64*100)

	// Build HTML
	html := `<div id="violation-` + violation.ID.String() + `" class="card" style="padding: var(--space-md); background: ` + bgColor + `; border-left: 4px solid ` + borderColor + `;">
		<!-- Violation Header -->
		<div style="display: flex; justify-content: space-between; align-items: start; margin-bottom: var(--space-sm);">
			<div style="display: flex; gap: var(--space-xs); align-items: center;">
				<span style="padding: 0.25rem 0.5rem; border-radius: 4px; font-size: 0.75rem; font-weight: 600; background: ` + severityBg + `; color: ` + severityText + `;">
					` + string(violation.Severity) + `
				</span>
				<span style="padding: 0.25rem 0.5rem; border-radius: 4px; font-size: 0.75rem; font-weight: 600; background: ` + statusBg + `; color: ` + statusText + `;">
					` + string(violation.Status) + `
				</span>
			</div>
			<span style="font-size: 0.75rem; color: #64748b;">
				` + confidence + `% confidence
			</span>
		</div>

		<!-- Violation Description -->
		<p style="font-size: 0.95rem; color: #1f2937; margin-bottom: var(--space-sm); line-height: 1.5;">
			` + violation.Description + `
		</p>`

	if violation.Location.Valid {
		html += `
		<p style="font-size: 0.875rem; color: #64748b; margin-bottom: var(--space-sm);">
			<strong>Location:</strong> ` + violation.Location.String + `
		</p>`
	}

	if safetyCodeText != "" {
		html += `
		<p style="font-size: 0.875rem; color: #64748b; margin-bottom: var(--space-md);">
			<strong>Safety Code:</strong> ` + safetyCodeText + `
		</p>`
	}

	// Action buttons based on status
	if violation.Status == database.ViolationStatusPending {
		html += `
		<!-- Action Buttons -->
		<div style="display: flex; gap: var(--space-sm); margin-top: var(--space-md);">
			<button
				hx-post="/api/violations/` + violation.ID.String() + `/confirm"
				hx-target="#violation-` + violation.ID.String() + `"
				hx-swap="outerHTML"
				class="btn-primary"
				style="flex: 1;">
				✓ Confirm Violation
			</button>
			<button
				hx-post="/api/violations/` + violation.ID.String() + `/dismiss"
				hx-target="#violation-` + violation.ID.String() + `"
				hx-swap="outerHTML"
				class="btn-secondary"
				style="flex: 1;">
				✗ Dismiss
			</button>
		</div>`
	} else if violation.Status == database.ViolationStatusConfirmed {
		html += `
		<div style="margin-top: var(--space-md);">
			<p style="color: #059669; font-weight: 600; font-size: 0.875rem; margin-bottom: var(--space-xs);">
				✓ Confirmed by inspector
			</p>
			<button
				hx-post="/api/violations/` + violation.ID.String() + `/dismiss"
				hx-target="#violation-` + violation.ID.String() + `"
				hx-swap="outerHTML"
				class="btn-secondary"
				style="font-size: 0.875rem; padding: 0.375rem 0.75rem;">
				Change to Dismissed
			</button>
		</div>`
	} else if violation.Status == database.ViolationStatusDismissed {
		html += `
		<div style="margin-top: var(--space-md);">
			<p style="color: #6b7280; font-weight: 600; font-size: 0.875rem; margin-bottom: var(--space-xs);">
				✗ Dismissed by inspector
			</p>
			<button
				hx-post="/api/violations/` + violation.ID.String() + `/confirm"
				hx-target="#violation-` + violation.ID.String() + `"
				hx-swap="outerHTML"
				class="btn-primary"
				style="font-size: 0.875rem; padding: 0.375rem 0.75rem;">
				Change to Confirmed
			</button>
		</div>`
	}

	html += `
	</div>`

	return c.HTML(http.StatusOK, html)
}

// CreateManualViolationRequest represents the request body for creating a manual violation
type CreateManualViolationRequest struct {
	PhotoID    string `form:"photo_id" json:"photo_id" validate:"required,uuid"`
	SafetyCode string `form:"safety_code" json:"safety_code" validate:"required"`
	Description string `form:"description" json:"description" validate:"required"`
	Severity   string `form:"severity" json:"severity" validate:"required,oneof=critical high medium low"`
	Location   string `form:"location" json:"location"`
}

// CreateManualViolation godoc
// @Summary Create a manual violation
// @Description Allows inspectors to manually add violations that AI might have missed
// @Tags violations
// @Accept json,multipart/form-data
// @Produce json,html
// @Param request body CreateManualViolationRequest true "Manual Violation Request"
// @Success 200 {object} ViolationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/violations/manual [post]
func (h *ViolationHandler) CreateManualViolation(c echo.Context) error {
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	var req CreateManualViolationRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	// Validate required fields
	if req.PhotoID == "" || req.SafetyCode == "" || req.Description == "" || req.Severity == "" {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div style="padding: var(--space-sm); background: #fef2f2; border-radius: 4px; color: #dc2626; font-size: 0.875rem;">All required fields must be filled out</div>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "All required fields must be filled out")
	}

	// Parse photo ID
	photoID, err := uuid.Parse(req.PhotoID)
	if err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div style="padding: var(--space-sm); background: #fef2f2; border-radius: 4px; color: #dc2626; font-size: 0.875rem;">Invalid photo ID</div>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid photo ID")
	}

	// Verify photo exists
	photo, err := h.db.GetPhoto(ctx, pgtype.UUID{Bytes: photoID, Valid: true})
	if err != nil {
		h.logger.Error("photo not found", slog.String("photo_id", photoID.String()))
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusNotFound, `<div style="padding: var(--space-sm); background: #fef2f2; border-radius: 4px; color: #dc2626; font-size: 0.875rem;">Photo not found</div>`)
		}
		return echo.NewHTTPError(http.StatusNotFound, "Photo not found")
	}

	// Authorization: verify user has access to this photo's organization
	userID, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session")
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusUnauthorized, `<div style="padding: var(--space-sm); background: #fef2f2; border-radius: 4px; color: #dc2626; font-size: 0.875rem;">Unauthorized</div>`)
		}
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	orgID, err := getOrganizationIDFromPhoto(ctx, h.db, photo.ID)
	if err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusForbidden, `<div style="padding: var(--space-sm); background: #fef2f2; border-radius: 4px; color: #dc2626; font-size: 0.875rem;">Access denied</div>`)
		}
		return err
	}

	_, err = requireOrganizationMembership(ctx, h.pool, h.logger, userID, orgID)
	if err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusForbidden, `<div style="padding: var(--space-sm); background: #fef2f2; border-radius: 4px; color: #dc2626; font-size: 0.875rem;">You are not a member of this organization</div>`)
		}
		return err
	}

	// Try to find matching safety code in database
	var safetyCodeID pgtype.UUID
	safetyCode, err := h.db.GetSafetyCodeByCode(ctx, req.SafetyCode)
	if err == nil {
		safetyCodeID = safetyCode.ID
	}
	// If not found, we'll create the violation anyway with the code string

	// Map severity string to database type
	var severity database.ViolationSeverity
	switch req.Severity {
	case "critical":
		severity = database.ViolationSeverityCritical
	case "high":
		severity = database.ViolationSeverityHigh
	case "medium":
		severity = database.ViolationSeverityMedium
	case "low":
		severity = database.ViolationSeverityLow
	default:
		severity = database.ViolationSeverityMedium
	}

	// Prepare location field
	locationText := pgtype.Text{
		String: req.Location,
		Valid:  req.Location != "",
	}

	// Create the violation with 100% confidence (manually created by inspector)
	confidenceInt := new(big.Int).SetInt64(10000) // 1.0 * 10000
	violation, err := h.db.CreateDetectedViolation(ctx, database.CreateDetectedViolationParams{
		PhotoID:         photo.ID,
		Description:     req.Description,
		ConfidenceScore: pgtype.Numeric{Int: confidenceInt, Exp: -4, Valid: true},
		SafetyCodeID:    safetyCodeID,
		Status:          database.ViolationStatusConfirmed, // Manual violations start as confirmed
		Severity:        severity,
		Location:        locationText,
	})

	if err != nil {
		h.logger.Error("failed to create manual violation",
			slog.String("photo_id", photoID.String()),
			slog.String("error", err.Error()))
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusInternalServerError, `<div style="padding: var(--space-sm); background: #fef2f2; border-radius: 4px; color: #dc2626; font-size: 0.875rem;">Failed to create violation. Please try again.</div>`)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create violation")
	}

	h.logger.Info("manual violation created",
		slog.String("photo_id", photoID.String()),
		slog.String("violation_id", violation.ID.String()),
		slog.String("safety_code", req.SafetyCode))

	// If HTMX request, return success message and reload page
	if c.Request().Header.Get("HX-Request") == "true" {
		html := `
		<div style="padding: var(--space-sm); background: #d1fae5; border-radius: 4px; color: #059669; font-size: 0.875rem;">
			✓ Violation added successfully! Refreshing page...
		</div>
		<script>
			setTimeout(function() {
				window.location.reload();
			}, 1500);
		</script>`
		return c.HTML(http.StatusOK, html)
	}

	// Return JSON response
	resp := ViolationResponse{
		ID:              violation.ID.String(),
		PhotoID:         violation.PhotoID.String(),
		Description:     violation.Description,
		ConfidenceScore: 1.0,
		Severity:        string(violation.Severity),
		Status:          string(violation.Status),
		CreatedAt:       violation.CreatedAt.Time.String(),
	}

	if violation.Location.Valid {
		resp.Location = &violation.Location.String
	}
	if violation.SafetyCodeID.Valid {
		id := violation.SafetyCodeID.String()
		resp.SafetyCodeID = &id
	}

	return c.JSON(http.StatusOK, resp)
}
