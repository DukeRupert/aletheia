package queue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MockQueue is an in-memory queue implementation for testing
type MockQueue struct {
	mu                sync.RWMutex
	jobs              map[uuid.UUID]*Job
	organizationUsage map[uuid.UUID]*usageTracker
}

type usageTracker struct {
	jobsInWindow   int
	windowStart    time.Time
	processingJobs int
}

// NewMockQueue creates a new in-memory mock queue
func NewMockQueue() *MockQueue {
	return &MockQueue{
		jobs:              make(map[uuid.UUID]*Job),
		organizationUsage: make(map[uuid.UUID]*usageTracker),
	}
}

func (m *MockQueue) Enqueue(ctx context.Context, queueName, jobType string, organizationID uuid.UUID, payload map[string]interface{}, opts *EnqueueOptions) (*Job, error) {
	if opts == nil {
		opts = DefaultEnqueueOptions()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	job := &Job{
		ID:             uuid.New(),
		QueueName:      queueName,
		JobType:        jobType,
		OrganizationID: organizationID,
		Payload:        payload,
		Status:         JobStatusPending,
		Priority:       opts.Priority,
		MaxAttempts:    opts.MaxAttempts,
		AttemptCount:   0,
		CreatedAt:      time.Now(),
		ScheduledAt:    time.Now(),
	}

	if opts.ScheduledAt != nil {
		job.ScheduledAt = *opts.ScheduledAt
	} else if opts.Delay > 0 {
		job.ScheduledAt = time.Now().Add(opts.Delay)
	}

	m.jobs[job.ID] = job
	return job, nil
}

func (m *MockQueue) Dequeue(ctx context.Context, workerID string, opts *DequeueOptions) (*Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Find next pending job
	var bestJob *Job
	for _, job := range m.jobs {
		if job.Status != JobStatusPending {
			continue
		}
		if job.ScheduledAt.After(time.Now()) {
			continue
		}
		if bestJob == nil || job.Priority > bestJob.Priority {
			bestJob = job
		}
	}

	if bestJob == nil {
		return nil, nil
	}

	// Lock the job
	now := time.Now()
	bestJob.Status = JobStatusProcessing
	bestJob.StartedAt = &now
	bestJob.AttemptCount++
	bestJob.WorkerID = workerID

	return bestJob, nil
}

func (m *MockQueue) Complete(ctx context.Context, jobID uuid.UUID, result map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, exists := m.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	now := time.Now()
	job.Status = JobStatusCompleted
	job.CompletedAt = &now
	job.Result = result

	return nil
}

func (m *MockQueue) Fail(ctx context.Context, jobID uuid.UUID, errMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, exists := m.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	job.ErrorMessage = errMsg

	if job.AttemptCount >= job.MaxAttempts {
		job.Status = JobStatusFailed
	} else {
		// Retry with exponential backoff
		job.Status = JobStatusPending
		backoff := time.Duration(1<<uint(job.AttemptCount)) * time.Minute
		job.ScheduledAt = time.Now().Add(backoff)
	}

	return nil
}

func (m *MockQueue) Delete(ctx context.Context, jobID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.jobs, jobID)
	return nil
}

func (m *MockQueue) GetJob(ctx context.Context, jobID uuid.UUID) (*Job, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	job, exists := m.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	return job, nil
}

func (m *MockQueue) ListJobs(ctx context.Context, filter JobFilter) ([]*Job, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var jobs []*Job
	for _, job := range m.jobs {
		if filter.QueueName != nil && *filter.QueueName != job.QueueName {
			continue
		}
		if filter.JobType != nil && *filter.JobType != job.JobType {
			continue
		}
		if filter.OrganizationID != nil && *filter.OrganizationID != job.OrganizationID {
			continue
		}
		if filter.Status != nil && *filter.Status != job.Status {
			continue
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

func (m *MockQueue) GetQueueStats(ctx context.Context, queueName string) (*QueueStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := &QueueStats{
		QueueName: queueName,
	}

	for _, job := range m.jobs {
		if job.QueueName != queueName {
			continue
		}

		switch job.Status {
		case JobStatusPending:
			stats.PendingJobs++
		case JobStatusProcessing:
			stats.ProcessingJobs++
		case JobStatusCompleted:
			stats.CompletedJobs++
		case JobStatusFailed:
			stats.FailedJobs++
		}
	}

	return stats, nil
}

func (m *MockQueue) CanProcessJob(ctx context.Context, organizationID uuid.UUID, queueName string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Mock always allows processing
	return true, nil
}

func (m *MockQueue) RecordJobProcessed(ctx context.Context, organizationID uuid.UUID, queueName string) error {
	// Mock does nothing
	return nil
}

func (m *MockQueue) Close() error {
	// Nothing to close for mock
	return nil
}

// Verify MockQueue implements Queue interface
var _ Queue = (*MockQueue)(nil)
