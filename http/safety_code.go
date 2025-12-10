package http

import (
	"log/slog"
	"net/http"

	"github.com/dukerupert/aletheia"
	"github.com/labstack/echo/v4"
)

// CreateSafetyCodeRequest is the request payload for creating a safety code.
type CreateSafetyCodeRequest struct {
	Code          string `json:"code" form:"code" validate:"required,min=2,max=50"`
	Description   string `json:"description" form:"description" validate:"required,min=5,max=500"`
	Country       string `json:"country" form:"country" validate:"omitempty,max=50"`
	StateProvince string `json:"state_province" form:"state_province" validate:"omitempty,max=50"`
}

func (s *Server) handleCreateSafetyCode(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	var req CreateSafetyCodeRequest
	if err := bind(c, &req); err != nil {
		return err
	}

	safetyCode := &aletheia.SafetyCode{
		Code:          req.Code,
		Description:   req.Description,
		Country:       req.Country,
		StateProvince: req.StateProvince,
	}

	if err := s.safetyCodeService.CreateSafetyCode(ctx, safetyCode); err != nil {
		return err
	}

	s.log(c).Info("safety code created",
		slog.String("safety_code_id", safetyCode.ID.String()),
		slog.String("code", safetyCode.Code),
	)

	return RespondCreated(c, safetyCode)
}

func (s *Server) handleListSafetyCodes(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	// Parse optional query parameters
	search := c.QueryParam("search")
	country := c.QueryParam("country")

	filter := aletheia.SafetyCodeFilter{
		Limit: 100,
	}

	if search != "" {
		filter.Search = &search
	}
	if country != "" {
		filter.Country = &country
	}

	safetyCodes, total, err := s.safetyCodeService.FindSafetyCodes(ctx, filter)
	if err != nil {
		return err
	}

	return RespondOK(c, map[string]interface{}{
		"safety_codes": safetyCodes,
		"total":        total,
	})
}

func (s *Server) handleGetSafetyCode(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	safetyCodeID, err := requireUUIDParam(c, "id")
	if err != nil {
		return err
	}

	safetyCode, err := s.safetyCodeService.FindSafetyCodeByID(ctx, safetyCodeID)
	if err != nil {
		return err
	}

	return RespondOK(c, safetyCode)
}

// UpdateSafetyCodeRequest is the request payload for updating a safety code.
type UpdateSafetyCodeRequest struct {
	Code          *string `json:"code" form:"code" validate:"omitempty,min=2,max=50"`
	Description   *string `json:"description" form:"description" validate:"omitempty,min=5,max=500"`
	Country       *string `json:"country" form:"country" validate:"omitempty,max=50"`
	StateProvince *string `json:"state_province" form:"state_province" validate:"omitempty,max=50"`
}

func (s *Server) handleUpdateSafetyCode(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	safetyCodeID, err := requireUUIDParam(c, "id")
	if err != nil {
		return err
	}

	var req UpdateSafetyCodeRequest
	if err := bind(c, &req); err != nil {
		return err
	}

	safetyCode, err := s.safetyCodeService.UpdateSafetyCode(ctx, safetyCodeID, aletheia.SafetyCodeUpdate{
		Code:          req.Code,
		Description:   req.Description,
		Country:       req.Country,
		StateProvince: req.StateProvince,
	})
	if err != nil {
		return err
	}

	s.log(c).Info("safety code updated", slog.String("safety_code_id", safetyCode.ID.String()))

	return RespondOK(c, safetyCode)
}

func (s *Server) handleDeleteSafetyCode(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	safetyCodeID, err := requireUUIDParam(c, "id")
	if err != nil {
		return err
	}

	if err := s.safetyCodeService.DeleteSafetyCode(ctx, safetyCodeID); err != nil {
		return err
	}

	s.log(c).Info("safety code deleted", slog.String("safety_code_id", safetyCodeID.String()))

	return c.NoContent(http.StatusNoContent)
}
