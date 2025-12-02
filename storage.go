package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/yourusername/distributed-task-queue/internal/task"
)

// Storage defines the interface for task persistence
type Storage interface {
	SaveTask(ctx context.Context, t *task.Task) error
	GetTask(ctx context.Context, id string) (*task.Task, error)
	UpdateTask(ctx context.Context, t *task.Task) error
	DeleteTask(ctx context.Context, id string) error
	GetTasksByStatus(ctx context.Context, status task.Status, limit int) ([]*task.Task, error)
	Close() error
}

// RedisStorage implements Storage using Redis
type RedisStorage struct {
	client *redis.Client
}

// NewRedisStorage creates a new Redis storage backend
func NewRedisStorage(addr, password string, db int) (*RedisStorage, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisStorage{client: client}, nil
}

// SaveTask persists a task to Redis
func (r *RedisStorage) SaveTask(ctx context.Context, t *task.Task) error {
	data, err := t.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize task: %w", err)
	}

	key := fmt.Sprintf("task:%s", t.ID)
	if err := r.client.Set(ctx, key, data, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to save task: %w", err)
	}

	// Add to status index
	statusKey := fmt.Sprintf("tasks:status:%s", t.Status)
	score := float64(t.Priority)*1000000 + float64(t.CreatedAt.Unix())
	if err := r.client.ZAdd(ctx, statusKey, &redis.Z{
		Score:  score,
		Member: t.ID,
	}).Err(); err != nil {
		return fmt.Errorf("failed to index task: %w", err)
	}

	return nil
}

// GetTask retrieves a task from Redis
func (r *RedisStorage) GetTask(ctx context.Context, id string) (*task.Task, error) {
	key := fmt.Sprintf("task:%s", id)
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("task not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	return task.FromJSON(data)
}

// UpdateTask updates an existing task
func (r *RedisStorage) UpdateTask(ctx context.Context, t *task.Task) error {
	// Remove from old status index
	oldTask, err := r.GetTask(ctx, t.ID)
	if err != nil {
		return err
	}

	if oldTask.Status != t.Status {
		oldStatusKey := fmt.Sprintf("tasks:status:%s", oldTask.Status)
		r.client.ZRem(ctx, oldStatusKey, t.ID)
	}

	// Save updated task
	return r.SaveTask(ctx, t)
}

// DeleteTask removes a task from Redis
func (r *RedisStorage) DeleteTask(ctx context.Context, id string) error {
	t, err := r.GetTask(ctx, id)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("task:%s", id)
	statusKey := fmt.Sprintf("tasks:status:%s", t.Status)

	pipe := r.client.Pipeline()
	pipe.Del(ctx, key)
	pipe.ZRem(ctx, statusKey, id)
	_, err = pipe.Exec(ctx)

	return err
}

// GetTasksByStatus retrieves tasks with a specific status
func (r *RedisStorage) GetTasksByStatus(ctx context.Context, status task.Status, limit int) ([]*task.Task, error) {
	statusKey := fmt.Sprintf("tasks:status:%s", status)
	
	// Get task IDs ordered by priority and creation time (descending)
	ids, err := r.client.ZRevRange(ctx, statusKey, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get task IDs: %w", err)
	}

	tasks := make([]*task.Task, 0, len(ids))
	for _, id := range ids {
		t, err := r.GetTask(ctx, id)
		if err != nil {
			continue // Skip tasks that can't be retrieved
		}
		tasks = append(tasks, t)
	}

	return tasks, nil
}

// Close closes the Redis connection
func (r *RedisStorage) Close() error {
	return r.client.Close()
}

// MemoryStorage implements Storage using in-memory map (for testing)
type MemoryStorage struct {
	tasks map[string]*task.Task
}

// NewMemoryStorage creates a new in-memory storage backend
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		tasks: make(map[string]*task.Task),
	}
}

func (m *MemoryStorage) SaveTask(ctx context.Context, t *task.Task) error {
	data, _ := json.Marshal(t)
	var taskCopy task.Task
	json.Unmarshal(data, &taskCopy)
	m.tasks[t.ID] = &taskCopy
	return nil
}

func (m *MemoryStorage) GetTask(ctx context.Context, id string) (*task.Task, error) {
	t, ok := m.tasks[id]
	if !ok {
		return nil, fmt.Errorf("task not found: %s", id)
	}
	data, _ := json.Marshal(t)
	var taskCopy task.Task
	json.Unmarshal(data, &taskCopy)
	return &taskCopy, nil
}

func (m *MemoryStorage) UpdateTask(ctx context.Context, t *task.Task) error {
	return m.SaveTask(ctx, t)
}

func (m *MemoryStorage) DeleteTask(ctx context.Context, id string) error {
	delete(m.tasks, id)
	return nil
}

func (m *MemoryStorage) GetTasksByStatus(ctx context.Context, status task.Status, limit int) ([]*task.Task, error) {
	var tasks []*task.Task
	for _, t := range m.tasks {
		if t.Status == status {
			tasks = append(tasks, t)
			if len(tasks) >= limit {
				break
			}
		}
	}
	return tasks, nil
}

func (m *MemoryStorage) Close() error {
	return nil
}
