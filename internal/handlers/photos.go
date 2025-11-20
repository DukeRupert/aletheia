package handlers

import (
	"fmt"
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

	// Check if this is an HTMX request
	if c.Request().Header.Get("HX-Request") == "true" {
		// Return HTML for HTMX with polling
		displayURL := photo.StorageUrl
		if photo.ThumbnailUrl.Valid {
			displayURL = photo.ThumbnailUrl.String
		}

		html := `<div class="card" style="padding: var(--space-sm);"
			hx-get="/api/photos/analyze/` + job.ID.String() + `"
			hx-trigger="every 2s"
			hx-swap="outerHTML">
			<a href="` + photo.StorageUrl + `" target="_blank">
				<img src="` + displayURL + `" alt="Inspection photo" style="width: 100%; height: 200px; object-fit: cover; border-radius: 4px; margin-bottom: var(--space-sm);">
			</a>
			<div style="margin-bottom: var(--space-sm);">
				<p style="color: #666; font-size: 0.75rem; margin: 0;">` + photo.CreatedAt.Time.Format("Jan 2, 3:04 PM") + `</p>
			</div>
			<div style="padding: var(--space-sm); background: #dbeafe; border-left: 4px solid #3b82f6; border-radius: 4px; margin-bottom: var(--space-sm);">
				<p style="font-weight: 600; font-size: 0.875rem; color: #1e40af; margin: 0;">
					⏳ Analyzing photo for safety violations...
				</p>
			</div>
		</div>`
		return c.HTML(http.StatusAccepted, html)
	}

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

	// Check if this is an HTMX request
	if c.Request().Header.Get("HX-Request") == "true" {
		// Get photo_id from job payload
		photoIDStr, ok := job.Payload["photo_id"].(string)
		if !ok {
			return echo.NewHTTPError(http.StatusInternalServerError, "Invalid job payload")
		}
		photoID, err := uuid.Parse(photoIDStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Invalid photo_id in job payload")
		}

		// Get photo
		photo, err := h.db.GetPhoto(ctx, pgtype.UUID{Bytes: photoID, Valid: true})
		if err != nil {
			return echo.NewHTTPError(http.StatusNotFound, "Photo not found")
		}

		displayURL := photo.StorageUrl
		if photo.ThumbnailUrl.Valid {
			displayURL = photo.ThumbnailUrl.String
		}

		// If job is still pending or processing, continue polling
		if job.Status == "pending" || job.Status == "processing" {
			html := `<div class="card" style="padding: var(--space-sm);"
				hx-get="/api/photos/analyze/` + job.ID.String() + `"
				hx-trigger="every 2s"
				hx-swap="outerHTML">
				<a href="` + photo.StorageUrl + `" target="_blank">
					<img src="` + displayURL + `" alt="Inspection photo" style="width: 100%; height: 200px; object-fit: cover; border-radius: 4px; margin-bottom: var(--space-sm);">
				</a>
				<div style="margin-bottom: var(--space-sm);">
					<p style="color: #666; font-size: 0.75rem; margin: 0;">` + photo.CreatedAt.Time.Format("Jan 2, 3:04 PM") + `</p>
				</div>
				<div style="padding: var(--space-sm); background: #dbeafe; border-left: 4px solid #3b82f6; border-radius: 4px; margin-bottom: var(--space-sm);">
					<p style="font-weight: 600; font-size: 0.875rem; color: #1e40af; margin: 0;">
						⏳ Analyzing photo for safety violations...
					</p>
				</div>
			</div>`
			return c.HTML(http.StatusOK, html)
		}

		// If job failed, show error
		if job.Status == "failed" {
			errorMsg := "Analysis failed"
			if job.ErrorMessage != "" {
				errorMsg = job.ErrorMessage
			}
			html := `<div class="card" style="padding: var(--space-sm);">
				<a href="` + photo.StorageUrl + `" target="_blank">
					<img src="` + displayURL + `" alt="Inspection photo" style="width: 100%; height: 200px; object-fit: cover; border-radius: 4px; margin-bottom: var(--space-sm);">
				</a>
				<div style="margin-bottom: var(--space-sm);">
					<p style="color: #666; font-size: 0.75rem; margin: 0;">` + photo.CreatedAt.Time.Format("Jan 2, 3:04 PM") + `</p>
				</div>
				<div style="padding: var(--space-sm); background: #fef2f2; border-left: 4px solid #dc2626; border-radius: 4px; margin-bottom: var(--space-sm);">
					<p style="font-weight: 600; font-size: 0.875rem; color: #dc2626; margin: 0;">
						❌ Analysis failed: ` + errorMsg + `
					</p>
				</div>
				<div style="display: flex; gap: var(--space-xs); flex-wrap: wrap;">
					<button hx-post="/api/photos/analyze" hx-vals='{"photo_id": "` + photo.ID.String() + `"}' hx-target="closest .card" hx-swap="outerHTML" class="btn-primary" style="padding: 0.25rem 0.5rem; font-size: 0.75rem; flex: 1;">Retry</button>
					<button hx-delete="/api/photos/` + photo.ID.String() + `" hx-confirm="Are you sure you want to delete this photo?" hx-target="closest .card" hx-swap="outerHTML swap:1s" class="btn-secondary" style="padding: 0.25rem 0.5rem; font-size: 0.75rem;">Delete</button>
				</div>
			</div>`
			return c.HTML(http.StatusOK, html)
		}

		// Job completed successfully - get violations and render full card
		violations, err := h.db.ListDetectedViolations(ctx, photo.ID)
		if err != nil {
			h.logger.Error("failed to list violations",
				slog.String("photo_id", photoID.String()),
				slog.String("error", err.Error()),
			)
			violations = []database.DetectedViolation{}
		}

		// Build violations HTML
		var violationsHTML string
		if len(violations) > 0 {
			violationsHTML = `<div style="margin-bottom: var(--space-sm); padding: var(--space-sm); background: #fef2f2; border-left: 4px solid #dc2626; border-radius: 4px;">
				<p style="font-weight: 600; font-size: 0.875rem; color: #dc2626; margin-bottom: var(--space-xs);">
					` + fmt.Sprintf("%d", len(violations)) + ` Violation(s) Detected
				</p>`
			for _, v := range violations {
				severityBg := "#94a3b8"
				severityText := "white"
				switch v.Severity {
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

				// Convert pgtype.Numeric to float64
				confidenceFloat, _ := v.ConfidenceScore.Float64Value()
				confidence := fmt.Sprintf("%.0f", confidenceFloat.Float64*100)
				violationsHTML += `<div style="margin-bottom: var(--space-xs); padding: var(--space-xs); background: white; border-radius: 4px;">
					<div style="display: flex; justify-content: space-between; align-items: start; margin-bottom: var(--space-2xs);">
						<span style="padding: 0.125rem 0.375rem; border-radius: 3px; font-size: 0.7rem; font-weight: 600; background: ` + severityBg + `; color: ` + severityText + `;">
							` + string(v.Severity) + `
						</span>
						<span style="font-size: 0.7rem; color: #64748b;">
							` + confidence + `% confidence
						</span>
					</div>
					<p style="font-size: 0.8rem; color: #334155; margin: 0;">
						` + v.Description + `
					</p>`
				if v.Location.Valid {
					violationsHTML += `<p style="font-size: 0.7rem; color: #64748b; margin-top: var(--space-2xs); margin-bottom: 0;">
						Location: ` + v.Location.String + `
					</p>`
				}
				violationsHTML += `</div>`
			}
			violationsHTML += `</div>`
		}

		html := `<div class="card" style="padding: var(--space-sm);">
			<a href="` + photo.StorageUrl + `" target="_blank">
				<img src="` + displayURL + `" alt="Inspection photo" style="width: 100%; height: 200px; object-fit: cover; border-radius: 4px; margin-bottom: var(--space-sm);">
			</a>
			<div style="margin-bottom: var(--space-sm);">
				<p style="color: #666; font-size: 0.75rem; margin: 0;">` + photo.CreatedAt.Time.Format("Jan 2, 3:04 PM") + `</p>
			</div>
			` + violationsHTML + `
			<div style="display: flex; gap: var(--space-xs); flex-wrap: wrap;">
				<button hx-post="/api/photos/analyze" hx-vals='{"photo_id": "` + photo.ID.String() + `"}' hx-target="closest .card" hx-swap="outerHTML" class="btn-primary" style="padding: 0.25rem 0.5rem; font-size: 0.75rem; flex: 1;">Re-analyze</button>
				<button hx-delete="/api/photos/` + photo.ID.String() + `" hx-confirm="Are you sure you want to delete this photo?" hx-target="closest .card" hx-swap="outerHTML swap:1s" class="btn-secondary" style="padding: 0.25rem 0.5rem; font-size: 0.75rem;">Delete</button>
			</div>
		</div>`
		return c.HTML(http.StatusOK, html)
	}

	return c.JSON(http.StatusOK, job)
}
