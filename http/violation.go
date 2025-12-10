package http

import (
	"log/slog"

	"github.com/dukerupert/aletheia"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// CreateViolationRequest is the request payload for creating a manual violation.
type CreateViolationRequest struct {
	PhotoID      string `json:"photo_id" form:"photo_id" validate:"required,uuid"`
	SafetyCodeID string `json:"safety_code_id" form:"safety_code_id" validate:"omitempty,uuid"`
	Description  string `json:"description" form:"description" validate:"required,min=5,max=500"`
	Severity     string `json:"severity" form:"severity" validate:"required,oneof=critical high medium low"`
	Location     string `json:"location" form:"location" validate:"omitempty,max=200"`
}

func (s *Server) handleCreateViolation(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	var req CreateViolationRequest
	if err := bind(c, &req); err != nil {
		return err
	}

	photoID, err := parseUUID(req.PhotoID)
	if err != nil {
		return err
	}

	// Verify photo exists
	_, err = s.photoService.FindPhotoByID(ctx, photoID)
	if err != nil {
		return err
	}

	violation := &aletheia.Violation{
		PhotoID:     photoID,
		Description: req.Description,
		Severity:    aletheia.Severity(req.Severity),
		Status:      aletheia.ViolationStatusConfirmed, // Manual entries are auto-confirmed
		Location:    req.Location,
	}

	// Parse optional safety code ID
	if req.SafetyCodeID != "" {
		safetyCodeID, err := parseUUID(req.SafetyCodeID)
		if err != nil {
			return err
		}
		violation.SafetyCodeID = safetyCodeID
	}

	if err := s.violationService.CreateViolation(ctx, violation); err != nil {
		return err
	}

	s.log(c).Info("violation created",
		slog.String("violation_id", violation.ID.String()),
		slog.String("photo_id", photoID.String()),
	)

	return RespondCreated(c, violation)
}

func (s *Server) handleListViolations(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	inspectionID, err := requireUUIDParam(c, "inspectionId")
	if err != nil {
		return err
	}

	// Parse optional query parameters
	statusStr := c.QueryParam("status")
	severityStr := c.QueryParam("severity")

	filter := aletheia.ViolationFilter{
		InspectionID: &inspectionID,
		Limit:        100,
	}

	if statusStr != "" {
		status := aletheia.ViolationStatus(statusStr)
		filter.Status = &status
	}
	if severityStr != "" {
		severity := aletheia.Severity(severityStr)
		filter.Severity = &severity
	}

	violations, total, err := s.violationService.FindViolations(ctx, filter)
	if err != nil {
		return err
	}

	return RespondOK(c, map[string]interface{}{
		"violations": violations,
		"total":      total,
	})
}

func (s *Server) handleConfirmViolation(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	violationID, err := requireUUIDParam(c, "id")
	if err != nil {
		return err
	}

	violation, err := s.violationService.ConfirmViolation(ctx, violationID)
	if err != nil {
		return err
	}

	s.log(c).Info("violation confirmed", slog.String("violation_id", violationID.String()))

	return RespondOK(c, violation)
}

func (s *Server) handleDismissViolation(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	violationID, err := requireUUIDParam(c, "id")
	if err != nil {
		return err
	}

	violation, err := s.violationService.DismissViolation(ctx, violationID)
	if err != nil {
		return err
	}

	s.log(c).Info("violation dismissed", slog.String("violation_id", violationID.String()))

	return RespondOK(c, violation)
}

func (s *Server) handleSetViolationPending(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	violationID, err := requireUUIDParam(c, "id")
	if err != nil {
		return err
	}

	violation, err := s.violationService.SetViolationPending(ctx, violationID)
	if err != nil {
		return err
	}

	s.log(c).Info("violation set to pending", slog.String("violation_id", violationID.String()))

	return RespondOK(c, violation)
}

// UpdateViolationRequest is the request payload for updating a violation.
type UpdateViolationRequest struct {
	Description  *string `json:"description" form:"description" validate:"omitempty,min=5,max=500"`
	Severity     *string `json:"severity" form:"severity" validate:"omitempty,oneof=critical high medium low"`
	SafetyCodeID *string `json:"safety_code_id" form:"safety_code_id" validate:"omitempty,uuid"`
	Location     *string `json:"location" form:"location" validate:"omitempty,max=200"`
}

func (s *Server) handleUpdateViolation(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	violationID, err := requireUUIDParam(c, "id")
	if err != nil {
		return err
	}

	var req UpdateViolationRequest
	if err := bind(c, &req); err != nil {
		return err
	}

	upd := aletheia.ViolationUpdate{
		Description: req.Description,
		Location:    req.Location,
	}

	if req.Severity != nil {
		severity := aletheia.Severity(*req.Severity)
		upd.Severity = &severity
	}

	if req.SafetyCodeID != nil {
		safetyCodeID, err := uuid.Parse(*req.SafetyCodeID)
		if err != nil {
			return aletheia.Invalid("Invalid safety_code_id format")
		}
		upd.SafetyCodeID = &safetyCodeID
	}

	violation, err := s.violationService.UpdateViolation(ctx, violationID, upd)
	if err != nil {
		return err
	}

	s.log(c).Info("violation updated", slog.String("violation_id", violationID.String()))

	return RespondOK(c, violation)
}
