package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// TasksSubmitted tracks total tasks submitted
	TasksSubmitted = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tasks_submitted_total",
			Help: "Total number of tasks submitted",
		},
		[]string{"type", "priority"},
	)

	// TasksProcessed tracks total tasks processed
	TasksProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tasks_processed_total",
			Help: "Total number of tasks processed",
		},
		[]string{"type", "status"},
	)

	// TaskDuration tracks task processing duration
	TaskDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "task_duration_seconds",
			Help:    "Duration of task processing",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"type"},
	)

	// QueueSize tracks current queue sizes
	QueueSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "queue_size",
			Help: "Current number of tasks in queue",
		},
		[]string{"priority"},
	)

	// WorkersActive tracks active workers
	WorkersActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "workers_active",
			Help: "Number of currently active workers",
		},
	)

	// TaskRetries tracks task retry counts
	TaskRetries = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "task_retries_total",
			Help: "Total number of task retries",
		},
		[]string{"type"},
	)
)
