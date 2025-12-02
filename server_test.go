package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/distributed-task-queue/internal/queue"
	"github.com/yourusername/distributed-task-queue/internal/storage"
	"github.com/yourusername/distributed-task-queue/internal/task"
	"go.uber.org/zap"
)

func setupTestServer(t *testing.T) (*Server, *queue.Queue) {
	logger, _ := zap.NewDevelopment()
	store := storage.NewMemoryStorage()
	
	q := queue.NewQueue(queue.Config{
		Storage: store,
		Logger:  logger,
	})

	server := NewServer(q, logger)
	return server, q
}

func TestAPI_SubmitTask(t *testing.T) {
	server, _ := setupTestServer(t)

	reqBody := map[string]interface{}{
		"type":     "test_task",
		"priority": 2,
		"payload": map[string]interface{}{
			"key": "value",
		},
		"max_retries": 3,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.NotEmpty(t, response["task_id"])
	assert.Equal(t, "submitted", response["status"])
}

func TestAPI_SubmitTask_InvalidRequest(t *testing.T) {
	server, _ := setupTestServer(t)

	tests := []struct {
		name     string
		reqBody  map[string]interface{}
		wantCode int
	}{
		{
			name: "missing task type",
			reqBody: map[string]interface{}{
				"priority": 2,
				"payload":  map[string]interface{}{},
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "invalid JSON",
			reqBody:  nil,
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			if tt.reqBody != nil {
				body, _ = json.Marshal(tt.reqBody)
			} else {
				body = []byte("invalid json")
			}

			req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)

			assert.Equal(t, tt.wantCode, w.Code)
		})
	}
}

func TestAPI_GetTask(t *testing.T) {
	server, q := setupTestServer(t)

	// Submit a task first
	ctx := context.Background()
	testTask := task.NewTask("test_task", task.PriorityHigh, map[string]interface{}{
		"key": "value",
	})
	err := q.Submit(ctx, testTask)
	require.NoError(t, err)

	// Get the task
	req := httptest.NewRequest("GET", "/api/v1/tasks/"+testTask.ID, nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response task.Task
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, testTask.ID, response.ID)
	assert.Equal(t, testTask.Type, response.Type)
	assert.Equal(t, testTask.Priority, response.Priority)
}

func TestAPI_GetTask_NotFound(t *testing.T) {
	server, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/v1/tasks/nonexistent-id", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPI_GetStats(t *testing.T) {
	server, q := setupTestServer(t)

	// Submit some tasks
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		testTask := task.NewTask("test_task", task.PriorityMedium, nil)
		q.Submit(ctx, testTask)
	}

	req := httptest.NewRequest("GET", "/api/v1/stats", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var stats map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&stats)
	require.NoError(t, err)

	assert.Contains(t, stats, "pending")
}

func TestAPI_Health(t *testing.T) {
	server, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "healthy", response["status"])
}

func TestAPI_Metrics(t *testing.T) {
	server, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "tasks_submitted_total")
}
