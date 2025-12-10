package http

// This file demonstrates the target handler pattern for Phase 4.
// It shows how handlers should use domain errors instead of echo.NewHTTPError.
//
// Key differences from current handlers:
// 1. Return domain errors (aletheia.NotFound, aletheia.Invalid, etc.)
// 2. Use HandleError() for error responses
// 3. Use WrapDatabaseError() for database errors
// 4. Use response helpers (RespondOK, RespondCreated, etc.)
//
// Example migration:
//
// BEFORE:
//
//	if err != nil {
//	    if errors.Is(err, pgx.ErrNoRows) {
//	        return echo.NewHTTPError(http.StatusNotFound, "safety code not found")
//	    }
//	    h.logger.Error("failed to get safety code", slog.String("err", err.Error()))
//	    return echo.NewHTTPError(http.StatusInternalServerError, "failed to get safety code")
//	}
//
// AFTER:
//
//	if err != nil {
//	    return WrapDatabaseError(err, "Safety code not found", "Failed to get safety code")
//	}
//
// The error middleware will:
// - Map domain error codes to HTTP status codes
// - Log internal errors with full context
// - Return appropriate JSON or HTMX responses

import (
	"context"
	"log/slog"
	"time"

	"github.com/dukerupert/aletheia"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// ExampleHandler demonstrates the target handler pattern.
// In Phase 4, this pattern will be used for all handlers.
type ExampleHandler struct {
	logger           *slog.Logger
	safetyCodeSvc    aletheia.SafetyCodeService // Uses domain interface
}

// NewExampleHandler creates a new example handler.
func NewExampleHandler(logger *slog.Logger, safetyCodeSvc aletheia.SafetyCodeService) *ExampleHandler {
	return &ExampleHandler{
		logger:           logger,
		safetyCodeSvc:    safetyCodeSvc,
	}
}

// GetSafetyCode demonstrates the target pattern for a GET handler.
func (h *ExampleHandler) GetSafetyCode(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	// Get user from context (set by auth middleware)
	user := aletheia.UserFromContext(ctx)
	if user == nil {
		return aletheia.Unauthorized("Authentication required")
	}

	// Parse and validate ID
	idStr := c.Param("id")
	if idStr == "" {
		return aletheia.Invalid("Safety code ID is required")
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return aletheia.Invalid("Invalid safety code ID format")
	}

	// Call service (returns domain errors)
	safetyCode, err := h.safetyCodeSvc.FindSafetyCodeByID(ctx, id)
	if err != nil {
		// Error is already a domain error from the service
		return err
	}

	// Return success response
	return RespondOK(c, safetyCode)
}

// CreateSafetyCode demonstrates the target pattern for a POST handler.
func (h *ExampleHandler) CreateSafetyCode(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	// Get user from context
	user := aletheia.UserFromContext(ctx)
	if user == nil {
		return aletheia.Unauthorized("Authentication required")
	}

	// Bind request
	var req struct {
		Code          string `json:"code"`
		Description   string `json:"description"`
		Country       string `json:"country,omitempty"`
		StateProvince string `json:"stateProvince,omitempty"`
	}
	if err := c.Bind(&req); err != nil {
		return aletheia.Invalid("Invalid request body")
	}

	// Validate (can also use go-playground/validator)
	if req.Code == "" {
		return aletheia.ErrorWithFields(map[string]string{
			"code": "Code is required",
		})
	}
	if req.Description == "" {
		return aletheia.ErrorWithFields(map[string]string{
			"description": "Description is required",
		})
	}

	// Create domain object
	safetyCode := &aletheia.SafetyCode{
		Code:          req.Code,
		Description:   req.Description,
		Country:       req.Country,
		StateProvince: req.StateProvince,
	}

	// Call service
	if err := h.safetyCodeSvc.CreateSafetyCode(ctx, safetyCode); err != nil {
		return err
	}

	h.logger.Info("safety code created",
		slog.String("id", safetyCode.ID.String()),
		slog.String("code", safetyCode.Code),
	)

	return RespondCreated(c, safetyCode)
}

// DeleteSafetyCode demonstrates the target pattern for a DELETE handler.
func (h *ExampleHandler) DeleteSafetyCode(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	// Get user from context
	user := aletheia.UserFromContext(ctx)
	if user == nil {
		return aletheia.Unauthorized("Authentication required")
	}

	// Parse ID
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return aletheia.Invalid("Invalid safety code ID format")
	}

	// Call service
	if err := h.safetyCodeSvc.DeleteSafetyCode(ctx, id); err != nil {
		return err
	}

	h.logger.Info("safety code deleted", slog.String("id", id.String()))

	return RespondNoContent(c)
}
