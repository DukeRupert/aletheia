package aletheia

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Queue defines operations for a job queue.
type Queue interface {
	// Enqueue adds a job to the queue.
	Enqueue(ctx context.Context, job *Job, opts ...EnqueueOption) error

	// Dequeue retrieves the next available job from a queue.
	// Returns nil if no jobs are available.
	Dequeue(ctx context.Context, queueName string) (*Job, error)

	// Complete marks a job as completed with optional result data.
	Complete(ctx context.Context, jobID uuid.UUID, result []byte) error

	// Fail marks a job as failed with an error message.
	// The job may be retried based on its retry configuration.
	Fail(ctx context.Context, jobID uuid.UUID, errMsg string) error

	// GetJob retrieves a job by its ID.
	// Returns ENOTFOUND if the job does not exist.
	GetJob(ctx context.Context, jobID uuid.UUID) (*Job, error)

	// CancelJob cancels a pending job.
	// Returns EINVALID if the job is already running or completed.
	CancelJob(ctx context.Context, jobID uuid.UUID) error

	// GetPendingJobs retrieves pending jobs for an organization.
	GetPendingJobs(ctx context.Context, orgID uuid.UUID, queueName string) ([]*Job, error)
}

// Job represents a background job.
type Job struct {
	ID             uuid.UUID  `json:"id"`
	QueueName      string     `json:"queueName"`
	JobType        string     `json:"jobType"`
	OrganizationID uuid.UUID  `json:"organizationId"`
	Payload        []byte     `json:"payload"`
	Status         JobStatus  `json:"status"`
	Priority       int        `json:"priority"`
	MaxAttempts    int        `json:"maxAttempts"`
	AttemptCount   int        `json:"attemptCount"`
	ScheduledAt    time.Time  `json:"scheduledAt"`
	CreatedAt      time.Time  `json:"createdAt"`
	StartedAt      *time.Time `json:"startedAt,omitempty"`
	CompletedAt    *time.Time `json:"completedAt,omitempty"`
	Result         []byte     `json:"result,omitempty"`
	ErrorMessage   string     `json:"errorMessage,omitempty"`
	WorkerID       string     `json:"workerId,omitempty"`
}

// JobStatus represents the status of a job.
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// IsTerminal returns true if the job is in a terminal state.
func (s JobStatus) IsTerminal() bool {
	return s == JobStatusCompleted || s == JobStatusFailed || s == JobStatusCancelled
}

// Common job types.
const (
	JobTypePhotoAnalysis     = "photo_analysis"
	JobTypeReportGeneration  = "report_generation"
	JobTypeNotificationEmail = "notification_email"
)

// Common queue names.
const (
	QueueDefault  = "default"
	QueueCritical = "critical"
	QueueLow      = "low"
)

// EnqueueOption configures job enqueueing.
type EnqueueOption func(*enqueueOptions)

type enqueueOptions struct {
	Priority    int
	MaxAttempts int
	ScheduledAt time.Time
	Delay       time.Duration
}

// WithPriority sets the job priority (higher = more important).
func WithPriority(priority int) EnqueueOption {
	return func(o *enqueueOptions) {
		o.Priority = priority
	}
}

// WithMaxAttempts sets the maximum retry attempts.
func WithMaxAttempts(attempts int) EnqueueOption {
	return func(o *enqueueOptions) {
		o.MaxAttempts = attempts
	}
}

// WithScheduledAt schedules the job for a specific time.
func WithScheduledAt(t time.Time) EnqueueOption {
	return func(o *enqueueOptions) {
		o.ScheduledAt = t
	}
}

// WithDelay schedules the job to run after a delay.
func WithDelay(d time.Duration) EnqueueOption {
	return func(o *enqueueOptions) {
		o.Delay = d
	}
}

// QueueConfig holds configuration for the job queue.
type QueueConfig struct {
	// Provider is the queue provider ("postgres" or "redis").
	Provider string

	// WorkerCount is the number of concurrent workers.
	WorkerCount int

	// PollInterval is how often to poll for jobs.
	PollInterval time.Duration

	// JobTimeout is the maximum time a job can run.
	JobTimeout time.Duration

	// EnableRateLimiting enables per-organization rate limits.
	EnableRateLimiting bool
}

// DefaultQueueConfig returns the default queue configuration.
func DefaultQueueConfig() QueueConfig {
	return QueueConfig{
		Provider:           "postgres",
		WorkerCount:        3,
		PollInterval:       time.Second,
		JobTimeout:         60 * time.Second,
		EnableRateLimiting: true,
	}
}

// JobHandler handles processing of a specific job type.
type JobHandler interface {
	// Handle processes a job.
	// Return nil on success, or an error to trigger retry logic.
	Handle(ctx context.Context, job *Job) error
}

// JobHandlerFunc is an adapter to allow ordinary functions as JobHandlers.
type JobHandlerFunc func(ctx context.Context, job *Job) error

// Handle implements JobHandler.
func (f JobHandlerFunc) Handle(ctx context.Context, job *Job) error {
	return f(ctx, job)
}
