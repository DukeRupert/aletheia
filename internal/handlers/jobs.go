package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/dukerupert/aletheia/internal/database"
	"github.com/dukerupert/aletheia/internal/queue"
	"github.com/dukerupert/aletheia/internal/session"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

// JobHandler handles job status HTTP requests
type JobHandler struct {
	db     *database.Queries
	pool   *pgxpool.Pool
	queue  queue.Queue
	logger *slog.Logger
}

// NewJobHandler creates a new job handler
func NewJobHandler(pool *pgxpool.Pool, db *database.Queries, q queue.Queue, logger *slog.Logger) *JobHandler {
	return &JobHandler{
		db:     db,
		pool:   pool,
		queue:  q,
		logger: logger,
	}
}

// JobStatusData represents job status for template rendering
type JobStatusData struct {
	JobCount       int
	JobType        string
	EstimatedTime  int
	CompletedCount int
	ShowDetails    bool
	Jobs           []JobInfo
}

// JobInfo represents individual job information
type JobInfo struct {
	ID          string
	Status      string
	Description string
}

// GetJobStatus godoc
// @Summary Get active job status
// @Description Get the status of active background jobs for the current user's organization
// @Tags jobs
// @Accept json
// @Produce json,html
// @Success 200 {object} JobStatusData
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/jobs/status [get]
func (h *JobHandler) GetJobStatus(c echo.Context) error {
	// Get user from session
	_, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session", slog.String("endpoint", "GetJobStatus"))
		if c.Request().Header.Get("HX-Request") == "true" {
			// Return empty div for HTMX
			return c.HTML(http.StatusOK, `<div id="job-status-bar" class="hidden"></div>`)
		}
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	// TODO: Implement GetActiveJobsByOrganization in queue interface
	// For now, just return 0 active jobs
	activeJobs := []interface{}{}

	// Future implementation:
	// Get user's primary organization (assume first one for now)
	// orgMemberships, err := h.db.ListUserOrganizations(ctx, uuidToPgUUID(userID))
	// if err != nil || len(orgMemberships) == 0 {
	// 	...
	// }
	// orgID := uuid.UUID(orgMemberships[0].OrganizationID.Bytes)
	// activeJobs, err := h.queue.GetActiveJobsByOrganization(ctx, orgID)
	// if err != nil {
	// 	...
	// }

	// Calculate statistics
	jobCount := len(activeJobs)

	// Prepare response data (all zeros since we have no active jobs for now)
	data := JobStatusData{
		JobCount:       jobCount,
		JobType:        "",
		EstimatedTime:  0,
		CompletedCount: 0,
		ShowDetails:    false,
		Jobs:           []JobInfo{},
	}

	// If HTMX request, render HTML
	if c.Request().Header.Get("HX-Request") == "true" {
		return c.Render(http.StatusOK, "job-status", data)
	}

	// Otherwise return JSON
	return c.JSON(http.StatusOK, data)
}

// GetActiveJobsByOrganization is a helper method to get active jobs
// This extends the queue interface if not already implemented
func (h *JobHandler) GetActiveJobsByOrganization(ctx context.Context, orgID uuid.UUID) ([]queue.Job, error) {
	// Query jobs table for active jobs belonging to this organization
	// This assumes the jobs table has an organization_id column
	rows, err := h.pool.Query(ctx, `
		SELECT id, queue_name, job_type, organization_id, payload, status,
		       priority, max_attempts, attempt_count, scheduled_at,
		       started_at, completed_at, error_message, result, created_at
		FROM jobs
		WHERE organization_id = $1
		  AND status IN ('pending', 'processing')
		ORDER BY created_at DESC
		LIMIT 50
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := []queue.Job{}
	for rows.Next() {
		var job queue.Job
		err := rows.Scan(
			&job.ID,
			&job.QueueName,
			&job.JobType,
			&job.OrganizationID,
			&job.Payload,
			&job.Status,
			&job.Priority,
			&job.MaxAttempts,
			&job.AttemptCount,
			&job.ScheduledAt,
			&job.StartedAt,
			&job.CompletedAt,
			&job.ErrorMessage,
			&job.Result,
			&job.CreatedAt,
		)
		if err != nil {
			h.logger.Error("failed to scan job row", slog.String("err", err.Error()))
			continue
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// CancelJob godoc
// @Summary Cancel a job
// @Description Cancel a pending or processing job
// @Tags jobs
// @Accept json
// @Produce json
// @Param job_id path string true "Job ID"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/jobs/{job_id}/cancel [post]
func (h *JobHandler) CancelJob(c echo.Context) error {
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	jobIDStr := c.Param("job_id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid job_id format")
	}

	// Get job to verify ownership
	job, err := h.queue.GetJob(ctx, jobID)
	if err != nil {
		h.logger.Error("job not found",
			slog.String("job_id", jobID.String()),
			slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusNotFound, "Job not found")
	}

	// Authorization: verify user has access to this job's organization
	userID, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session", slog.String("job_id", jobID.String()))
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	_, err = requireOrganizationMembership(ctx, h.pool, h.logger, userID, uuidToPgUUID(job.OrganizationID))
	if err != nil {
		return err
	}

	// Only allow cancelling pending or processing jobs
	if job.Status != "pending" && job.Status != "processing" {
		return echo.NewHTTPError(http.StatusConflict,
			"Can only cancel pending or processing jobs")
	}

	// Update job status to failed with cancellation message
	_, err = h.pool.Exec(ctx, `
		UPDATE jobs
		SET status = 'failed',
		    error_message = 'Cancelled by user',
		    completed_at = $1
		WHERE id = $2
	`, time.Now(), jobID)

	if err != nil {
		h.logger.Error("failed to cancel job",
			slog.String("job_id", jobID.String()),
			slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to cancel job")
	}

	h.logger.Info("job cancelled",
		slog.String("job_id", jobID.String()),
		slog.String("user_id", userID.String()))

	return c.NoContent(http.StatusNoContent)
}
