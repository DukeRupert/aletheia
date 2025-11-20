package queue

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

// JobHandler is a function that processes a job
type JobHandler func(ctx context.Context, job *Job) (result map[string]interface{}, err error)

// WorkerPool manages a pool of workers that process jobs from queues
type WorkerPool struct {
	queue    Queue
	logger   *slog.Logger
	config   Config
	handlers map[string]JobHandler // job_type -> handler
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	mu       sync.RWMutex
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(queue Queue, logger *slog.Logger, config Config) *WorkerPool {
	return &WorkerPool{
		queue:    queue,
		logger:   logger,
		config:   config,
		handlers: make(map[string]JobHandler),
	}
}

// RegisterHandler registers a handler for a specific job type
func (wp *WorkerPool) RegisterHandler(jobType string, handler JobHandler) {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	wp.handlers[jobType] = handler
	wp.logger.Info("registered job handler",
		slog.String("job_type", jobType),
	)
}

// Start starts the worker pool with the specified number of workers
func (wp *WorkerPool) Start(ctx context.Context, queueNames []string) error {
	wp.mu.Lock()
	if wp.cancel != nil {
		wp.mu.Unlock()
		return fmt.Errorf("worker pool already started")
	}

	workerCtx, cancel := context.WithCancel(ctx)
	wp.cancel = cancel
	wp.mu.Unlock()

	// Start workers
	for i := 0; i < wp.config.WorkerCount; i++ {
		wp.wg.Add(1)
		workerID := fmt.Sprintf("worker-%d", i+1)

		go wp.worker(workerCtx, workerID, queueNames)
	}

	wp.logger.Info("worker pool started",
		slog.Int("worker_count", wp.config.WorkerCount),
		slog.Any("queues", queueNames),
	)

	return nil
}

// Stop gracefully stops the worker pool
func (wp *WorkerPool) Stop() error {
	wp.mu.Lock()
	if wp.cancel == nil {
		wp.mu.Unlock()
		return fmt.Errorf("worker pool not started")
	}
	cancel := wp.cancel
	wp.cancel = nil
	wp.mu.Unlock()

	wp.logger.Info("stopping worker pool")

	// Signal workers to stop
	cancel()

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		wp.logger.Info("worker pool stopped gracefully")
		return nil
	case <-time.After(wp.config.ShutdownTimeout):
		wp.logger.Warn("worker pool shutdown timeout",
			slog.Duration("timeout", wp.config.ShutdownTimeout),
		)
		return fmt.Errorf("shutdown timeout after %v", wp.config.ShutdownTimeout)
	}
}

// worker is the main worker loop
func (wp *WorkerPool) worker(ctx context.Context, workerID string, queueNames []string) {
	defer wp.wg.Done()

	wp.logger.Debug("worker started", slog.String("worker_id", workerID))

	ticker := time.NewTicker(wp.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			wp.logger.Debug("worker stopping", slog.String("worker_id", workerID))
			return

		case <-ticker.C:
			if err := wp.processNextJob(ctx, workerID, queueNames); err != nil {
				wp.logger.Error("failed to process job",
					slog.String("worker_id", workerID),
					slog.String("error", err.Error()),
				)
			}
		}
	}
}

// processNextJob attempts to dequeue and process a single job
func (wp *WorkerPool) processNextJob(ctx context.Context, workerID string, queueNames []string) error {
	// Dequeue next job
	opts := &DequeueOptions{
		QueueNames: queueNames,
		Timeout:    wp.config.PollInterval,
	}

	job, err := wp.queue.Dequeue(ctx, workerID, opts)
	if err != nil {
		return fmt.Errorf("failed to dequeue job: %w", err)
	}

	if job == nil {
		// No jobs available
		return nil
	}

	// Check rate limits before processing
	if wp.config.EnableRateLimiting {
		canProcess, err := wp.queue.CanProcessJob(ctx, job.OrganizationID, job.QueueName)
		if err != nil {
			return fmt.Errorf("failed to check rate limits: %w", err)
		}

		if !canProcess {
			// Re-queue the job for later
			wp.logger.Debug("rate limit exceeded, requeueing job",
				slog.String("job_id", job.ID.String()),
				slog.String("org_id", job.OrganizationID.String()),
			)

			// Mark as pending again with a delay
			if err := wp.queue.Fail(ctx, job.ID, "rate limit exceeded"); err != nil {
				return fmt.Errorf("failed to requeue job: %w", err)
			}

			return nil
		}
	}

	// Process the job
	return wp.executeJob(ctx, job)
}

// executeJob runs the job handler and updates the job status
func (wp *WorkerPool) executeJob(ctx context.Context, job *Job) error {
	wp.logger.Info("processing job",
		slog.String("job_id", job.ID.String()),
		slog.String("queue", job.QueueName),
		slog.String("type", job.JobType),
		slog.Int("attempt", job.AttemptCount),
	)

	// Find handler
	wp.mu.RLock()
	handler, exists := wp.handlers[job.JobType]
	wp.mu.RUnlock()

	if !exists {
		errMsg := fmt.Sprintf("no handler registered for job type: %s", job.JobType)
		wp.logger.Error("handler not found",
			slog.String("job_id", job.ID.String()),
			slog.String("job_type", job.JobType),
		)
		return wp.queue.Fail(ctx, job.ID, errMsg)
	}

	// Create job context with timeout
	jobCtx, cancel := context.WithTimeout(ctx, wp.config.JobTimeout)
	defer cancel()

	// Execute handler
	startTime := time.Now()
	result, err := handler(jobCtx, job)
	duration := time.Since(startTime)

	if err != nil {
		wp.logger.Error("job failed",
			slog.String("job_id", job.ID.String()),
			slog.String("error", err.Error()),
			slog.Duration("duration", duration),
		)

		return wp.queue.Fail(ctx, job.ID, err.Error())
	}

	// Record successful processing for rate limiting
	if wp.config.EnableRateLimiting {
		if err := wp.queue.RecordJobProcessed(ctx, job.OrganizationID, job.QueueName); err != nil {
			wp.logger.Warn("failed to record job processed",
				slog.String("error", err.Error()),
			)
		}
	}

	wp.logger.Info("job completed",
		slog.String("job_id", job.ID.String()),
		slog.Duration("duration", duration),
	)

	return wp.queue.Complete(ctx, job.ID, result)
}

// GetHandler retrieves a registered handler (for testing)
func (wp *WorkerPool) GetHandler(jobType string) (JobHandler, bool) {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	handler, exists := wp.handlers[jobType]
	return handler, exists
}

// StartBackgroundCleanup starts a goroutine that periodically cleans up old jobs
func (wp *WorkerPool) StartBackgroundCleanup(ctx context.Context) {
	wp.wg.Add(1)

	go func() {
		defer wp.wg.Done()

		ticker := time.NewTicker(wp.config.CleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return

			case <-ticker.C:
				if err := wp.cleanupOldJobs(ctx); err != nil {
					wp.logger.Error("cleanup failed",
						slog.String("error", err.Error()),
					)
				}
			}
		}
	}()

	wp.logger.Info("background cleanup started",
		slog.Duration("interval", wp.config.CleanupInterval),
		slog.Duration("retention", wp.config.CleanupRetention),
	)
}

// cleanupOldJobs deletes old completed and failed jobs
func (wp *WorkerPool) cleanupOldJobs(ctx context.Context) error {
	// This would need to be implemented based on your Queue interface
	// For now, we'll log that it ran
	wp.logger.Debug("running job cleanup")

	// If using PostgresQueue, you could add a cleanup method to the interface
	// or run raw SQL here. For now, we'll skip the actual deletion.

	return nil
}

// EnqueueJob is a convenience method to enqueue a job
// This is a helper that can be used by handlers
func (wp *WorkerPool) EnqueueJob(ctx context.Context, queueName, jobType string, organizationID uuid.UUID, payload map[string]interface{}, opts *EnqueueOptions) (*Job, error) {
	return wp.queue.Enqueue(ctx, queueName, jobType, organizationID, payload, opts)
}
