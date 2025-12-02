package queue

import (
	"context"
	"testing"

	"github.com/yourusername/distributed-task-queue/internal/storage"
	"github.com/yourusername/distributed-task-queue/internal/task"
	"go.uber.org/zap"
)

func BenchmarkQueue_Submit(b *testing.B) {
	store := storage.NewMemoryStorage()
	logger, _ := zap.NewDevelopment()
	
	q := NewQueue(Config{
		Storage: store,
		Logger:  logger,
	})

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := task.NewTask("benchmark_task", task.PriorityMedium, map[string]interface{}{
			"index": i,
		})
		q.Submit(ctx, t)
	}
}

func BenchmarkQueue_ProcessTask(b *testing.B) {
	store := storage.NewMemoryStorage()
	logger, _ := zap.NewDevelopment()
	
	q := NewQueue(Config{
		Storage: store,
		Logger:  logger,
	})

	q.RegisterHandler("benchmark_task", func(ctx context.Context, t *task.Task) error {
		return nil
	})

	ctx := context.Background()

	// Pre-populate with tasks
	tasks := make([]*task.Task, b.N)
	for i := 0; i < b.N; i++ {
		t := task.NewTask("benchmark_task", task.PriorityMedium, map[string]interface{}{
			"index": i,
		})
		tasks[i] = t
		q.Submit(ctx, t)
	}

	q.Start(ctx, 4)
	defer q.Stop()

	b.ResetTimer()
	// Workers will process tasks in background
	// This measures throughput
}

func BenchmarkTask_Serialization(b *testing.B) {
	t := task.NewTask("benchmark_task", task.PriorityMedium, map[string]interface{}{
		"key1": "value1",
		"key2": 12345,
		"key3": map[string]interface{}{
			"nested": "data",
		},
	})

	b.Run("ToJSON", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := t.ToJSON()
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	data, _ := t.ToJSON()

	b.Run("FromJSON", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := task.FromJSON(data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkStorage_Operations(b *testing.B) {
	store := storage.NewMemoryStorage()
	ctx := context.Background()

	t := task.NewTask("benchmark_task", task.PriorityMedium, nil)

	b.Run("SaveTask", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			store.SaveTask(ctx, t)
		}
	})

	b.Run("GetTask", func(b *testing.B) {
		store.SaveTask(ctx, t)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			store.GetTask(ctx, t.ID)
		}
	})

	b.Run("UpdateTask", func(b *testing.B) {
		store.SaveTask(ctx, t)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			store.UpdateTask(ctx, t)
		}
	})
}
