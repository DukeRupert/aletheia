package handlers

import (
	"log/slog"
	"net/http"

	"github.com/dukerupert/aletheia/internal/database"
	"github.com/dukerupert/aletheia/internal/queue"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
)

// PhotoHandler handles photo-related HTTP requests
type PhotoHandler struct {
	db     *database.Queries
	queue  queue.Queue
	logger *slog.Logger
}

// NewPhotoHandler creates a new photo handler
func NewPhotoHandler(db *database.Queries, q queue.Queue, logger *slog.Logger) *PhotoHandler {
	return &PhotoHandler{
		db:     db,
		queue:  q,
		logger: logger,
	}
}

// AnalyzePhotoRequest is the request body for triggering photo analysis
type AnalyzePhotoRequest struct {
	PhotoID string `json:"photo_id" validate:"required,uuid"`
}

// AnalyzePhotoResponse is the response for a photo analysis request
type AnalyzePhotoResponse struct {
	JobID   string `json:"job_id"`
	PhotoID string `json:"photo_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// AnalyzePhoto godoc
// @Summary Trigger AI analysis on a photo
// @Description Enqueues a photo for AI analysis to detect safety violations
// @Tags photos
// @Accept json
// @Produce json
// @Param request body AnalyzePhotoRequest true "Photo Analysis Request"
// @Success 202 {object} AnalyzePhotoResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/photos/analyze [post]
func (h *PhotoHandler) AnalyzePhoto(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse request
	var req AnalyzePhotoRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	photoID, err := uuid.Parse(req.PhotoID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid photo_id format")
	}

	// Verify photo exists
	photo, err := h.db.GetPhoto(ctx, pgtype.UUID{Bytes: photoID, Valid: true})
	if err != nil {
		h.logger.Error("photo not found",
			slog.String("photo_id", photoID.String()),
			slog.String("error", err.Error()),
		)
		return echo.NewHTTPError(http.StatusNotFound, "Photo not found")
	}

	// Get the current user's organization ID from session
	// TODO: Add authorization check to ensure user has access to this photo's organization
	// userOrgID := c.Get("organization_id").(uuid.UUID)

	// Fetch inspection to get organization ID
	inspection, err := h.db.GetInspection(ctx, photo.InspectionID)
	if err != nil {
		h.logger.Error("inspection not found",
			slog.String("inspection_id", photo.InspectionID.String()),
			slog.String("error", err.Error()),
		)
		return echo.NewHTTPError(http.StatusNotFound, "Inspection not found")
	}

	// Fetch project to get organization ID
	project, err := h.db.GetProject(ctx, inspection.ProjectID)
	if err != nil {
		h.logger.Error("project not found",
			slog.String("project_id", inspection.ProjectID.String()),
			slog.String("error", err.Error()),
		)
		return echo.NewHTTPError(http.StatusNotFound, "Project not found")
	}

	// Enqueue photo analysis job
	payload := map[string]interface{}{
		"photo_id":      photoID.String(),
		"inspection_id": photo.InspectionID.String(),
	}

	job, err := h.queue.Enqueue(
		ctx,
		"photo_analysis",                           // queue name
		"analyze_photo",                            // job type
		uuid.UUID(project.OrganizationID.Bytes), // organization ID for rate limiting
		payload,
		&queue.EnqueueOptions{
			Priority:    5, // Medium priority
			MaxAttempts: 3,
		},
	)

	if err != nil {
		h.logger.Error("failed to enqueue photo analysis job",
			slog.String("photo_id", photoID.String()),
			slog.String("error", err.Error()),
		)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to enqueue analysis job")
	}

	h.logger.Info("photo analysis job enqueued",
		slog.String("photo_id", photoID.String()),
		slog.String("job_id", job.ID.String()),
	)

	return c.JSON(http.StatusAccepted, AnalyzePhotoResponse{
		JobID:   job.ID.String(),
		PhotoID: photoID.String(),
		Status:  "queued",
		Message: "Photo analysis has been queued for processing",
	})
}

// GetPhotoAnalysisStatus godoc
// @Summary Get photo analysis job status
// @Description Get the status of a photo analysis job
// @Tags photos
// @Accept json
// @Produce json
// @Param job_id path string true "Job ID"
// @Success 200 {object} queue.Job
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/photos/analyze/{job_id} [get]
func (h *PhotoHandler) GetPhotoAnalysisStatus(c echo.Context) error {
	ctx := c.Request().Context()

	jobIDStr := c.Param("job_id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid job_id format")
	}

	// Get job from queue
	job, err := h.queue.GetJob(ctx, jobID)
	if err != nil {
		h.logger.Error("job not found",
			slog.String("job_id", jobID.String()),
			slog.String("error", err.Error()),
		)
		return echo.NewHTTPError(http.StatusNotFound, "Job not found")
	}

	return c.JSON(http.StatusOK, job)
}
