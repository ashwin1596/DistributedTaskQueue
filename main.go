package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yourusername/distributed-task-queue/internal/queue"
	"github.com/yourusername/distributed-task-queue/internal/storage"
	"github.com/yourusername/distributed-task-queue/internal/task"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("failed to create logger: %v", err))
	}
	defer logger.Sync()

	// Get configuration from environment
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	workerID := getEnv("WORKER_ID", "worker-1")

	logger.Info("starting worker", zap.String("worker_id", workerID))

	// Initialize storage
	store, err := storage.NewRedisStorage(redisAddr, redisPassword, 0)
	if err != nil {
		logger.Fatal("failed to initialize storage", zap.Error(err))
	}
	defer store.Close()

	// Initialize queue
	q := queue.NewQueue(queue.Config{
		Storage:      store,
		Logger:       logger,
		PollInterval: 1 * time.Second,
		TaskTimeout:  5 * time.Minute,
	})

	// Register task handlers
	registerWorkerHandlers(q, logger)

	// Start queue workers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	numWorkers := 3 // Number of concurrent workers
	q.Start(ctx, numWorkers)
	defer q.Stop()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("shutting down worker...")
	logger.Info("worker stopped")
}

// registerWorkerHandlers registers task handlers for this worker
func registerWorkerHandlers(q *queue.Queue, logger *zap.Logger) {
	// Email handler
	q.RegisterHandler("send_email", func(ctx context.Context, t *task.Task) error {
		logger.Info("sending email", zap.String("task_id", t.ID))
		
		// Simulate work
		time.Sleep(2 * time.Second)
		
		recipient, _ := t.Payload["recipient"].(string)
		subject, _ := t.Payload["subject"].(string)
		
		logger.Info("email sent successfully",
			zap.String("recipient", recipient),
			zap.String("subject", subject),
		)
		return nil
	})

	// Image processing handler
	q.RegisterHandler("process_image", func(ctx context.Context, t *task.Task) error {
		logger.Info("processing image", zap.String("task_id", t.ID))
		
		// Simulate work
		time.Sleep(5 * time.Second)
		
		imageURL, _ := t.Payload["image_url"].(string)
		logger.Info("image processed", zap.String("url", imageURL))
		return nil
	})

	// Data export handler
	q.RegisterHandler("export_data", func(ctx context.Context, t *task.Task) error {
		logger.Info("exporting data", zap.String("task_id", t.ID))
		
		// Simulate work
		time.Sleep(10 * time.Second)
		
		format, _ := t.Payload["format"].(string)
		logger.Info("data exported", zap.String("format", format))
		return nil
	})

	// Webhook handler
	q.RegisterHandler("call_webhook", func(ctx context.Context, t *task.Task) error {
		logger.Info("calling webhook", zap.String("task_id", t.ID))
		
		// Simulate work
		time.Sleep(3 * time.Second)
		
		url, _ := t.Payload["url"].(string)
		logger.Info("webhook called", zap.String("url", url))
		return nil
	})

	// Batch processing handler
	q.RegisterHandler("batch_process", func(ctx context.Context, t *task.Task) error {
		logger.Info("batch processing", zap.String("task_id", t.ID))
		
		// Simulate work
		time.Sleep(15 * time.Second)
		
		batchSize, _ := t.Payload["batch_size"].(float64)
		logger.Info("batch processed", zap.Float64("size", batchSize))
		return nil
	})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
