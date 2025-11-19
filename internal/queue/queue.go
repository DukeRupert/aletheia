package queue

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// JobStatus represents the state of a job in the queue
type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
)

// Job represents a unit of work in the queue
type Job struct {
	ID             uuid.UUID
	QueueName      string
	JobType        string
	OrganizationID uuid.UUID
	Payload        map[string]interface{} // Generic payload as map
	Status         JobStatus
	Priority       int
	MaxAttempts    int
	AttemptCount   int
	ScheduledAt    time.Time
	CreatedAt      time.Time
	StartedAt      *time.Time
	CompletedAt    *time.Time
	Result         map[string]interface{} // Generic result as map
	ErrorMessage   string
	WorkerID       string
}

// EnqueueOptions allows customization when enqueuing a job
type EnqueueOptions struct {
	Priority    int           // Higher = more important, default 0
	MaxAttempts int           // Maximum retry attempts, default 3
	ScheduledAt *time.Time    // When to run the job, default NOW
	Delay       time.Duration // Alternative to ScheduledAt: delay from now
}

// DequeueOptions configures how jobs are dequeued
type DequeueOptions struct {
	QueueNames []string      // Which queues to check (empty = all queues)
	Timeout    time.Duration // How long to wait for a job
}

// Queue defines the interface for job queue operations
type Queue interface {
	// Enqueue adds a new job to the specified queue
	Enqueue(ctx context.Context, queueName, jobType string, organizationID uuid.UUID, payload map[string]interface{}, opts *EnqueueOptions) (*Job, error)

	// Dequeue retrieves and locks the next available job for processing
	// Returns nil if no jobs available
	Dequeue(ctx context.Context, workerID string, opts *DequeueOptions) (*Job, error)

	// Complete marks a job as successfully completed with results
	Complete(ctx context.Context, jobID uuid.UUID, result map[string]interface{}) error

	// Fail marks a job as failed
	// Will automatically retry if attempts < maxAttempts with exponential backoff
	Fail(ctx context.Context, jobID uuid.UUID, errMsg string) error

	// Delete removes a job from the queue
	Delete(ctx context.Context, jobID uuid.UUID) error

	// GetJob retrieves a job by ID
	GetJob(ctx context.Context, jobID uuid.UUID) (*Job, error)

	// ListJobs retrieves jobs with filtering
	ListJobs(ctx context.Context, filter JobFilter) ([]*Job, error)

	// GetQueueStats returns statistics for queues
	GetQueueStats(ctx context.Context, queueName string) (*QueueStats, error)

	// CanProcessJob checks if an organization can process more jobs (rate limiting)
	CanProcessJob(ctx context.Context, organizationID uuid.UUID, queueName string) (bool, error)

	// RecordJobProcessed increments the job counter for rate limiting
	RecordJobProcessed(ctx context.Context, organizationID uuid.UUID, queueName string) error

	// Close gracefully shuts down the queue
	Close() error
}

// JobFilter defines filtering options for listing jobs
type JobFilter struct {
	QueueName      *string
	JobType        *string
	OrganizationID *uuid.UUID
	Status         *JobStatus
	Limit          int
	Offset         int
}

// QueueStats provides statistics about a queue
type QueueStats struct {
	QueueName        string
	PendingJobs      int
	ProcessingJobs   int
	CompletedJobs    int
	FailedJobs       int
	AvgProcessingTime time.Duration
}

// Config holds configuration for queue implementations
type Config struct {
	// Provider specifies the queue implementation: "postgres" or "redis"
	Provider string

	// PostgreSQL config
	PostgresConnectionString string

	// Redis config (for future use)
	RedisURL string

	// Worker configuration
	WorkerCount      int           // Number of concurrent workers per queue
	PollInterval     time.Duration // How often to poll for new jobs
	JobTimeout       time.Duration // Default timeout for job processing
	ShutdownTimeout  time.Duration // How long to wait for graceful shutdown
	CleanupInterval  time.Duration // How often to cleanup old completed jobs
	CleanupRetention time.Duration // How long to keep completed jobs

	// Rate limiting defaults
	DefaultMaxJobsPerHour      int
	DefaultMaxConcurrentJobs   int
	EnableRateLimiting         bool
}

// DefaultEnqueueOptions returns sensible defaults for enqueuing jobs
func DefaultEnqueueOptions() *EnqueueOptions {
	return &EnqueueOptions{
		Priority:    0,
		MaxAttempts: 3,
	}
}

// DefaultDequeueOptions returns sensible defaults for dequeuing jobs
func DefaultDequeueOptions() *DequeueOptions {
	return &DequeueOptions{
		QueueNames: []string{}, // Empty means all queues
		Timeout:    5 * time.Second,
	}
}

// DefaultConfig returns default queue configuration
func DefaultConfig() Config {
	return Config{
		Provider:                   "postgres",
		WorkerCount:                3,
		PollInterval:               1 * time.Second,
		JobTimeout:                 60 * time.Second,
		ShutdownTimeout:            30 * time.Second,
		CleanupInterval:            1 * time.Hour,
		CleanupRetention:           24 * time.Hour,
		DefaultMaxJobsPerHour:      100,
		DefaultMaxConcurrentJobs:   5,
		EnableRateLimiting:         true,
	}
}
