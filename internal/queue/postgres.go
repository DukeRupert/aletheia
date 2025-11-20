package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresQueue implements the Queue interface using PostgreSQL
type PostgresQueue struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
	config Config
}

// NewPostgresQueue creates a new PostgreSQL-backed queue
func NewPostgresQueue(pool *pgxpool.Pool, logger *slog.Logger, config Config) *PostgresQueue {
	return &PostgresQueue{
		pool:   pool,
		logger: logger,
		config: config,
	}
}

// Enqueue adds a new job to the queue
func (q *PostgresQueue) Enqueue(ctx context.Context, queueName, jobType string, organizationID uuid.UUID, payload map[string]interface{}, opts *EnqueueOptions) (*Job, error) {
	if opts == nil {
		opts = DefaultEnqueueOptions()
	}

	// Convert payload to JSONB
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Calculate scheduled time
	scheduledAt := time.Now()
	if opts.ScheduledAt != nil {
		scheduledAt = *opts.ScheduledAt
	} else if opts.Delay > 0 {
		scheduledAt = time.Now().Add(opts.Delay)
	}

	query := `
		INSERT INTO jobs (
			queue_name, job_type, organization_id, payload,
			priority, max_attempts, scheduled_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`

	var jobID uuid.UUID
	var createdAt time.Time

	err = q.pool.QueryRow(ctx, query,
		queueName, jobType, organizationID, payloadJSON,
		opts.Priority, opts.MaxAttempts, scheduledAt,
	).Scan(&jobID, &createdAt)

	if err != nil {
		return nil, fmt.Errorf("failed to enqueue job: %w", err)
	}

	job := &Job{
		ID:             jobID,
		QueueName:      queueName,
		JobType:        jobType,
		OrganizationID: organizationID,
		Payload:        payload,
		Status:         JobStatusPending,
		Priority:       opts.Priority,
		MaxAttempts:    opts.MaxAttempts,
		AttemptCount:   0,
		ScheduledAt:    scheduledAt,
		CreatedAt:      createdAt,
	}

	q.logger.Debug("job enqueued",
		slog.String("job_id", jobID.String()),
		slog.String("queue", queueName),
		slog.String("type", jobType),
	)

	return job, nil
}

// Dequeue retrieves and locks the next available job
func (q *PostgresQueue) Dequeue(ctx context.Context, workerID string, opts *DequeueOptions) (*Job, error) {
	if opts == nil {
		opts = DefaultDequeueOptions()
	}

	// Build queue filter
	queueFilter := ""
	if len(opts.QueueNames) > 0 {
		queueFilter = "AND queue_name = ANY($2)"
	}

	query := fmt.Sprintf(`
		UPDATE jobs
		SET
			status = 'processing',
			started_at = NOW(),
			attempt_count = attempt_count + 1,
			worker_id = $1
		WHERE id = (
			SELECT id
			FROM jobs
			WHERE status = 'pending'
			  AND scheduled_at <= NOW()
			  %s
			ORDER BY priority DESC, created_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING
			id, queue_name, job_type, organization_id, payload,
			status, priority, max_attempts, attempt_count,
			scheduled_at, created_at, started_at, completed_at,
			result, error_message, worker_id
	`, queueFilter)

	var row pgx.Row
	if len(opts.QueueNames) > 0 {
		row = q.pool.QueryRow(ctx, query, workerID, opts.QueueNames)
	} else {
		row = q.pool.QueryRow(ctx, query, workerID)
	}

	job, err := q.scanJob(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // No jobs available
		}
		return nil, fmt.Errorf("failed to dequeue job: %w", err)
	}

	q.logger.Debug("job dequeued",
		slog.String("job_id", job.ID.String()),
		slog.String("queue", job.QueueName),
		slog.String("worker", workerID),
	)

	return job, nil
}

// Complete marks a job as successfully completed
func (q *PostgresQueue) Complete(ctx context.Context, jobID uuid.UUID, result map[string]interface{}) error {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	query := `
		UPDATE jobs
		SET
			status = 'completed',
			completed_at = NOW(),
			result = $1
		WHERE id = $2
	`

	_, err = q.pool.Exec(ctx, query, resultJSON, jobID)
	if err != nil {
		return fmt.Errorf("failed to complete job: %w", err)
	}

	q.logger.Debug("job completed",
		slog.String("job_id", jobID.String()),
	)

	return nil
}

// Fail marks a job as failed and schedules retry if attempts remain
func (q *PostgresQueue) Fail(ctx context.Context, jobID uuid.UUID, errMsg string) error {
	query := `
		UPDATE jobs
		SET
			status = CASE
				WHEN attempt_count >= max_attempts THEN 'failed'
				ELSE 'pending'
			END,
			error_message = $1,
			scheduled_at = CASE
				WHEN attempt_count < max_attempts
				THEN NOW() + (INTERVAL '1 minute' * POW(2, attempt_count))
				ELSE scheduled_at
			END,
			completed_at = CASE
				WHEN attempt_count >= max_attempts THEN NOW()
				ELSE NULL
			END
		WHERE id = $2
		RETURNING status, attempt_count, max_attempts
	`

	var status string
	var attemptCount, maxAttempts int

	err := q.pool.QueryRow(ctx, query, errMsg, jobID).Scan(&status, &attemptCount, &maxAttempts)
	if err != nil {
		return fmt.Errorf("failed to mark job as failed: %w", err)
	}

	if status == string(JobStatusFailed) {
		q.logger.Warn("job permanently failed",
			slog.String("job_id", jobID.String()),
			slog.Int("attempts", attemptCount),
			slog.String("error", errMsg),
		)
	} else {
		backoff := time.Duration(1<<uint(attemptCount)) * time.Minute
		q.logger.Debug("job failed, will retry",
			slog.String("job_id", jobID.String()),
			slog.Int("attempt", attemptCount),
			slog.Int("max_attempts", maxAttempts),
			slog.Duration("retry_in", backoff),
		)
	}

	return nil
}

// Delete removes a job from the queue
func (q *PostgresQueue) Delete(ctx context.Context, jobID uuid.UUID) error {
	query := `DELETE FROM jobs WHERE id = $1`

	_, err := q.pool.Exec(ctx, query, jobID)
	if err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	return nil
}

// GetJob retrieves a job by ID
func (q *PostgresQueue) GetJob(ctx context.Context, jobID uuid.UUID) (*Job, error) {
	query := `
		SELECT
			id, queue_name, job_type, organization_id, payload,
			status, priority, max_attempts, attempt_count,
			scheduled_at, created_at, started_at, completed_at,
			result, error_message, worker_id
		FROM jobs
		WHERE id = $1
	`

	job, err := q.scanJob(q.pool.QueryRow(ctx, query, jobID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("job not found: %s", jobID)
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	return job, nil
}

// ListJobs retrieves jobs with filtering
func (q *PostgresQueue) ListJobs(ctx context.Context, filter JobFilter) ([]*Job, error) {
	query := `
		SELECT
			id, queue_name, job_type, organization_id, payload,
			status, priority, max_attempts, attempt_count,
			scheduled_at, created_at, started_at, completed_at,
			result, error_message, worker_id
		FROM jobs
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	if filter.QueueName != nil {
		query += fmt.Sprintf(" AND queue_name = $%d", argPos)
		args = append(args, *filter.QueueName)
		argPos++
	}

	if filter.JobType != nil {
		query += fmt.Sprintf(" AND job_type = $%d", argPos)
		args = append(args, *filter.JobType)
		argPos++
	}

	if filter.OrganizationID != nil {
		query += fmt.Sprintf(" AND organization_id = $%d", argPos)
		args = append(args, *filter.OrganizationID)
		argPos++
	}

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, string(*filter.Status))
		argPos++
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, filter.Limit)
		argPos++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argPos)
		args = append(args, filter.Offset)
		argPos++
	}

	rows, err := q.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		job, err := q.scanJob(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job: %w", err)
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// GetQueueStats returns statistics for a queue
func (q *PostgresQueue) GetQueueStats(ctx context.Context, queueName string) (*QueueStats, error) {
	query := `
		SELECT
			status,
			COUNT(*) as count,
			AVG(EXTRACT(EPOCH FROM (COALESCE(completed_at, NOW()) - started_at))) as avg_duration
		FROM jobs
		WHERE queue_name = $1
		  AND created_at > NOW() - INTERVAL '24 hours'
		GROUP BY status
	`

	rows, err := q.pool.Query(ctx, query, queueName)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue stats: %w", err)
	}
	defer rows.Close()

	stats := &QueueStats{
		QueueName: queueName,
	}

	var totalDuration float64
	var totalCompleted int

	for rows.Next() {
		var status string
		var count int
		var avgDuration pgtype.Float8

		if err := rows.Scan(&status, &count, &avgDuration); err != nil {
			return nil, fmt.Errorf("failed to scan stats: %w", err)
		}

		switch JobStatus(status) {
		case JobStatusPending:
			stats.PendingJobs = count
		case JobStatusProcessing:
			stats.ProcessingJobs = count
		case JobStatusCompleted:
			stats.CompletedJobs = count
			if avgDuration.Valid {
				totalDuration += avgDuration.Float64
				totalCompleted = count
			}
		case JobStatusFailed:
			stats.FailedJobs = count
		}
	}

	if totalCompleted > 0 {
		stats.AvgProcessingTime = time.Duration(totalDuration/float64(totalCompleted)) * time.Second
	}

	return stats, nil
}

// CanProcessJob checks if an organization can process more jobs (rate limiting)
func (q *PostgresQueue) CanProcessJob(ctx context.Context, organizationID uuid.UUID, queueName string) (bool, error) {
	if !q.config.EnableRateLimiting {
		return true, nil
	}

	query := `
		SELECT
			max_concurrent_jobs - COALESCE((
				SELECT COUNT(*)
				FROM jobs
				WHERE organization_id = $1
				  AND queue_name = $2
				  AND status = 'processing'
			), 0) as available_slots,

			max_jobs_per_hour - jobs_in_current_window as remaining_quota
		FROM organization_rate_limits
		WHERE organization_id = $1 AND queue_name = $2
	`

	var availableSlots, remainingQuota int
	err := q.pool.QueryRow(ctx, query, organizationID, queueName).Scan(&availableSlots, &remainingQuota)

	if err != nil {
		if err == pgx.ErrNoRows {
			// No rate limit configured, use defaults
			return true, nil
		}
		return false, fmt.Errorf("failed to check rate limits: %w", err)
	}

	canProcess := availableSlots > 0 && remainingQuota > 0

	if !canProcess {
		q.logger.Debug("rate limit exceeded",
			slog.String("org_id", organizationID.String()),
			slog.String("queue", queueName),
			slog.Int("available_slots", availableSlots),
			slog.Int("remaining_quota", remainingQuota),
		)
	}

	return canProcess, nil
}

// RecordJobProcessed increments the job counter for rate limiting
func (q *PostgresQueue) RecordJobProcessed(ctx context.Context, organizationID uuid.UUID, queueName string) error {
	if !q.config.EnableRateLimiting {
		return nil
	}

	query := `
		INSERT INTO organization_rate_limits (
			organization_id, queue_name,
			max_jobs_per_hour, max_concurrent_jobs,
			jobs_in_current_window, window_start
		) VALUES ($1, $2, $3, $4, 1, NOW())
		ON CONFLICT (organization_id, queue_name)
		DO UPDATE SET
			jobs_in_current_window = CASE
				WHEN organization_rate_limits.window_start < NOW() - INTERVAL '1 hour'
				THEN 1
				ELSE organization_rate_limits.jobs_in_current_window + 1
			END,
			window_start = CASE
				WHEN organization_rate_limits.window_start < NOW() - INTERVAL '1 hour'
				THEN NOW()
				ELSE organization_rate_limits.window_start
			END
	`

	_, err := q.pool.Exec(ctx, query,
		organizationID, queueName,
		q.config.DefaultMaxJobsPerHour,
		q.config.DefaultMaxConcurrentJobs,
	)

	if err != nil {
		return fmt.Errorf("failed to record job processed: %w", err)
	}

	return nil
}

// Close gracefully shuts down the queue
func (q *PostgresQueue) Close() error {
	q.logger.Info("closing PostgreSQL queue")
	q.pool.Close()
	return nil
}

// scanJob is a helper to scan a job from a row
func (q *PostgresQueue) scanJob(row pgx.Row) (*Job, error) {
	var job Job
	var payloadJSON []byte
	var resultJSON []byte
	var startedAt, completedAt pgtype.Timestamptz
	var workerID pgtype.Text
	var errorMessage pgtype.Text

	err := row.Scan(
		&job.ID, &job.QueueName, &job.JobType, &job.OrganizationID,
		&payloadJSON, &job.Status, &job.Priority, &job.MaxAttempts,
		&job.AttemptCount, &job.ScheduledAt, &job.CreatedAt,
		&startedAt, &completedAt, &resultJSON, &errorMessage, &workerID,
	)

	if err != nil {
		return nil, err
	}

	// Unmarshal payload
	if err := json.Unmarshal(payloadJSON, &job.Payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Unmarshal result if present
	if len(resultJSON) > 0 {
		if err := json.Unmarshal(resultJSON, &job.Result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal result: %w", err)
		}
	}

	// Handle nullable fields
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	if workerID.Valid {
		job.WorkerID = workerID.String
	}
	if errorMessage.Valid {
		job.ErrorMessage = errorMessage.String
	}

	return &job, nil
}

// Verify PostgresQueue implements Queue interface
var _ Queue = (*PostgresQueue)(nil)
