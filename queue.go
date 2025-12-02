package queue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/yourusername/distributed-task-queue/internal/metrics"
	"github.com/yourusername/distributed-task-queue/internal/storage"
	"github.com/yourusername/distributed-task-queue/internal/task"
	"go.uber.org/zap"
)

// Queue manages task distribution and execution
type Queue struct {
	storage  storage.Storage
	logger   *zap.Logger
	handlers map[string]TaskHandler
	mu       sync.RWMutex
	
	// Channels for task distribution
	taskChannels map[task.Priority]chan *task.Task
	stopChan     chan struct{}
	wg           sync.WaitGroup
}

// TaskHandler is a function that processes a task
type TaskHandler func(ctx context.Context, t *task.Task) error

// Config holds queue configuration
type Config struct {
	Storage         storage.Storage
	Logger          *zap.Logger
	MaxWorkers      int
	PollInterval    time.Duration
	TaskTimeout     time.Duration
}

// NewQueue creates a new task queue
func NewQueue(cfg Config) *Queue {
	if cfg.Logger == nil {
		cfg.Logger, _ = zap.NewProduction()
	}
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 1 * time.Second
	}
	if cfg.TaskTimeout == 0 {
		cfg.TaskTimeout = 5 * time.Minute
	}

	q := &Queue{
		storage:  cfg.Storage,
		logger:   cfg.Logger,
		handlers: make(map[string]TaskHandler),
		taskChannels: map[task.Priority]chan *task.Task{
			task.PriorityCritical: make(chan *task.Task, 100),
			task.PriorityHigh:     make(chan *task.Task, 100),
			task.PriorityMedium:   make(chan *task.Task, 100),
			task.PriorityLow:      make(chan *task.Task, 100),
		},
		stopChan: make(chan struct{}),
	}

	return q
}

// RegisterHandler registers a handler for a specific task type
func (q *Queue) RegisterHandler(taskType string, handler TaskHandler) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.handlers[taskType] = handler
	q.logger.Info("registered task handler", zap.String("type", taskType))
}

// Submit adds a new task to the queue
func (q *Queue) Submit(ctx context.Context, t *task.Task) error {
	if err := q.storage.SaveTask(ctx, t); err != nil {
		return fmt.Errorf("failed to save task: %w", err)
	}

	metrics.TasksSubmitted.WithLabelValues(t.Type, fmt.Sprintf("%d", t.Priority)).Inc()
	metrics.QueueSize.WithLabelValues(fmt.Sprintf("%d", t.Priority)).Inc()

	q.logger.Info("task submitted",
		zap.String("id", t.ID),
		zap.String("type", t.Type),
		zap.Int("priority", int(t.Priority)),
	)

	// Try to send to channel (non-blocking)
	select {
	case q.taskChannels[t.Priority] <- t:
	default:
		// Channel full, will be picked up by polling
	}

	return nil
}

// GetTask retrieves a task by ID
func (q *Queue) GetTask(ctx context.Context, id string) (*task.Task, error) {
	return q.storage.GetTask(ctx, id)
}

// Start begins processing tasks
func (q *Queue) Start(ctx context.Context, numWorkers int) {
	q.logger.Info("starting queue", zap.Int("workers", numWorkers))

	// Start workers for each priority level
	for priority := range q.taskChannels {
		for i := 0; i < numWorkers; i++ {
			q.wg.Add(1)
			go q.worker(ctx, priority, i)
		}
	}

	// Start poller to refill channels from storage
	q.wg.Add(1)
	go q.poller(ctx)
}

// Stop gracefully stops the queue
func (q *Queue) Stop() {
	q.logger.Info("stopping queue")
	close(q.stopChan)
	q.wg.Wait()
	q.logger.Info("queue stopped")
}

// worker processes tasks from a priority channel
func (q *Queue) worker(ctx context.Context, priority task.Priority, workerID int) {
	defer q.wg.Done()

	workerName := fmt.Sprintf("worker-%d-%d", priority, workerID)
	q.logger.Info("worker started", zap.String("worker", workerName))
	metrics.WorkersActive.Inc()
	defer metrics.WorkersActive.Dec()

	for {
		select {
		case <-q.stopChan:
			q.logger.Info("worker stopping", zap.String("worker", workerName))
			return
		case <-ctx.Done():
			return
		case t := <-q.taskChannels[priority]:
			q.processTask(ctx, t, workerName)
		}
	}
}

// processTask executes a single task
func (q *Queue) processTask(ctx context.Context, t *task.Task, workerID string) {
	startTime := time.Now()
	
	q.logger.Info("processing task",
		zap.String("id", t.ID),
		zap.String("type", t.Type),
		zap.String("worker", workerID),
	)

	// Mark task as started
	t.MarkStarted(workerID)
	if err := q.storage.UpdateTask(ctx, t); err != nil {
		q.logger.Error("failed to update task status", zap.Error(err))
	}

	// Get handler
	q.mu.RLock()
	handler, exists := q.handlers[t.Type]
	q.mu.RUnlock()

	if !exists {
		q.logger.Error("no handler for task type", zap.String("type", t.Type))
		t.MarkFailed(fmt.Errorf("no handler for task type: %s", t.Type))
		q.storage.UpdateTask(ctx, t)
		metrics.TasksProcessed.WithLabelValues(t.Type, "failed").Inc()
		return
	}

	// Execute with timeout
	taskCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	err := handler(taskCtx, t)
	duration := time.Since(startTime)

	// Update metrics
	metrics.TaskDuration.WithLabelValues(t.Type).Observe(duration.Seconds())
	metrics.QueueSize.WithLabelValues(fmt.Sprintf("%d", t.Priority)).Dec()

	if err != nil {
		q.logger.Error("task failed",
			zap.String("id", t.ID),
			zap.Error(err),
			zap.Duration("duration", duration),
		)

		if t.CanRetry() {
			t.MarkRetrying()
			q.storage.UpdateTask(ctx, t)
			metrics.TaskRetries.WithLabelValues(t.Type).Inc()

			// Re-submit with exponential backoff
			backoff := time.Duration(t.RetryCount*t.RetryCount) * time.Second
			time.Sleep(backoff)
			q.taskChannels[t.Priority] <- t
		} else {
			t.MarkFailed(err)
			q.storage.UpdateTask(ctx, t)
			metrics.TasksProcessed.WithLabelValues(t.Type, "failed").Inc()
		}
	} else {
		t.MarkCompleted()
		q.storage.UpdateTask(ctx, t)
		metrics.TasksProcessed.WithLabelValues(t.Type, "completed").Inc()
		
		q.logger.Info("task completed",
			zap.String("id", t.ID),
			zap.Duration("duration", duration),
		)
	}
}

// poller continuously checks storage for pending tasks
func (q *Queue) poller(ctx context.Context) {
	defer q.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-q.stopChan:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			q.pollPendingTasks(ctx)
		}
	}
}

// pollPendingTasks retrieves pending tasks from storage
func (q *Queue) pollPendingTasks(ctx context.Context) {
	tasks, err := q.storage.GetTasksByStatus(ctx, task.StatusPending, 50)
	if err != nil {
		q.logger.Error("failed to poll tasks", zap.Error(err))
		return
	}

	for _, t := range tasks {
		select {
		case q.taskChannels[t.Priority] <- t:
		default:
			// Channel full, will be picked up in next poll
		}
	}

	// Also check for retrying tasks
	retryingTasks, err := q.storage.GetTasksByStatus(ctx, task.StatusRetrying, 20)
	if err == nil {
		for _, t := range retryingTasks {
			select {
			case q.taskChannels[t.Priority] <- t:
			default:
			}
		}
	}
}

// GetStats returns queue statistics
func (q *Queue) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	for status := range map[task.Status]bool{
		task.StatusPending:    true,
		task.StatusProcessing: true,
		task.StatusCompleted:  true,
		task.StatusFailed:     true,
	} {
		tasks, err := q.storage.GetTasksByStatus(ctx, status, 1000)
		if err != nil {
			return nil, err
		}
		stats[string(status)] = len(tasks)
	}

	return stats, nil
}
