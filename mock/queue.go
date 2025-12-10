package mock

import (
	"context"
	"sync"
	"time"

	"github.com/dukerupert/aletheia"
	"github.com/google/uuid"
)

// Compile-time interface check
var _ aletheia.Queue = (*Queue)(nil)

// Queue is a mock implementation of aletheia.Queue.
type Queue struct {
	EnqueueFn       func(ctx context.Context, job *aletheia.Job, opts ...aletheia.EnqueueOption) error
	DequeueFn       func(ctx context.Context, queueName string) (*aletheia.Job, error)
	CompleteFn      func(ctx context.Context, jobID uuid.UUID, result []byte) error
	FailFn          func(ctx context.Context, jobID uuid.UUID, errMsg string) error
	GetJobFn        func(ctx context.Context, jobID uuid.UUID) (*aletheia.Job, error)
	CancelJobFn     func(ctx context.Context, jobID uuid.UUID) error
	GetPendingJobsFn func(ctx context.Context, orgID uuid.UUID, queueName string) ([]*aletheia.Job, error)

	// In-memory job storage for testing
	mu   sync.RWMutex
	jobs map[uuid.UUID]*aletheia.Job
}

// NewQueue creates a new mock queue with initialized storage.
func NewQueue() *Queue {
	return &Queue{
		jobs: make(map[uuid.UUID]*aletheia.Job),
	}
}

func (q *Queue) Enqueue(ctx context.Context, job *aletheia.Job, opts ...aletheia.EnqueueOption) error {
	if q.EnqueueFn != nil {
		return q.EnqueueFn(ctx, job, opts...)
	}

	q.mu.Lock()
	defer q.mu.Unlock()

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

	q.jobs[job.ID] = job
	return nil
}

func (q *Queue) Dequeue(ctx context.Context, queueName string) (*aletheia.Job, error) {
	if q.DequeueFn != nil {
		return q.DequeueFn(ctx, queueName)
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	for _, job := range q.jobs {
		if job.QueueName == queueName && job.Status == aletheia.JobStatusPending {
			job.Status = aletheia.JobStatusRunning
			now := time.Now()
			job.StartedAt = &now
			return job, nil
		}
	}
	return nil, nil
}

func (q *Queue) Complete(ctx context.Context, jobID uuid.UUID, result []byte) error {
	if q.CompleteFn != nil {
		return q.CompleteFn(ctx, jobID, result)
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	job, ok := q.jobs[jobID]
	if !ok {
		return aletheia.NotFound("Job not found")
	}
	job.Status = aletheia.JobStatusCompleted
	job.Result = result
	now := time.Now()
	job.CompletedAt = &now
	return nil
}

func (q *Queue) Fail(ctx context.Context, jobID uuid.UUID, errMsg string) error {
	if q.FailFn != nil {
		return q.FailFn(ctx, jobID, errMsg)
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	job, ok := q.jobs[jobID]
	if !ok {
		return aletheia.NotFound("Job not found")
	}
	job.Status = aletheia.JobStatusFailed
	job.ErrorMessage = errMsg
	now := time.Now()
	job.CompletedAt = &now
	return nil
}

func (q *Queue) GetJob(ctx context.Context, jobID uuid.UUID) (*aletheia.Job, error) {
	if q.GetJobFn != nil {
		return q.GetJobFn(ctx, jobID)
	}

	q.mu.RLock()
	defer q.mu.RUnlock()

	job, ok := q.jobs[jobID]
	if !ok {
		return nil, aletheia.NotFound("Job not found")
	}
	return job, nil
}

func (q *Queue) CancelJob(ctx context.Context, jobID uuid.UUID) error {
	if q.CancelJobFn != nil {
		return q.CancelJobFn(ctx, jobID)
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	job, ok := q.jobs[jobID]
	if !ok {
		return aletheia.NotFound("Job not found")
	}
	if job.Status != aletheia.JobStatusPending {
		return aletheia.Invalid("Can only cancel pending jobs")
	}
	job.Status = aletheia.JobStatusCancelled
	return nil
}

func (q *Queue) GetPendingJobs(ctx context.Context, orgID uuid.UUID, queueName string) ([]*aletheia.Job, error) {
	if q.GetPendingJobsFn != nil {
		return q.GetPendingJobsFn(ctx, orgID, queueName)
	}

	q.mu.RLock()
	defer q.mu.RUnlock()

	var result []*aletheia.Job
	for _, job := range q.jobs {
		if job.OrganizationID == orgID && job.QueueName == queueName && job.Status == aletheia.JobStatusPending {
			result = append(result, job)
		}
	}
	return result, nil
}

// Reset clears all jobs from the mock queue.
func (q *Queue) Reset() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.jobs = make(map[uuid.UUID]*aletheia.Job)
}

// AllJobs returns all jobs in the mock queue.
func (q *Queue) AllJobs() []*aletheia.Job {
	q.mu.RLock()
	defer q.mu.RUnlock()

	result := make([]*aletheia.Job, 0, len(q.jobs))
	for _, job := range q.jobs {
		result = append(result, job)
	}
	return result
}

// JobsByType returns all jobs of a specific type.
func (q *Queue) JobsByType(jobType string) []*aletheia.Job {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var result []*aletheia.Job
	for _, job := range q.jobs {
		if job.JobType == jobType {
			result = append(result, job)
		}
	}
	return result
}
