package queue

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkerPool_RegisterHandler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockQueue := NewMockQueue()
	cfg := DefaultConfig()

	pool := NewWorkerPool(mockQueue, logger, cfg)

	// Register a handler
	handler := func(ctx context.Context, job *Job) (map[string]interface{}, error) {
		return map[string]interface{}{"status": "ok"}, nil
	}

	pool.RegisterHandler("test_job", handler)

	// Verify handler was registered
	registeredHandler, exists := pool.GetHandler("test_job")
	assert.True(t, exists)
	assert.NotNil(t, registeredHandler)
}

func TestWorkerPool_StartStop(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockQueue := NewMockQueue()
	cfg := DefaultConfig()
	cfg.WorkerCount = 2

	pool := NewWorkerPool(mockQueue, logger, cfg)

	ctx := context.Background()

	// Start the pool
	err := pool.Start(ctx, []string{"test_queue"})
	require.NoError(t, err)

	// Starting again should error
	err = pool.Start(ctx, []string{"test_queue"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already started")

	// Stop the pool
	err = pool.Stop()
	require.NoError(t, err)

	// Stopping again should error
	err = pool.Stop()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not started")
}

func TestWorkerPool_ProcessJob_Success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockQueue := NewMockQueue()
	cfg := DefaultConfig()
	cfg.WorkerCount = 1
	cfg.PollInterval = 50 * time.Millisecond

	pool := NewWorkerPool(mockQueue, logger, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Track handler execution
	var handlerCalled bool
	var handlerMu sync.Mutex
	var processedJob *Job

	handler := func(ctx context.Context, job *Job) (map[string]interface{}, error) {
		handlerMu.Lock()
		defer handlerMu.Unlock()
		handlerCalled = true
		processedJob = job
		return map[string]interface{}{"result": "success"}, nil
	}

	pool.RegisterHandler("test_job", handler)

	// Enqueue a job
	orgID := uuid.New()
	job, err := mockQueue.Enqueue(ctx, "test_queue", "test_job", orgID,
		map[string]interface{}{"data": "test"}, nil)
	require.NoError(t, err)

	// Start workers
	err = pool.Start(ctx, []string{"test_queue"})
	require.NoError(t, err)

	// Wait for job to be processed
	time.Sleep(500 * time.Millisecond)

	// Stop workers
	err = pool.Stop()
	require.NoError(t, err)

	// Verify handler was called
	handlerMu.Lock()
	assert.True(t, handlerCalled)
	assert.NotNil(t, processedJob)
	assert.Equal(t, job.ID, processedJob.ID)
	handlerMu.Unlock()

	// Verify job was completed
	completedJob, err := mockQueue.GetJob(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, JobStatusCompleted, completedJob.Status)
	assert.NotNil(t, completedJob.Result)
	assert.Equal(t, "success", completedJob.Result["result"])
}

func TestWorkerPool_ProcessJob_Failure(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockQueue := NewMockQueue()
	cfg := DefaultConfig()
	cfg.WorkerCount = 1
	cfg.PollInterval = 50 * time.Millisecond

	pool := NewWorkerPool(mockQueue, logger, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Handler that fails
	handler := func(ctx context.Context, job *Job) (map[string]interface{}, error) {
		return nil, errors.New("processing failed")
	}

	pool.RegisterHandler("test_job", handler)

	// Enqueue a job
	orgID := uuid.New()
	job, err := mockQueue.Enqueue(ctx, "test_queue", "test_job", orgID,
		map[string]interface{}{"data": "test"}, &EnqueueOptions{MaxAttempts: 1})
	require.NoError(t, err)

	// Start workers
	err = pool.Start(ctx, []string{"test_queue"})
	require.NoError(t, err)

	// Wait for job to be processed
	time.Sleep(500 * time.Millisecond)

	// Stop workers
	err = pool.Stop()
	require.NoError(t, err)

	// Verify job failed
	failedJob, err := mockQueue.GetJob(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, JobStatusFailed, failedJob.Status)
	assert.Contains(t, failedJob.ErrorMessage, "processing failed")
}

func TestWorkerPool_ProcessJob_NoHandler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockQueue := NewMockQueue()
	cfg := DefaultConfig()
	cfg.WorkerCount = 1
	cfg.PollInterval = 50 * time.Millisecond

	pool := NewWorkerPool(mockQueue, logger, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Don't register a handler for this job type

	// Enqueue a job
	orgID := uuid.New()
	job, err := mockQueue.Enqueue(ctx, "test_queue", "unknown_job", orgID,
		map[string]interface{}{"data": "test"}, &EnqueueOptions{MaxAttempts: 1})
	require.NoError(t, err)

	// Start workers
	err = pool.Start(ctx, []string{"test_queue"})
	require.NoError(t, err)

	// Wait for job to be processed
	time.Sleep(500 * time.Millisecond)

	// Stop workers
	err = pool.Stop()
	require.NoError(t, err)

	// Verify job failed with "no handler" error
	failedJob, err := mockQueue.GetJob(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, JobStatusFailed, failedJob.Status)
	assert.Contains(t, failedJob.ErrorMessage, "no handler registered")
}

func TestWorkerPool_ProcessJob_Timeout(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockQueue := NewMockQueue()
	cfg := DefaultConfig()
	cfg.WorkerCount = 1
	cfg.PollInterval = 50 * time.Millisecond
	cfg.JobTimeout = 100 * time.Millisecond // Very short timeout

	pool := NewWorkerPool(mockQueue, logger, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Handler that takes too long
	handler := func(ctx context.Context, job *Job) (map[string]interface{}, error) {
		select {
		case <-time.After(1 * time.Second):
			return map[string]interface{}{"status": "ok"}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	pool.RegisterHandler("slow_job", handler)

	// Enqueue a job
	orgID := uuid.New()
	job, err := mockQueue.Enqueue(ctx, "test_queue", "slow_job", orgID,
		map[string]interface{}{"data": "test"}, &EnqueueOptions{MaxAttempts: 1})
	require.NoError(t, err)

	// Start workers
	err = pool.Start(ctx, []string{"test_queue"})
	require.NoError(t, err)

	// Wait for job to be processed
	time.Sleep(500 * time.Millisecond)

	// Stop workers
	err = pool.Stop()
	require.NoError(t, err)

	// Verify job failed with timeout error
	failedJob, err := mockQueue.GetJob(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, JobStatusFailed, failedJob.Status)
	assert.Contains(t, failedJob.ErrorMessage, "context deadline exceeded")
}

func TestWorkerPool_MultipleWorkers(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockQueue := NewMockQueue()
	cfg := DefaultConfig()
	cfg.WorkerCount = 3
	cfg.PollInterval = 50 * time.Millisecond

	pool := NewWorkerPool(mockQueue, logger, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Track processed jobs
	var processedCount int
	var mu sync.Mutex

	handler := func(ctx context.Context, job *Job) (map[string]interface{}, error) {
		mu.Lock()
		processedCount++
		mu.Unlock()
		time.Sleep(100 * time.Millisecond) // Simulate work
		return map[string]interface{}{"status": "ok"}, nil
	}

	pool.RegisterHandler("test_job", handler)

	// Enqueue multiple jobs
	orgID := uuid.New()
	for i := 0; i < 5; i++ {
		_, err := mockQueue.Enqueue(ctx, "test_queue", "test_job", orgID,
			map[string]interface{}{"index": i}, nil)
		require.NoError(t, err)
	}

	// Start workers
	err := pool.Start(ctx, []string{"test_queue"})
	require.NoError(t, err)

	// Wait for jobs to be processed
	time.Sleep(1 * time.Second)

	// Stop workers
	err = pool.Stop()
	require.NoError(t, err)

	// Verify all jobs were processed
	mu.Lock()
	assert.Equal(t, 5, processedCount)
	mu.Unlock()
}

func TestWorkerPool_GracefulShutdown(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockQueue := NewMockQueue()
	cfg := DefaultConfig()
	cfg.WorkerCount = 1
	cfg.PollInterval = 50 * time.Millisecond
	cfg.ShutdownTimeout = 2 * time.Second

	pool := NewWorkerPool(mockQueue, logger, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Handler that takes some time
	var handlerCompleted bool
	var mu sync.Mutex

	handler := func(ctx context.Context, job *Job) (map[string]interface{}, error) {
		time.Sleep(500 * time.Millisecond)
		mu.Lock()
		handlerCompleted = true
		mu.Unlock()
		return map[string]interface{}{"status": "ok"}, nil
	}

	pool.RegisterHandler("test_job", handler)

	// Enqueue a job
	orgID := uuid.New()
	_, err := mockQueue.Enqueue(ctx, "test_queue", "test_job", orgID,
		map[string]interface{}{"data": "test"}, nil)
	require.NoError(t, err)

	// Start workers
	err = pool.Start(ctx, []string{"test_queue"})
	require.NoError(t, err)

	// Wait a bit for job to start processing
	time.Sleep(200 * time.Millisecond)

	// Stop workers (should wait for handler to complete)
	err = pool.Stop()
	require.NoError(t, err)

	// Verify handler completed
	mu.Lock()
	assert.True(t, handlerCompleted)
	mu.Unlock()
}

func TestWorkerPool_EnqueueJob(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockQueue := NewMockQueue()
	cfg := DefaultConfig()

	pool := NewWorkerPool(mockQueue, logger, cfg)

	ctx := context.Background()
	orgID := uuid.New()

	// Use the convenience method to enqueue
	job, err := pool.EnqueueJob(ctx, "test_queue", "test_job", orgID,
		map[string]interface{}{"data": "test"}, nil)

	require.NoError(t, err)
	assert.NotNil(t, job)
	assert.Equal(t, "test_queue", job.QueueName)
	assert.Equal(t, "test_job", job.JobType)

	// Verify job exists in queue
	retrievedJob, err := mockQueue.GetJob(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, job.ID, retrievedJob.ID)
}

func TestWorkerPool_ContextCancellation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockQueue := NewMockQueue()
	cfg := DefaultConfig()
	cfg.WorkerCount = 1
	cfg.PollInterval = 50 * time.Millisecond

	pool := NewWorkerPool(mockQueue, logger, cfg)

	ctx, cancel := context.WithCancel(context.Background())

	handler := func(ctx context.Context, job *Job) (map[string]interface{}, error) {
		return map[string]interface{}{"status": "ok"}, nil
	}

	pool.RegisterHandler("test_job", handler)

	// Start workers
	err := pool.Start(ctx, []string{"test_queue"})
	require.NoError(t, err)

	// Cancel context
	cancel()

	// Wait a bit
	time.Sleep(200 * time.Millisecond)

	// Stop should complete quickly since context is cancelled
	err = pool.Stop()
	require.NoError(t, err)
}

func TestWorkerPool_MultipleQueues(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockQueue := NewMockQueue()
	cfg := DefaultConfig()
	cfg.WorkerCount = 2
	cfg.PollInterval = 50 * time.Millisecond

	pool := NewWorkerPool(mockQueue, logger, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Track which queues were processed
	var processedQueues []string
	var mu sync.Mutex

	handler := func(ctx context.Context, job *Job) (map[string]interface{}, error) {
		mu.Lock()
		processedQueues = append(processedQueues, job.QueueName)
		mu.Unlock()
		return map[string]interface{}{"status": "ok"}, nil
	}

	pool.RegisterHandler("test_job", handler)

	// Enqueue jobs in different queues
	orgID := uuid.New()
	_, err := mockQueue.Enqueue(ctx, "queue_a", "test_job", orgID,
		map[string]interface{}{"queue": "a"}, nil)
	require.NoError(t, err)

	_, err = mockQueue.Enqueue(ctx, "queue_b", "test_job", orgID,
		map[string]interface{}{"queue": "b"}, nil)
	require.NoError(t, err)

	// Start workers for both queues
	err = pool.Start(ctx, []string{"queue_a", "queue_b"})
	require.NoError(t, err)

	// Wait for jobs to be processed
	time.Sleep(500 * time.Millisecond)

	// Stop workers
	err = pool.Stop()
	require.NoError(t, err)

	// Verify both queues were processed
	mu.Lock()
	assert.Len(t, processedQueues, 2)
	assert.Contains(t, processedQueues, "queue_a")
	assert.Contains(t, processedQueues, "queue_b")
	mu.Unlock()
}
