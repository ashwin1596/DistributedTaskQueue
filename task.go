package task

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Priority represents task priority levels
type Priority int

const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
	PriorityCritical
)

// Status represents the current state of a task
type Status string

const (
	StatusPending    Status = "pending"
	StatusProcessing Status = "processing"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
	StatusRetrying   Status = "retrying"
)

// Task represents a unit of work to be executed
type Task struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Priority    Priority               `json:"priority"`
	Status      Status                 `json:"status"`
	Payload     map[string]interface{} `json:"payload"`
	MaxRetries  int                    `json:"max_retries"`
	RetryCount  int                    `json:"retry_count"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Error       string                 `json:"error,omitempty"`
	WorkerID    string                 `json:"worker_id,omitempty"`
}

// NewTask creates a new task with default values
func NewTask(taskType string, priority Priority, payload map[string]interface{}) *Task {
	return &Task{
		ID:         uuid.New().String(),
		Type:       taskType,
		Priority:   priority,
		Status:     StatusPending,
		Payload:    payload,
		MaxRetries: 3,
		RetryCount: 0,
		CreatedAt:  time.Now(),
	}
}

// ToJSON serializes the task to JSON
func (t *Task) ToJSON() ([]byte, error) {
	return json.Marshal(t)
}

// FromJSON deserializes a task from JSON
func FromJSON(data []byte) (*Task, error) {
	var task Task
	err := json.Unmarshal(data, &task)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// CanRetry determines if a task can be retried
func (t *Task) CanRetry() bool {
	return t.RetryCount < t.MaxRetries
}

// MarkStarted marks a task as started
func (t *Task) MarkStarted(workerID string) {
	now := time.Now()
	t.Status = StatusProcessing
	t.StartedAt = &now
	t.WorkerID = workerID
}

// MarkCompleted marks a task as completed
func (t *Task) MarkCompleted() {
	now := time.Now()
	t.Status = StatusCompleted
	t.CompletedAt = &now
}

// MarkFailed marks a task as failed
func (t *Task) MarkFailed(err error) {
	t.Status = StatusFailed
	t.Error = err.Error()
	now := time.Now()
	t.CompletedAt = &now
}

// MarkRetrying marks a task for retry
func (t *Task) MarkRetrying() {
	t.Status = StatusRetrying
	t.RetryCount++
}

// Result represents the result of task execution
type Result struct {
	TaskID    string                 `json:"task_id"`
	Success   bool                   `json:"success"`
	Output    map[string]interface{} `json:"output,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Duration  time.Duration          `json:"duration"`
	Timestamp time.Time              `json:"timestamp"`
}
