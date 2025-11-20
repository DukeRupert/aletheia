package queue

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"os"
)

func setupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	// Use test database connection string from environment
	connString := os.Getenv("GOOSE_DBSTRING")
	if connString == "" {
		t.Skip("GOOSE_DBSTRING not set, skipping integration tests")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connString)
	require.NoError(t, err)

	// Verify connection
	err = pool.Ping(ctx)
	require.NoError(t, err)

	// Create test organization
	testOrgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	_, err = pool.Exec(ctx, `
		INSERT INTO organizations (id, name, created_at)
		VALUES ($1, 'Test Organization', NOW())
		ON CONFLICT (id) DO NOTHING
	`, testOrgID)
	require.NoError(t, err)

	// Cleanup function
	cleanup := func() {
		// Clean up test data (cascading deletes will handle jobs and rate_limits)
		_, _ = pool.Exec(ctx, "DELETE FROM jobs WHERE organization_id = $1", testOrgID)
		_, _ = pool.Exec(ctx, "DELETE FROM organization_rate_limits WHERE organization_id = $1", testOrgID)
		_, _ = pool.Exec(ctx, "DELETE FROM organizations WHERE id = $1", testOrgID)
		pool.Close()
	}

	return pool, cleanup
}

func TestPostgresQueue_EnqueueDequeue(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := DefaultConfig()
	queue := NewPostgresQueue(pool, logger, cfg)

	ctx := context.Background()
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Enqueue a job
	payload := map[string]interface{}{
		"photo_id": "test-photo-123",
		"action":   "analyze",
	}

	job, err := queue.Enqueue(ctx, "photo_analysis", "analyze_photo", orgID, payload, nil)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, job.ID)
	assert.Equal(t, "photo_analysis", job.QueueName)
	assert.Equal(t, "analyze_photo", job.JobType)
	assert.Equal(t, orgID, job.OrganizationID)
	assert.Equal(t, JobStatusPending, job.Status)
	assert.Equal(t, 0, job.AttemptCount)
	assert.Equal(t, 3, job.MaxAttempts) // default

	// Dequeue the job
	dequeuedJob, err := queue.Dequeue(ctx, "worker-1", nil)
	require.NoError(t, err)
	require.NotNil(t, dequeuedJob)
	assert.Equal(t, job.ID, dequeuedJob.ID)
	assert.Equal(t, JobStatusProcessing, dequeuedJob.Status)
	assert.Equal(t, 1, dequeuedJob.AttemptCount)
	assert.Equal(t, "worker-1", dequeuedJob.WorkerID)
	assert.NotNil(t, dequeuedJob.StartedAt)

	// Trying to dequeue again should return nil (no more jobs)
	noJob, err := queue.Dequeue(ctx, "worker-2", nil)
	require.NoError(t, err)
	assert.Nil(t, noJob)
}

func TestPostgresQueue_Complete(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := DefaultConfig()
	queue := NewPostgresQueue(pool, logger, cfg)

	ctx := context.Background()
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Enqueue and dequeue
	job, err := queue.Enqueue(ctx, "test_queue", "test_job", orgID, map[string]interface{}{"key": "value"}, nil)
	require.NoError(t, err)

	dequeuedJob, err := queue.Dequeue(ctx, "worker-1", nil)
	require.NoError(t, err)
	require.NotNil(t, dequeuedJob)

	// Complete the job
	result := map[string]interface{}{
		"violations_found": 3,
		"processing_time":  "2.5s",
	}
	err = queue.Complete(ctx, dequeuedJob.ID, result)
	require.NoError(t, err)

	// Verify job status
	completedJob, err := queue.GetJob(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, JobStatusCompleted, completedJob.Status)
	assert.NotNil(t, completedJob.CompletedAt)
	// JSON unmarshaling converts numbers to float64, so compare carefully
	assert.Equal(t, "2.5s", completedJob.Result["processing_time"])
	assert.Equal(t, float64(3), completedJob.Result["violations_found"])
}

func TestPostgresQueue_Fail_WithRetry(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := DefaultConfig()
	queue := NewPostgresQueue(pool, logger, cfg)

	ctx := context.Background()
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Enqueue with max 3 attempts
	job, err := queue.Enqueue(ctx, "test_queue", "test_job", orgID, map[string]interface{}{"key": "value"}, nil)
	require.NoError(t, err)

	// First attempt - dequeue and fail
	dequeuedJob, err := queue.Dequeue(ctx, "worker-1", nil)
	require.NoError(t, err)
	require.NotNil(t, dequeuedJob)
	assert.Equal(t, 1, dequeuedJob.AttemptCount)

	err = queue.Fail(ctx, dequeuedJob.ID, "temporary error")
	require.NoError(t, err)

	// Job should be pending again with scheduled retry
	retriedJob, err := queue.GetJob(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, JobStatusPending, retriedJob.Status)
	assert.Equal(t, "temporary error", retriedJob.ErrorMessage)
	assert.True(t, retriedJob.ScheduledAt.After(time.Now()), "should be scheduled for future")
}

func TestPostgresQueue_Fail_Permanent(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := DefaultConfig()
	queue := NewPostgresQueue(pool, logger, cfg)

	ctx := context.Background()
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Enqueue with max 1 attempt
	opts := &EnqueueOptions{
		MaxAttempts: 1,
	}
	job, err := queue.Enqueue(ctx, "test_queue", "test_job", orgID, map[string]interface{}{"key": "value"}, opts)
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
	assert.NotNil(t, failedJob.CompletedAt)
}

func TestPostgresQueue_Priority(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := DefaultConfig()
	queue := NewPostgresQueue(pool, logger, cfg)

	ctx := context.Background()
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Enqueue jobs with different priorities
	lowPriorityJob, err := queue.Enqueue(ctx, "test_queue", "test_job", orgID,
		map[string]interface{}{"priority": "low"}, &EnqueueOptions{Priority: 1})
	require.NoError(t, err)

	highPriorityJob, err := queue.Enqueue(ctx, "test_queue", "test_job", orgID,
		map[string]interface{}{"priority": "high"}, &EnqueueOptions{Priority: 10})
	require.NoError(t, err)

	mediumPriorityJob, err := queue.Enqueue(ctx, "test_queue", "test_job", orgID,
		map[string]interface{}{"priority": "medium"}, &EnqueueOptions{Priority: 5})
	require.NoError(t, err)

	// Dequeue should return highest priority first
	job1, err := queue.Dequeue(ctx, "worker-1", nil)
	require.NoError(t, err)
	assert.Equal(t, highPriorityJob.ID, job1.ID)

	job2, err := queue.Dequeue(ctx, "worker-1", nil)
	require.NoError(t, err)
	assert.Equal(t, mediumPriorityJob.ID, job2.ID)

	job3, err := queue.Dequeue(ctx, "worker-1", nil)
	require.NoError(t, err)
	assert.Equal(t, lowPriorityJob.ID, job3.ID)
}

func TestPostgresQueue_ScheduledJobs(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := DefaultConfig()
	queue := NewPostgresQueue(pool, logger, cfg)

	ctx := context.Background()
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Enqueue job scheduled for future
	futureTime := time.Now().Add(1 * time.Hour)
	_, err := queue.Enqueue(ctx, "test_queue", "test_job", orgID,
		map[string]interface{}{"scheduled": true}, &EnqueueOptions{ScheduledAt: &futureTime})
	require.NoError(t, err)

	// Dequeue should return nil (job not ready yet)
	job, err := queue.Dequeue(ctx, "worker-1", nil)
	require.NoError(t, err)
	assert.Nil(t, job)

	// Enqueue job scheduled for now
	now := time.Now()
	immediateJob, err := queue.Enqueue(ctx, "test_queue", "test_job", orgID,
		map[string]interface{}{"scheduled": false}, &EnqueueOptions{ScheduledAt: &now})
	require.NoError(t, err)

	// This should be dequeued
	job, err = queue.Dequeue(ctx, "worker-1", nil)
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, immediateJob.ID, job.ID)
}

func TestPostgresQueue_ListJobs(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := DefaultConfig()
	queue := NewPostgresQueue(pool, logger, cfg)

	ctx := context.Background()
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Enqueue multiple jobs
	for i := 0; i < 5; i++ {
		_, err := queue.Enqueue(ctx, "test_queue", "test_job", orgID,
			map[string]interface{}{"index": i}, nil)
		require.NoError(t, err)
	}

	// List all jobs for organization
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

	// Dequeue one job
	_, err = queue.Dequeue(ctx, "worker-1", nil)
	require.NoError(t, err)

	// Filter by pending should now return 4
	jobs, err = queue.ListJobs(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, jobs, 4)

	// Filter by processing should return 1
	processingStatus := JobStatusProcessing
	filter.Status = &processingStatus
	jobs, err = queue.ListJobs(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, jobs, 1)
}

func TestPostgresQueue_GetQueueStats(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := DefaultConfig()
	queue := NewPostgresQueue(pool, logger, cfg)

	ctx := context.Background()
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Enqueue multiple jobs
	_, err := queue.Enqueue(ctx, "test_queue", "test_job", orgID, map[string]interface{}{"n": 1}, nil)
	require.NoError(t, err)
	job2, err := queue.Enqueue(ctx, "test_queue", "test_job", orgID, map[string]interface{}{"n": 2}, nil)
	require.NoError(t, err)
	job3, err := queue.Enqueue(ctx, "test_queue", "test_job", orgID, map[string]interface{}{"n": 3}, nil)
	require.NoError(t, err)

	// Dequeue and complete one
	dequeuedJob, err := queue.Dequeue(ctx, "worker-1", nil)
	require.NoError(t, err)
	err = queue.Complete(ctx, dequeuedJob.ID, map[string]interface{}{"status": "ok"})
	require.NoError(t, err)

	// Dequeue another (leave it processing)
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

	// Fail remaining jobs
	_, _ = queue.Dequeue(ctx, "worker-3", nil)
	_ = queue.Fail(ctx, job2.ID, "error 1")
	_ = queue.Fail(ctx, job3.ID, "error 2")
}

func TestPostgresQueue_RateLimiting(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := DefaultConfig()
	cfg.EnableRateLimiting = true
	cfg.DefaultMaxConcurrentJobs = 2
	cfg.DefaultMaxJobsPerHour = 10
	queue := NewPostgresQueue(pool, logger, cfg)

	ctx := context.Background()
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Initially should be able to process
	canProcess, err := queue.CanProcessJob(ctx, orgID, "test_queue")
	require.NoError(t, err)
	assert.True(t, canProcess)

	// Record job processed
	err = queue.RecordJobProcessed(ctx, orgID, "test_queue")
	require.NoError(t, err)

	// Should still be able to process (limit is 10 per hour)
	canProcess, err = queue.CanProcessJob(ctx, orgID, "test_queue")
	require.NoError(t, err)
	assert.True(t, canProcess)
}

func TestPostgresQueue_Delete(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := DefaultConfig()
	queue := NewPostgresQueue(pool, logger, cfg)

	ctx := context.Background()
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Enqueue a job
	job, err := queue.Enqueue(ctx, "test_queue", "test_job", orgID, map[string]interface{}{"key": "value"}, nil)
	require.NoError(t, err)

	// Delete the job
	err = queue.Delete(ctx, job.ID)
	require.NoError(t, err)

	// Job should not exist
	_, err = queue.GetJob(ctx, job.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "job not found")
}

func TestPostgresQueue_QueueFilter(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := DefaultConfig()
	queue := NewPostgresQueue(pool, logger, cfg)

	ctx := context.Background()
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Enqueue jobs in different queues
	_, err := queue.Enqueue(ctx, "queue_a", "test_job", orgID, map[string]interface{}{"q": "a"}, nil)
	require.NoError(t, err)
	_, err = queue.Enqueue(ctx, "queue_b", "test_job", orgID, map[string]interface{}{"q": "b"}, nil)
	require.NoError(t, err)
	_, err = queue.Enqueue(ctx, "queue_a", "test_job", orgID, map[string]interface{}{"q": "a2"}, nil)
	require.NoError(t, err)

	// Dequeue from specific queue
	opts := &DequeueOptions{
		QueueNames: []string{"queue_a"},
	}
	job, err := queue.Dequeue(ctx, "worker-1", opts)
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, "queue_a", job.QueueName)

	// Dequeue from any queue
	job, err = queue.Dequeue(ctx, "worker-2", nil)
	require.NoError(t, err)
	require.NotNil(t, job)
	// Could be either queue_a or queue_b
}
