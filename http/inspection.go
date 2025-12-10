package http

import (
	"log/slog"

	"github.com/dukerupert/aletheia"
	"github.com/labstack/echo/v4"
)

// CreateInspectionRequest is the request payload for creating an inspection.
type CreateInspectionRequest struct {
	ProjectID string `json:"project_id" form:"project_id" validate:"required,uuid"`
}

func (s *Server) handleCreateInspection(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	userID, err := requireUserID(c)
	if err != nil {
		return err
	}

	var req CreateInspectionRequest
	if err := bind(c, &req); err != nil {
		return err
	}

	projectID, err := parseUUID(req.ProjectID)
	if err != nil {
		return err
	}

	inspection := &aletheia.Inspection{
		ProjectID:   projectID,
		InspectorID: userID,
		Status:      aletheia.InspectionStatusDraft,
	}

	if err := s.inspectionService.CreateInspection(ctx, inspection); err != nil {
		return err
	}

	s.log(c).Info("inspection created",
		slog.String("inspection_id", inspection.ID.String()),
		slog.String("project_id", projectID.String()),
	)

	return RespondCreated(c, inspection)
}

func (s *Server) handleGetInspection(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	inspectionID, err := requireUUIDParam(c, "id")
	if err != nil {
		return err
	}

	inspection, err := s.inspectionService.FindInspectionByID(ctx, inspectionID)
	if err != nil {
		return err
	}

	return RespondOK(c, inspection)
}

func (s *Server) handleListInspections(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	projectID, err := requireUUIDParam(c, "projectId")
	if err != nil {
		return err
	}

	// Parse optional query parameters
	statusStr := c.QueryParam("status")

	filter := aletheia.InspectionFilter{
		ProjectID: &projectID,
		Limit:     100,
	}

	if statusStr != "" {
		status := aletheia.InspectionStatus(statusStr)
		filter.Status = &status
	}

	inspections, total, err := s.inspectionService.FindInspections(ctx, filter)
	if err != nil {
		return err
	}

	return RespondOK(c, map[string]interface{}{
		"inspections": inspections,
		"total":       total,
	})
}

// UpdateInspectionStatusRequest is the request payload for updating inspection status.
type UpdateInspectionStatusRequest struct {
	Status string `json:"status" form:"status" validate:"required,oneof=draft in_progress completed"`
}

func (s *Server) handleUpdateInspectionStatus(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	inspectionID, err := requireUUIDParam(c, "id")
	if err != nil {
		return err
	}

	var req UpdateInspectionStatusRequest
	if err := bind(c, &req); err != nil {
		return err
	}

	status := aletheia.InspectionStatus(req.Status)

	inspection, err := s.inspectionService.UpdateInspectionStatus(ctx, inspectionID, status)
	if err != nil {
		return err
	}

	s.log(c).Info("inspection status updated",
		slog.String("inspection_id", inspectionID.String()),
		slog.String("status", string(status)),
	)

	return RespondOK(c, inspection)
}
