package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/dukerupert/aletheia/internal/database"
	"github.com/dukerupert/aletheia/internal/session"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

type SafetyCodeHandler struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewSafetyCodeHandler(pool *pgxpool.Pool, logger *slog.Logger) *SafetyCodeHandler {
	return &SafetyCodeHandler{
		pool:   pool,
		logger: logger,
	}
}

// CreateSafetyCodeRequest is the request payload for creating a safety code
type CreateSafetyCodeRequest struct {
	Code          string  `json:"code" validate:"required"`
	Description   string  `json:"description" validate:"required"`
	Country       *string `json:"country,omitempty"`
	StateProvince *string `json:"state_province,omitempty"`
}

// SafetyCodeResponse is the response payload for safety code operations
type SafetyCodeResponse struct {
	ID            string  `json:"id"`
	Code          string  `json:"code"`
	Description   string  `json:"description"`
	Country       *string `json:"country,omitempty"`
	StateProvince *string `json:"state_province,omitempty"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

// CreateSafetyCode creates a new safety code
func (h *SafetyCodeHandler) CreateSafetyCode(c echo.Context) error {
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	// Get authenticated user from session
	_, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	var req CreateSafetyCodeRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// Validate required fields
	if req.Code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "code is required")
	}
	if req.Description == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "description is required")
	}

	queries := database.New(h.pool)

	// Convert optional fields to pgtype.Text
	var country, stateProvince pgtype.Text
	if req.Country != nil {
		country = pgtype.Text{String: *req.Country, Valid: true}
	}
	if req.StateProvince != nil {
		stateProvince = pgtype.Text{String: *req.StateProvince, Valid: true}
	}

	// Create safety code
	safetyCode, err := queries.CreateSafetyCode(ctx, database.CreateSafetyCodeParams{
		Code:          req.Code,
		Description:   req.Description,
		Country:       country,
		StateProvince: stateProvince,
	})
	if err != nil {
		h.logger.Error("failed to create safety code", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create safety code")
	}

	h.logger.Info("safety code created",
		slog.String("safety_code_id", safetyCode.ID.String()),
		slog.String("code", safetyCode.Code))

	return c.JSON(http.StatusCreated, safetyCodeToResponse(safetyCode))
}

// GetSafetyCode retrieves a safety code by ID
func (h *SafetyCodeHandler) GetSafetyCode(c echo.Context) error {
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	// Get authenticated user from session
	_, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	safetyCodeID := c.Param("id")
	if safetyCodeID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "safety code id is required")
	}

	queries := database.New(h.pool)

	// Parse safety code ID
	safetyCodeUUID, err := parseUUID(safetyCodeID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid safety code id")
	}

	// Get safety code
	safetyCode, err := queries.GetSafetyCode(ctx, safetyCodeUUID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "safety code not found")
	}

	return c.JSON(http.StatusOK, safetyCodeToResponse(safetyCode))
}

// ListSafetyCodesResponse is the response payload for listing safety codes
type ListSafetyCodesResponse struct {
	SafetyCodes []SafetyCodeResponse `json:"safety_codes"`
}

// ListSafetyCodes lists all safety codes with optional filtering
func (h *SafetyCodeHandler) ListSafetyCodes(c echo.Context) error {
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	// Get authenticated user from session
	_, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	queries := database.New(h.pool)

	// Check for optional filters
	country := c.QueryParam("country")
	stateProvince := c.QueryParam("state_province")

	var safetyCodes []database.SafetyCode
	var err error

	if country != "" {
		// Filter by country
		safetyCodes, err = queries.ListSafetyCodesByCountry(ctx, pgtype.Text{String: country, Valid: true})
	} else if stateProvince != "" {
		// Filter by state/province
		safetyCodes, err = queries.ListSafetyCodesByStateProvince(ctx, pgtype.Text{String: stateProvince, Valid: true})
	} else {
		// List all
		safetyCodes, err = queries.ListSafetyCodes(ctx)
	}

	if err != nil {
		h.logger.Error("failed to list safety codes", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list safety codes")
	}

	responses := make([]SafetyCodeResponse, len(safetyCodes))
	for i, sc := range safetyCodes {
		responses[i] = safetyCodeToResponse(sc)
	}

	return c.JSON(http.StatusOK, ListSafetyCodesResponse{
		SafetyCodes: responses,
	})
}

// UpdateSafetyCodeRequest is the request payload for updating a safety code
type UpdateSafetyCodeRequest struct {
	Code          *string `json:"code,omitempty"`
	Description   *string `json:"description,omitempty"`
	Country       *string `json:"country,omitempty"`
	StateProvince *string `json:"state_province,omitempty"`
}

// UpdateSafetyCode updates an existing safety code
func (h *SafetyCodeHandler) UpdateSafetyCode(c echo.Context) error {
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	// Get authenticated user from session
	_, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	safetyCodeID := c.Param("id")
	if safetyCodeID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "safety code id is required")
	}

	var req UpdateSafetyCodeRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	queries := database.New(h.pool)

	// Parse safety code ID
	safetyCodeUUID, err := parseUUID(safetyCodeID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid safety code id")
	}

	// Get existing safety code to use as base values
	existingSafetyCode, err := queries.GetSafetyCode(ctx, safetyCodeUUID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "safety code not found")
	}

	// Use existing values if not provided in request
	code := existingSafetyCode.Code
	if req.Code != nil {
		code = *req.Code
	}

	description := existingSafetyCode.Description
	if req.Description != nil {
		description = *req.Description
	}

	country := existingSafetyCode.Country
	if req.Country != nil {
		country = pgtype.Text{String: *req.Country, Valid: true}
	}

	stateProvince := existingSafetyCode.StateProvince
	if req.StateProvince != nil {
		stateProvince = pgtype.Text{String: *req.StateProvince, Valid: true}
	}

	// Update safety code
	safetyCode, err := queries.UpdateSafetyCode(ctx, database.UpdateSafetyCodeParams{
		ID:            safetyCodeUUID,
		Code:          code,
		Description:   description,
		Country:       country,
		StateProvince: stateProvince,
	})
	if err != nil {
		h.logger.Error("failed to update safety code", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update safety code")
	}

	h.logger.Info("safety code updated",
		slog.String("safety_code_id", safetyCode.ID.String()))

	return c.JSON(http.StatusOK, safetyCodeToResponse(safetyCode))
}

// DeleteSafetyCode deletes a safety code by ID
func (h *SafetyCodeHandler) DeleteSafetyCode(c echo.Context) error {
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	// Get authenticated user from session
	_, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	safetyCodeID := c.Param("id")
	if safetyCodeID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "safety code id is required")
	}

	queries := database.New(h.pool)

	// Parse safety code ID
	safetyCodeUUID, err := parseUUID(safetyCodeID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid safety code id")
	}

	// Check if safety code exists
	_, err = queries.GetSafetyCode(ctx, safetyCodeUUID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "safety code not found")
	}

	// Delete safety code
	if err := queries.DeleteSafetyCode(ctx, safetyCodeUUID); err != nil {
		h.logger.Error("failed to delete safety code", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete safety code")
	}

	h.logger.Info("safety code deleted",
		slog.String("safety_code_id", safetyCodeID))

	return c.NoContent(http.StatusNoContent)
}

// safetyCodeToResponse converts a database safety code to a response
func safetyCodeToResponse(sc database.SafetyCode) SafetyCodeResponse {
	response := SafetyCodeResponse{
		ID:          sc.ID.String(),
		Code:        sc.Code,
		Description: sc.Description,
		CreatedAt:   sc.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   sc.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}

	if sc.Country.Valid {
		response.Country = &sc.Country.String
	}
	if sc.StateProvince.Valid {
		response.StateProvince = &sc.StateProvince.String
	}

	return response
}
