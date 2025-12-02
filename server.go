package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/yourusername/distributed-task-queue/internal/queue"
	"github.com/yourusername/distributed-task-queue/internal/task"
	"go.uber.org/zap"
)

// Server represents the HTTP API server
type Server struct {
	queue  *queue.Queue
	logger *zap.Logger
	router *chi.Mux
}

// NewServer creates a new API server
func NewServer(q *queue.Queue, logger *zap.Logger) *Server {
	s := &Server{
		queue:  q,
		logger: logger,
		router: chi.NewRouter(),
	}

	s.setupRoutes()
	return s
}

// setupRoutes configures the API routes
func (s *Server) setupRoutes() {
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.Timeout(60))

	// API routes
	s.router.Route("/api/v1", func(r chi.Router) {
		r.Post("/tasks", s.handleSubmitTask)
		r.Get("/tasks/{id}", s.handleGetTask)
		r.Get("/tasks", s.handleListTasks)
		r.Get("/stats", s.handleGetStats)
	})

	// Health check
	s.router.Get("/health", s.handleHealth)

	// Metrics endpoint
	s.router.Handle("/metrics", promhttp.Handler())
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// handleSubmitTask handles task submission
func (s *Server) handleSubmitTask(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type       string                 `json:"type"`
		Priority   int                    `json:"priority"`
		Payload    map[string]interface{} `json:"payload"`
		MaxRetries int                    `json:"max_retries,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Type == "" {
		s.respondError(w, http.StatusBadRequest, "task type is required")
		return
	}

	priority := task.Priority(req.Priority)
	if priority < task.PriorityLow || priority > task.PriorityCritical {
		priority = task.PriorityMedium
	}

	t := task.NewTask(req.Type, priority, req.Payload)
	if req.MaxRetries > 0 {
		t.MaxRetries = req.MaxRetries
	}

	if err := s.queue.Submit(r.Context(), t); err != nil {
		s.logger.Error("failed to submit task", zap.Error(err))
		s.respondError(w, http.StatusInternalServerError, "failed to submit task")
		return
	}

	s.respondJSON(w, http.StatusCreated, map[string]interface{}{
		"task_id": t.ID,
		"status":  "submitted",
	})
}

// handleGetTask retrieves a task by ID
func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		s.respondError(w, http.StatusBadRequest, "task ID is required")
		return
	}

	t, err := s.queue.GetTask(r.Context(), id)
	if err != nil {
		s.respondError(w, http.StatusNotFound, "task not found")
		return
	}

	s.respondJSON(w, http.StatusOK, t)
}

// handleListTasks lists tasks (placeholder for pagination)
func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	statusParam := r.URL.Query().Get("status")
	limitParam := r.URL.Query().Get("limit")

	limit := 10
	if limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	status := task.StatusPending
	if statusParam != "" {
		status = task.Status(statusParam)
	}

	// This is a simplified implementation
	// In production, you'd want proper pagination
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"tasks":  []task.Task{},
		"total":  0,
		"limit":  limit,
		"status": status,
	})
}

// handleGetStats returns queue statistics
func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.queue.GetStats(r.Context())
	if err != nil {
		s.logger.Error("failed to get stats", zap.Error(err))
		s.respondError(w, http.StatusInternalServerError, "failed to get stats")
		return
	}

	s.respondJSON(w, http.StatusOK, stats)
}

// handleHealth returns health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.respondJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})
}

// respondJSON writes a JSON response
func (s *Server) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError writes an error response
func (s *Server) respondError(w http.ResponseWriter, status int, message string) {
	s.respondJSON(w, status, map[string]string{
		"error": message,
	})
}
