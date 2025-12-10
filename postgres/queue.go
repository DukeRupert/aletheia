package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/dukerupert/aletheia"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Compile-time interface check
var _ aletheia.Queue = (*Queue)(nil)

// NewQueue creates a queue implementation based on the configuration.
func NewQueue(pool *pgxpool.Pool, logger *slog.Logger, cfg aletheia.QueueConfig) aletheia.Queue {
	return &Queue{
		pool:   pool,
		logger: logger,
		cfg:    cfg,
	}
}

// Queue is a PostgreSQL-backed job queue implementation.
type Queue struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
	cfg    aletheia.QueueConfig
}

// Enqueue adds a job to the queue.
func (q *Queue) Enqueue(ctx context.Context, job *aletheia.Job, opts ...aletheia.EnqueueOption) error {
	// Set defaults
	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}
	if job.Status == "" {
		job.Status = aletheia.JobStatusPending
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}
	if job.ScheduledAt.IsZero() {
		job.ScheduledAt = time.Now()
	}
	if job.MaxAttempts == 0 {
		job.MaxAttempts = 3
	}

	query := `
		INSERT INTO jobs (
			id, queue_name, job_type, organization_id, payload, status,
			priority, max_attempts, attempt_count, scheduled_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := q.pool.Exec(ctx, query,
		job.ID,
		job.QueueName,
		job.JobType,
		job.OrganizationID,
		job.Payload,
		job.Status,
		job.Priority,
		job.MaxAttempts,
		job.AttemptCount,
		job.ScheduledAt,
		job.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("enqueueing job: %w", err)
	}

	q.logger.Debug("job enqueued",
		slog.String("job_id", job.ID.String()),
		slog.String("job_type", job.JobType),
		slog.String("queue", job.QueueName))

	return nil
}

// Dequeue retrieves the next available job from a queue.
func (q *Queue) Dequeue(ctx context.Context, queueName string) (*aletheia.Job, error) {
	query := `
		UPDATE jobs
		SET status = $1, started_at = $2, attempt_count = attempt_count + 1
		WHERE id = (
			SELECT id FROM jobs
			WHERE queue_name = $3
			AND status = $4
			AND scheduled_at <= $5
			ORDER BY priority DESC, created_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING id, queue_name, job_type, organization_id, payload, status,
			priority, max_attempts, attempt_count, scheduled_at, created_at,
			started_at, completed_at, result, error_message, worker_id
	`

	now := time.Now()
	row := q.pool.QueryRow(ctx, query,
		aletheia.JobStatusRunning,
		now,
		queueName,
		aletheia.JobStatusPending,
		now,
	)

	job := &aletheia.Job{}
	var completedAt sql.NullTime
	var result, errorMessage, workerID sql.NullString

	err := row.Scan(
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
		&job.CreatedAt,
		&job.StartedAt,
		&completedAt,
		&result,
		&errorMessage,
		&workerID,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil // No jobs available
		}
		return nil, fmt.Errorf("dequeuing job: %w", err)
	}

	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	if result.Valid {
		job.Result = []byte(result.String)
	}
	if errorMessage.Valid {
		job.ErrorMessage = errorMessage.String
	}
	if workerID.Valid {
		job.WorkerID = workerID.String
	}

	return job, nil
}

// Complete marks a job as completed.
func (q *Queue) Complete(ctx context.Context, jobID uuid.UUID, result []byte) error {
	query := `
		UPDATE jobs
		SET status = $1, completed_at = $2, result = $3
		WHERE id = $4
	`

	_, err := q.pool.Exec(ctx, query,
		aletheia.JobStatusCompleted,
		time.Now(),
		result,
		jobID,
	)
	if err != nil {
		return fmt.Errorf("completing job: %w", err)
	}

	q.logger.Debug("job completed", slog.String("job_id", jobID.String()))
	return nil
}

// Fail marks a job as failed.
func (q *Queue) Fail(ctx context.Context, jobID uuid.UUID, errMsg string) error {
	query := `
		UPDATE jobs
		SET status = $1, completed_at = $2, error_message = $3
		WHERE id = $4
	`

	_, err := q.pool.Exec(ctx, query,
		aletheia.JobStatusFailed,
		time.Now(),
		errMsg,
		jobID,
	)
	if err != nil {
		return fmt.Errorf("failing job: %w", err)
	}

	q.logger.Debug("job failed",
		slog.String("job_id", jobID.String()),
		slog.String("error", errMsg))
	return nil
}

// GetJob retrieves a job by its ID.
func (q *Queue) GetJob(ctx context.Context, jobID uuid.UUID) (*aletheia.Job, error) {
	query := `
		SELECT id, queue_name, job_type, organization_id, payload, status,
			priority, max_attempts, attempt_count, scheduled_at, created_at,
			started_at, completed_at, result, error_message, worker_id
		FROM jobs
		WHERE id = $1
	`

	row := q.pool.QueryRow(ctx, query, jobID)

	job := &aletheia.Job{}
	var startedAt, completedAt sql.NullTime
	var result, errorMessage, workerID sql.NullString

	err := row.Scan(
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
		&job.CreatedAt,
		&startedAt,
		&completedAt,
		&result,
		&errorMessage,
		&workerID,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, aletheia.NotFound("Job not found")
		}
		return nil, fmt.Errorf("getting job: %w", err)
	}

	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	if result.Valid {
		job.Result = []byte(result.String)
	}
	if errorMessage.Valid {
		job.ErrorMessage = errorMessage.String
	}
	if workerID.Valid {
		job.WorkerID = workerID.String
	}

	return job, nil
}

// CancelJob cancels a pending job.
func (q *Queue) CancelJob(ctx context.Context, jobID uuid.UUID) error {
	query := `
		UPDATE jobs
		SET status = $1, completed_at = $2
		WHERE id = $3 AND status = $4
	`

	result, err := q.pool.Exec(ctx, query,
		aletheia.JobStatusCancelled,
		time.Now(),
		jobID,
		aletheia.JobStatusPending,
	)
	if err != nil {
		return fmt.Errorf("cancelling job: %w", err)
	}

	if result.RowsAffected() == 0 {
		return aletheia.Invalid("Can only cancel pending jobs")
	}

	q.logger.Debug("job cancelled", slog.String("job_id", jobID.String()))
	return nil
}

// GetPendingJobs retrieves pending jobs for an organization.
func (q *Queue) GetPendingJobs(ctx context.Context, orgID uuid.UUID, queueName string) ([]*aletheia.Job, error) {
	query := `
		SELECT id, queue_name, job_type, organization_id, payload, status,
			priority, max_attempts, attempt_count, scheduled_at, created_at,
			started_at, completed_at, result, error_message, worker_id
		FROM jobs
		WHERE organization_id = $1 AND queue_name = $2 AND status = $3
		ORDER BY priority DESC, created_at ASC
	`

	rows, err := q.pool.Query(ctx, query, orgID, queueName, aletheia.JobStatusPending)
	if err != nil {
		return nil, fmt.Errorf("querying pending jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*aletheia.Job
	for rows.Next() {
		job := &aletheia.Job{}
		var startedAt, completedAt sql.NullTime
		var result, errorMessage, workerID sql.NullString

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
			&job.CreatedAt,
			&startedAt,
			&completedAt,
			&result,
			&errorMessage,
			&workerID,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning job: %w", err)
		}

		if startedAt.Valid {
			job.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			job.CompletedAt = &completedAt.Time
		}
		if result.Valid {
			job.Result = []byte(result.String)
		}
		if errorMessage.Valid {
			job.ErrorMessage = errorMessage.String
		}
		if workerID.Valid {
			job.WorkerID = workerID.String
		}

		jobs = append(jobs, job)
	}

	return jobs, nil
}
