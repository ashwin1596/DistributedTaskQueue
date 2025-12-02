package queue

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/distributed-task-queue/internal/storage"
	"github.com/yourusername/distributed-task-queue/internal/task"
	"go.uber.org/zap"
)

func TestQueue_Submit(t *testing.T) {
	store := storage.NewMemoryStorage()
	logger, _ := zap.NewDevelopment()
	
	q := NewQueue(Config{
		Storage: store,
		Logger:  logger,
	})

	ctx := context.Background()
	testTask := task.NewTask("test_task", task.PriorityHigh, map[string]interface{}{
		"key": "value",
	})

	err := q.Submit(ctx, testTask)
	require.NoError(t, err)

	// Verify task was saved
	retrieved, err := store.GetTask(ctx, testTask.ID)
	require.NoError(t, err)
	assert.Equal(t, testTask.ID, retrieved.ID)
	assert.Equal(t, task.StatusPending, retrieved.Status)
}

func TestQueue_ProcessTask_Success(t *testing.T) {
	store := storage.NewMemoryStorage()
	logger, _ := zap.NewDevelopment()
	
	q := NewQueue(Config{
		Storage: store,
		Logger:  logger,
	})

	// Register a successful handler
	handlerCalled := false
	q.RegisterHandler("test_task", func(ctx context.Context, t *task.Task) error {
		handlerCalled = true
		return nil
	})

	ctx := context.Background()
	testTask := task.NewTask("test_task", task.PriorityHigh, map[string]interface{}{
		"key": "value",
	})

	err := q.Submit(ctx, testTask)
	require.NoError(t, err)

	// Start queue with 1 worker
	q.Start(ctx, 1)
	
	// Wait for processing
	time.Sleep(2 * time.Second)
	
	q.Stop()

	assert.True(t, handlerCalled, "handler should have been called")

	// Verify task was completed
	retrieved, err := store.GetTask(ctx, testTask.ID)
	require.NoError(t, err)
	assert.Equal(t, task.StatusCompleted, retrieved.Status)
}

func TestQueue_ProcessTask_WithRetry(t *testing.T) {
	store := storage.NewMemoryStorage()
	logger, _ := zap.NewDevelopment()
	
	q := NewQueue(Config{
		Storage: store,
		Logger:  logger,
	})

	// Register a handler that fails then succeeds
	callCount := 0
	q.RegisterHandler("test_task", func(ctx context.Context, t *task.Task) error {
		callCount++
		if callCount == 1 {
			return errors.New("temporary failure")
		}
		return nil
	})

	ctx := context.Background()
	testTask := task.NewTask("test_task", task.PriorityHigh, map[string]interface{}{
		"key": "value",
	})
	testTask.MaxRetries = 3

	err := q.Submit(ctx, testTask)
	require.NoError(t, err)

	// Start queue
	q.Start(ctx, 1)
	
	// Wait for processing and retry
	time.Sleep(5 * time.Second)
	
	q.Stop()

	assert.Equal(t, 2, callCount, "handler should have been called twice")

	// Verify task was eventually completed
	retrieved, err := store.GetTask(ctx, testTask.ID)
	require.NoError(t, err)
	assert.Equal(t, task.StatusCompleted, retrieved.Status)
	assert.Equal(t, 1, retrieved.RetryCount)
}

func TestQueue_ProcessTask_MaxRetriesExceeded(t *testing.T) {
	store := storage.NewMemoryStorage()
	logger, _ := zap.NewDevelopment()
	
	q := NewQueue(Config{
		Storage: store,
		Logger:  logger,
	})

	// Register a handler that always fails
	q.RegisterHandler("test_task", func(ctx context.Context, t *task.Task) error {
		return errors.New("permanent failure")
	})

	ctx := context.Background()
	testTask := task.NewTask("test_task", task.PriorityHigh, map[string]interface{}{
		"key": "value",
	})
	testTask.MaxRetries = 2

	err := q.Submit(ctx, testTask)
	require.NoError(t, err)

	// Start queue
	q.Start(ctx, 1)
	
	// Wait for all retries
	time.Sleep(8 * time.Second)
	
	q.Stop()

	// Verify task failed after max retries
	retrieved, err := store.GetTask(ctx, testTask.ID)
	require.NoError(t, err)
	assert.Equal(t, task.StatusFailed, retrieved.Status)
	assert.Equal(t, 2, retrieved.RetryCount)
}

func TestQueue_PriorityOrdering(t *testing.T) {
	store := storage.NewMemoryStorage()
	logger, _ := zap.NewDevelopment()
	
	q := NewQueue(Config{
		Storage: store,
		Logger:  logger,
	})

	processedOrder := make([]string, 0)
	q.RegisterHandler("test_task", func(ctx context.Context, t *task.Task) error {
		processedOrder = append(processedOrder, t.ID)
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	ctx := context.Background()

	// Submit tasks in reverse priority order
	lowTask := task.NewTask("test_task", task.PriorityLow, nil)
	medTask := task.NewTask("test_task", task.PriorityMedium, nil)
	highTask := task.NewTask("test_task", task.PriorityHigh, nil)

	q.Submit(ctx, lowTask)
	q.Submit(ctx, medTask)
	q.Submit(ctx, highTask)

	// Start queue with 1 worker to ensure sequential processing
	q.Start(ctx, 1)
	
	time.Sleep(2 * time.Second)
	
	q.Stop()

	// High priority should be processed first
	require.Len(t, processedOrder, 3)
	assert.Equal(t, highTask.ID, processedOrder[0])
}

func TestQueue_GetStats(t *testing.T) {
	store := storage.NewMemoryStorage()
	logger, _ := zap.NewDevelopment()
	
	q := NewQueue(Config{
		Storage: store,
		Logger:  logger,
	})

	ctx := context.Background()

	// Submit various tasks
	for i := 0; i < 5; i++ {
		testTask := task.NewTask("test_task", task.PriorityMedium, nil)
		q.Submit(ctx, testTask)
	}

	stats, err := q.GetStats(ctx)
	require.NoError(t, err)
	
	assert.NotNil(t, stats)
	pendingCount, ok := stats["pending"].(int)
	assert.True(t, ok)
	assert.Equal(t, 5, pendingCount)
}

func TestTask_Lifecycle(t *testing.T) {
	testTask := task.NewTask("test_task", task.PriorityHigh, map[string]interface{}{
		"key": "value",
	})

	assert.Equal(t, task.StatusPending, testTask.Status)
	assert.True(t, testTask.CanRetry())

	// Mark as started
	testTask.MarkStarted("worker-1")
	assert.Equal(t, task.StatusProcessing, testTask.Status)
	assert.NotNil(t, testTask.StartedAt)
	assert.Equal(t, "worker-1", testTask.WorkerID)

	// Mark as completed
	testTask.MarkCompleted()
	assert.Equal(t, task.StatusCompleted, testTask.Status)
	assert.NotNil(t, testTask.CompletedAt)
}

func TestTask_Retries(t *testing.T) {
	testTask := task.NewTask("test_task", task.PriorityHigh, nil)
	testTask.MaxRetries = 3

	assert.True(t, testTask.CanRetry())
	
	testTask.MarkRetrying()
	assert.Equal(t, 1, testTask.RetryCount)
	assert.True(t, testTask.CanRetry())
	
	testTask.MarkRetrying()
	testTask.MarkRetrying()
	assert.Equal(t, 3, testTask.RetryCount)
	assert.False(t, testTask.CanRetry())
}
