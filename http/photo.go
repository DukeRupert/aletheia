package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/dukerupert/aletheia"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const maxPhotoSize = 5 * 1024 * 1024 // 5MB

// UploadPhotoRequest is the request payload for uploading a photo.
type UploadPhotoRequest struct {
	InspectionID string `form:"inspection_id" validate:"required,uuid"`
}

func (s *Server) handleUploadPhoto(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	// Parse inspection ID from form
	inspectionIDStr := c.FormValue("inspection_id")
	if inspectionIDStr == "" {
		return aletheia.Invalid("inspection_id is required")
	}

	inspectionID, err := parseUUID(inspectionIDStr)
	if err != nil {
		return err
	}

	// Verify inspection exists
	_, err = s.inspectionService.FindInspectionByID(ctx, inspectionID)
	if err != nil {
		return err
	}

	// Get uploaded file
	file, err := c.FormFile("image")
	if err != nil {
		return aletheia.Invalid("image file is required")
	}

	// Check file size
	if file.Size > maxPhotoSize {
		return aletheia.Invalid("image file exceeds maximum size of 5MB")
	}

	// Validate content type
	contentType := file.Header.Get("Content-Type")
	if !isAllowedImageType(contentType) {
		return aletheia.Invalid("invalid image type, must be JPEG, PNG, or WebP")
	}

	// Open file for reading
	src, err := file.Open()
	if err != nil {
		return aletheia.Internal("Failed to read uploaded file", err)
	}
	defer src.Close()

	// Generate storage path
	photoID := uuid.New()
	storagePath := "photos/" + inspectionID.String() + "/" + photoID.String()

	// Upload to storage
	storageURL, err := s.fileStorage.Upload(ctx, storagePath, src, contentType)
	if err != nil {
		s.log(c).Error("failed to upload photo", slog.String("error", err.Error()))
		return aletheia.Internal("Failed to upload photo", err)
	}

	// Create photo record
	photo := &aletheia.Photo{
		ID:           photoID,
		InspectionID: inspectionID,
		StorageURL:   storageURL,
	}

	if err := s.photoService.CreatePhoto(ctx, photo); err != nil {
		// Clean up uploaded file on error
		_ = s.fileStorage.Delete(ctx, storagePath)
		return err
	}

	s.log(c).Info("photo uploaded",
		slog.String("photo_id", photo.ID.String()),
		slog.String("inspection_id", inspectionID.String()),
	)

	return RespondCreated(c, photo)
}

func (s *Server) handleGetPhoto(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	photoID, err := requireUUIDParam(c, "id")
	if err != nil {
		return err
	}

	photo, err := s.photoService.FindPhotoWithViolations(ctx, photoID)
	if err != nil {
		return err
	}

	return RespondOK(c, photo)
}

func (s *Server) handleListPhotos(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	inspectionID, err := requireUUIDParam(c, "inspectionId")
	if err != nil {
		return err
	}

	filter := aletheia.PhotoFilter{
		InspectionID: &inspectionID,
		Limit:        100,
	}

	photos, total, err := s.photoService.FindPhotos(ctx, filter)
	if err != nil {
		return err
	}

	return RespondOK(c, map[string]interface{}{
		"photos": photos,
		"total":  total,
	})
}

func (s *Server) handleDeletePhoto(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	photoID, err := requireUUIDParam(c, "id")
	if err != nil {
		return err
	}

	// Get photo to find storage path
	photo, err := s.photoService.FindPhotoByID(ctx, photoID)
	if err != nil {
		return err
	}

	// Delete from database first
	if err := s.photoService.DeletePhoto(ctx, photoID); err != nil {
		return err
	}

	// Delete from storage (best effort)
	if err := s.fileStorage.Delete(ctx, photo.StorageURL); err != nil {
		s.log(c).Error("failed to delete photo from storage",
			slog.String("photo_id", photoID.String()),
			slog.String("error", err.Error()),
		)
	}

	s.log(c).Info("photo deleted", slog.String("photo_id", photoID.String()))

	return c.NoContent(http.StatusNoContent)
}

// AnalyzePhotoRequest is the request payload for analyzing a photo.
type AnalyzePhotoRequest struct {
	PhotoID string `json:"photo_id" form:"photo_id" validate:"required,uuid"`
}

func (s *Server) handleAnalyzePhoto(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	var req AnalyzePhotoRequest
	if err := bind(c, &req); err != nil {
		return err
	}

	photoID, err := parseUUID(req.PhotoID)
	if err != nil {
		return err
	}

	// Verify photo exists
	photo, err := s.photoService.FindPhotoByID(ctx, photoID)
	if err != nil {
		return err
	}

	// Get organization ID for rate limiting
	inspection, err := s.inspectionService.FindInspectionByID(ctx, photo.InspectionID)
	if err != nil {
		return err
	}

	project, err := s.projectService.FindProjectByID(ctx, inspection.ProjectID)
	if err != nil {
		return err
	}

	// Enqueue analysis job
	if s.queue == nil {
		return aletheia.Internal("Queue service not available", nil)
	}

	// Create job with payload
	payload, _ := json.Marshal(map[string]interface{}{
		"photo_id": photoID.String(),
	})

	job := &aletheia.Job{
		ID:             uuid.New(),
		QueueName:      aletheia.QueueDefault,
		JobType:        aletheia.JobTypePhotoAnalysis,
		OrganizationID: project.OrganizationID,
		Payload:        payload,
		Status:         aletheia.JobStatusPending,
		MaxAttempts:    3,
	}

	if err := s.queue.Enqueue(ctx, job); err != nil {
		s.log(c).Error("failed to enqueue photo analysis", slog.String("error", err.Error()))
		return aletheia.Internal("Failed to queue analysis", err)
	}

	s.log(c).Info("photo analysis queued",
		slog.String("photo_id", photoID.String()),
		slog.String("job_id", job.ID.String()),
	)

	return RespondOK(c, map[string]interface{}{
		"job_id":   job.ID.String(),
		"photo_id": photoID.String(),
		"status":   "queued",
	})
}

func (s *Server) handleGetPhotoAnalysisStatus(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	jobID, err := requireUUIDParam(c, "jobId")
	if err != nil {
		return err
	}

	if s.queue == nil {
		return aletheia.Internal("Queue service not available", nil)
	}

	job, err := s.queue.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	return RespondOK(c, map[string]interface{}{
		"job_id":       job.ID.String(),
		"status":       string(job.Status),
		"result":       job.Result,
		"error":        job.ErrorMessage,
		"created_at":   job.CreatedAt,
		"completed_at": job.CompletedAt,
	})
}

// Helper functions

func isAllowedImageType(contentType string) bool {
	switch contentType {
	case "image/jpeg", "image/png", "image/webp":
		return true
	default:
		return false
	}
}
