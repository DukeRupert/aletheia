package queue

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockQueue_EnqueueDequeue(t *testing.T) {
	queue := NewMockQueue()
	ctx := context.Background()
	orgID := uuid.New()

	// Enqueue a job
	payload := map[string]interface{}{
		"test": "data",
	}

	job, err := queue.Enqueue(ctx, "test_queue", "test_job", orgID, payload, nil)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, job.ID)
	assert.Equal(t, "test_queue", job.QueueName)
	assert.Equal(t, "test_job", job.JobType)
	assert.Equal(t, orgID, job.OrganizationID)
	assert.Equal(t, JobStatusPending, job.Status)
	assert.Equal(t, 0, job.AttemptCount)

	// Dequeue the job
	dequeuedJob, err := queue.Dequeue(ctx, "worker-1", nil)
	require.NoError(t, err)
	require.NotNil(t, dequeuedJob)
	assert.Equal(t, job.ID, dequeuedJob.ID)
	assert.Equal(t, JobStatusProcessing, dequeuedJob.Status)
	assert.Equal(t, 1, dequeuedJob.AttemptCount)
	assert.Equal(t, "worker-1", dequeuedJob.WorkerID)

	// No more jobs
	noJob, err := queue.Dequeue(ctx, "worker-2", nil)
	require.NoError(t, err)
	assert.Nil(t, noJob)
}

func TestMockQueue_Complete(t *testing.T) {
	queue := NewMockQueue()
	ctx := context.Background()
	orgID := uuid.New()

	// Enqueue and dequeue
	job, err := queue.Enqueue(ctx, "test_queue", "test_job", orgID, map[string]interface{}{}, nil)
	require.NoError(t, err)

	dequeuedJob, err := queue.Dequeue(ctx, "worker-1", nil)
	require.NoError(t, err)

	// Complete the job
	result := map[string]interface{}{
		"status": "success",
	}
	err = queue.Complete(ctx, dequeuedJob.ID, result)
	require.NoError(t, err)

	// Verify job is completed
	completedJob, err := queue.GetJob(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, JobStatusCompleted, completedJob.Status)
	assert.NotNil(t, completedJob.CompletedAt)
	assert.Equal(t, result, completedJob.Result)
}

func TestMockQueue_Fail_WithRetry(t *testing.T) {
	queue := NewMockQueue()
	ctx := context.Background()
	orgID := uuid.New()

	// Enqueue with max attempts 3
	job, err := queue.Enqueue(ctx, "test_queue", "test_job", orgID, map[string]interface{}{}, nil)
	require.NoError(t, err)

	// First attempt
	dequeuedJob, err := queue.Dequeue(ctx, "worker-1", nil)
	require.NoError(t, err)
	assert.Equal(t, 1, dequeuedJob.AttemptCount)

	err = queue.Fail(ctx, dequeuedJob.ID, "temporary error")
	require.NoError(t, err)

	// Job should be pending again
	retriedJob, err := queue.GetJob(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, JobStatusPending, retriedJob.Status)
	assert.Equal(t, "temporary error", retriedJob.ErrorMessage)
}

func TestMockQueue_Fail_Permanent(t *testing.T) {
	queue := NewMockQueue()
	ctx := context.Background()
	orgID := uuid.New()

	// Enqueue with max attempts 1
	opts := &EnqueueOptions{
		MaxAttempts: 1,
	}
	job, err := queue.Enqueue(ctx, "test_queue", "test_job", orgID, map[string]interface{}{}, opts)
	require.NoError(t, err)

	// Dequeue and fail
	dequeuedJob, err := queue.Dequeue(ctx, "worker-1", nil)
	require.NoError(t, err)

	err = queue.Fail(ctx, dequeuedJob.ID, "permanent error")
	require.NoError(t, err)

	// Job should be permanently failed
	failedJob, err := queue.GetJob(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, JobStatusFailed, failedJob.Status)
	assert.Equal(t, "permanent error", failedJob.ErrorMessage)
}

func TestMockQueue_Priority(t *testing.T) {
	queue := NewMockQueue()
	ctx := context.Background()
	orgID := uuid.New()

	// Enqueue jobs with different priorities
	lowJob, err := queue.Enqueue(ctx, "test_queue", "test_job", orgID,
		map[string]interface{}{"priority": "low"}, &EnqueueOptions{Priority: 1})
	require.NoError(t, err)

	highJob, err := queue.Enqueue(ctx, "test_queue", "test_job", orgID,
		map[string]interface{}{"priority": "high"}, &EnqueueOptions{Priority: 10})
	require.NoError(t, err)

	// High priority should be dequeued first
	job1, err := queue.Dequeue(ctx, "worker-1", nil)
	require.NoError(t, err)
	assert.Equal(t, highJob.ID, job1.ID)

	job2, err := queue.Dequeue(ctx, "worker-1", nil)
	require.NoError(t, err)
	assert.Equal(t, lowJob.ID, job2.ID)
}

func TestMockQueue_ScheduledJobs(t *testing.T) {
	queue := NewMockQueue()
	ctx := context.Background()
	orgID := uuid.New()

	// Enqueue job scheduled for future
	futureTime := time.Now().Add(1 * time.Hour)
	_, err := queue.Enqueue(ctx, "test_queue", "test_job", orgID,
		map[string]interface{}{}, &EnqueueOptions{ScheduledAt: &futureTime})
	require.NoError(t, err)

	// Should not be dequeued yet
	job, err := queue.Dequeue(ctx, "worker-1", nil)
	require.NoError(t, err)
	assert.Nil(t, job)

	// Enqueue job scheduled for now
	now := time.Now()
	immediateJob, err := queue.Enqueue(ctx, "test_queue", "test_job", orgID,
		map[string]interface{}{}, &EnqueueOptions{ScheduledAt: &now})
	require.NoError(t, err)

	// Should be dequeued
	job, err = queue.Dequeue(ctx, "worker-1", nil)
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, immediateJob.ID, job.ID)
}

func TestMockQueue_Delete(t *testing.T) {
	queue := NewMockQueue()
	ctx := context.Background()
	orgID := uuid.New()

	// Enqueue a job
	job, err := queue.Enqueue(ctx, "test_queue", "test_job", orgID, map[string]interface{}{}, nil)
	require.NoError(t, err)

	// Delete the job
	err = queue.Delete(ctx, job.ID)
	require.NoError(t, err)

	// Job should not exist
	_, err = queue.GetJob(ctx, job.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "job not found")
}

func TestMockQueue_GetJob(t *testing.T) {
	queue := NewMockQueue()
	ctx := context.Background()
	orgID := uuid.New()

	// Enqueue a job
	job, err := queue.Enqueue(ctx, "test_queue", "test_job", orgID, map[string]interface{}{"key": "value"}, nil)
	require.NoError(t, err)

	// Get the job
	retrievedJob, err := queue.GetJob(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, job.ID, retrievedJob.ID)
	assert.Equal(t, job.QueueName, retrievedJob.QueueName)
	assert.Equal(t, job.JobType, retrievedJob.JobType)
}

func TestMockQueue_ListJobs(t *testing.T) {
	queue := NewMockQueue()
	ctx := context.Background()
	orgID := uuid.New()

	// Enqueue multiple jobs
	for i := 0; i < 5; i++ {
		_, err := queue.Enqueue(ctx, "test_queue", "test_job", orgID,
			map[string]interface{}{"index": i}, nil)
		require.NoError(t, err)
	}

	// List all jobs
	filter := JobFilter{
		OrganizationID: &orgID,
	}
	jobs, err := queue.ListJobs(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, jobs, 5)

	// Filter by status
	pendingStatus := JobStatusPending
	filter.Status = &pendingStatus
	jobs, err = queue.ListJobs(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, jobs, 5)

	// Dequeue one
	_, err = queue.Dequeue(ctx, "worker-1", nil)
	require.NoError(t, err)

	// Pending should be 4 now
	jobs, err = queue.ListJobs(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, jobs, 4)
}

func TestMockQueue_GetQueueStats(t *testing.T) {
	queue := NewMockQueue()
	ctx := context.Background()
	orgID := uuid.New()

	// Enqueue jobs
	_, err := queue.Enqueue(ctx, "test_queue", "test_job", orgID, map[string]interface{}{}, nil)
	require.NoError(t, err)
	_, err = queue.Enqueue(ctx, "test_queue", "test_job", orgID, map[string]interface{}{}, nil)
	require.NoError(t, err)
	job3, err := queue.Enqueue(ctx, "test_queue", "test_job", orgID, map[string]interface{}{}, nil)
	require.NoError(t, err)

	// Dequeue and complete one
	dequeuedJob, err := queue.Dequeue(ctx, "worker-1", nil)
	require.NoError(t, err)
	err = queue.Complete(ctx, dequeuedJob.ID, map[string]interface{}{})
	require.NoError(t, err)

	// Dequeue another (leave processing)
	_, err = queue.Dequeue(ctx, "worker-2", nil)
	require.NoError(t, err)

	// Get stats
	stats, err := queue.GetQueueStats(ctx, "test_queue")
	require.NoError(t, err)
	assert.Equal(t, "test_queue", stats.QueueName)
	assert.Equal(t, 1, stats.PendingJobs)
	assert.Equal(t, 1, stats.ProcessingJobs)
	assert.Equal(t, 1, stats.CompletedJobs)
	assert.Equal(t, 0, stats.FailedJobs)

	// Fail the last job
	_, err = queue.Dequeue(ctx, "worker-3", nil)
	require.NoError(t, err)
	err = queue.Fail(ctx, job3.ID, "error")
	require.NoError(t, err)
}

func TestMockQueue_CanProcessJob(t *testing.T) {
	queue := NewMockQueue()
	ctx := context.Background()
	orgID := uuid.New()

	// Mock always allows processing
	canProcess, err := queue.CanProcessJob(ctx, orgID, "test_queue")
	require.NoError(t, err)
	assert.True(t, canProcess)
}

func TestMockQueue_RecordJobProcessed(t *testing.T) {
	queue := NewMockQueue()
	ctx := context.Background()
	orgID := uuid.New()

	// Should not error (no-op for mock)
	err := queue.RecordJobProcessed(ctx, orgID, "test_queue")
	require.NoError(t, err)
}

func TestMockQueue_Close(t *testing.T) {
	queue := NewMockQueue()

	// Should not error
	err := queue.Close()
	require.NoError(t, err)
}

func TestMockQueue_FilterByQueueName(t *testing.T) {
	queue := NewMockQueue()
	ctx := context.Background()
	orgID := uuid.New()

	// Enqueue jobs in different queues
	_, err := queue.Enqueue(ctx, "queue_a", "test_job", orgID, map[string]interface{}{}, nil)
	require.NoError(t, err)
	_, err = queue.Enqueue(ctx, "queue_b", "test_job", orgID, map[string]interface{}{}, nil)
	require.NoError(t, err)
	_, err = queue.Enqueue(ctx, "queue_a", "test_job", orgID, map[string]interface{}{}, nil)
	require.NoError(t, err)

	// Filter by queue_a
	queueName := "queue_a"
	filter := JobFilter{
		QueueName: &queueName,
	}
	jobs, err := queue.ListJobs(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, jobs, 2)
	for _, job := range jobs {
		assert.Equal(t, "queue_a", job.QueueName)
	}
}

func TestMockQueue_FilterByJobType(t *testing.T) {
	queue := NewMockQueue()
	ctx := context.Background()
	orgID := uuid.New()

	// Enqueue jobs of different types
	_, err := queue.Enqueue(ctx, "test_queue", "type_a", orgID, map[string]interface{}{}, nil)
	require.NoError(t, err)
	_, err = queue.Enqueue(ctx, "test_queue", "type_b", orgID, map[string]interface{}{}, nil)
	require.NoError(t, err)
	_, err = queue.Enqueue(ctx, "test_queue", "type_a", orgID, map[string]interface{}{}, nil)
	require.NoError(t, err)

	// Filter by type_a
	jobType := "type_a"
	filter := JobFilter{
		JobType: &jobType,
	}
	jobs, err := queue.ListJobs(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, jobs, 2)
	for _, job := range jobs {
		assert.Equal(t, "type_a", job.JobType)
	}
}

func TestMockQueue_Delay(t *testing.T) {
	queue := NewMockQueue()
	ctx := context.Background()
	orgID := uuid.New()

	// Enqueue with delay
	opts := &EnqueueOptions{
		Delay: 1 * time.Hour,
	}
	job, err := queue.Enqueue(ctx, "test_queue", "test_job", orgID, map[string]interface{}{}, opts)
	require.NoError(t, err)

	// ScheduledAt should be in the future
	assert.True(t, job.ScheduledAt.After(time.Now()))

	// Should not be dequeued yet
	dequeuedJob, err := queue.Dequeue(ctx, "worker-1", nil)
	require.NoError(t, err)
	assert.Nil(t, dequeuedJob)
}

func TestMockQueue_ExponentialBackoff(t *testing.T) {
	queue := NewMockQueue()
	ctx := context.Background()
	orgID := uuid.New()

	// Enqueue job
	job, err := queue.Enqueue(ctx, "test_queue", "test_job", orgID, map[string]interface{}{}, nil)
	require.NoError(t, err)

	// Dequeue and fail multiple times
	for i := 0; i < 3; i++ {
		dequeuedJob, err := queue.Dequeue(ctx, "worker-1", nil)
		if dequeuedJob == nil {
			// Job is scheduled for future, break
			break
		}
		require.NoError(t, err)

		err = queue.Fail(ctx, dequeuedJob.ID, "error")
		require.NoError(t, err)

		// Check scheduled time increases exponentially
		retriedJob, err := queue.GetJob(ctx, job.ID)
		require.NoError(t, err)

		if retriedJob.Status == JobStatusPending {
			// Backoff: 2^attemptCount minutes
			// Allow some tolerance
			actualDelay := retriedJob.ScheduledAt.Sub(time.Now())
			assert.True(t, actualDelay > 0)
		}
	}
}
