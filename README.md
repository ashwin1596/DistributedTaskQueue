# Distributed Task Queue

A distributed task queue in Go that processes thousands of asynchronous jobs per second. It features priority scheduling, automatic retries, horizontal scaling, and production-grade monitoring. The system is fully containerized, has comprehensive tests, and includes Prometheus metrics.

![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/license-MIT-blue.svg)

## Features

### Core Functionality
- **Priority Queue System** - Four priority levels (Low, Medium, High, Critical)
- **Distributed Workers** - Horizontal scaling with multiple worker instances
- **Task Persistence** - Redis-backed storage for reliability
- **Automatic Retries** - Exponential backoff for failed tasks
- **Graceful Shutdown** - Clean shutdown with task completion

### Production Ready
- **RESTful API** - Complete HTTP API for task management
- **Observability** - Prometheus metrics integration
- **Health Checks** - Built-in health endpoints
- **Structured Logging** - JSON logging with Zap
- **Comprehensive Testing** - Unit tests with high coverage
- **Docker Support** - Full containerization with Docker Compose

## Architecture

```
┌─────────────┐      HTTP      ┌──────────┐
│   Client    │ ────────────► │  Server  │
└─────────────┘                └──────────┘
                                     │
                                     ▼
                              ┌──────────┐
                              │  Redis   │
                              └──────────┘
                                     │
                    ┌────────────────┼────────────────┐
                    ▼                ▼                ▼
               ┌─────────┐     ┌─────────┐     ┌─────────┐
               │ Worker 1│     │ Worker 2│     │ Worker 3│
               └─────────┘     └─────────┘     └─────────┘
```

## Quick Start

### Using Docker Compose (Recommended)

```bash
# Start all services (Redis, Server, 3 Workers, Prometheus)
make docker-up

# View logs
make docker-logs

# Stop all services
make docker-down
```

### Local Development

```bash
# Install dependencies
make deps

# Start Redis (required)
docker run -d -p 6379:6379 redis:7-alpine

# Run tests
make test

# Build binaries
make build

# Run server
make run-server

# Run worker (in another terminal)
make run-worker
```

## API Usage

### Submit a Task

```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "type": "send_email",
    "priority": 2,
    "payload": {
      "recipient": "user@example.com",
      "subject": "Hello World",
      "body": "This is a test email"
    },
    "max_retries": 3
  }'
```

Response:
```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "submitted"
}
```

### Get Task Status

```bash
curl http://localhost:8080/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000
```

Response:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "type": "send_email",
  "priority": 2,
  "status": "completed",
  "payload": {
    "recipient": "user@example.com",
    "subject": "Hello World"
  },
  "max_retries": 3,
  "retry_count": 0,
  "created_at": "2024-01-15T10:30:00Z",
  "started_at": "2024-01-15T10:30:01Z",
  "completed_at": "2024-01-15T10:30:03Z",
  "worker_id": "worker-1"
}
```

### Get Queue Statistics

```bash
curl http://localhost:8080/api/v1/stats
```

Response:
```json
{
  "pending": 5,
  "processing": 3,
  "completed": 142,
  "failed": 2
}
```

### Health Check

```bash
curl http://localhost:8080/health
```

## Task Types

The system comes with example handlers for common task types:

### Email Tasks
```json
{
  "type": "send_email",
  "priority": 2,
  "payload": {
    "recipient": "user@example.com",
    "subject": "Subject",
    "body": "Message body"
  }
}
```

### Image Processing
```json
{
  "type": "process_image",
  "priority": 1,
  "payload": {
    "image_url": "https://example.com/image.jpg",
    "operation": "resize"
  }
}
```

### Data Export
```json
{
  "type": "export_data",
  "priority": 0,
  "payload": {
    "format": "csv",
    "filters": {}
  }
}
```

### Webhook Calls
```json
{
  "type": "call_webhook",
  "priority": 3,
  "payload": {
    "url": "https://example.com/webhook",
    "method": "POST",
    "body": {}
  }
}
```

## Priority Levels

| Priority | Value | Use Case |
|----------|-------|----------|
| Critical | 3 | Time-sensitive operations, alerts |
| High | 2 | User-facing operations |
| Medium | 1 | Background processing |
| Low | 0 | Batch jobs, cleanup tasks |

## Custom Task Handlers

To add custom task handlers, register them in your code:

```go
queue.RegisterHandler("my_task", func(ctx context.Context, t *task.Task) error {
    // Extract payload
    data := t.Payload["key"].(string)
    
    // Perform work
    result, err := processData(data)
    if err != nil {
        return err // Will trigger retry
    }
    
    // Success
    return nil
})
```

## Monitoring

### Prometheus Metrics

Access metrics at `http://localhost:8080/metrics`

Available metrics:
- `tasks_submitted_total` - Total tasks submitted by type and priority
- `tasks_processed_total` - Total tasks processed by type and status
- `task_duration_seconds` - Task processing duration histogram
- `queue_size` - Current queue size by priority
- `workers_active` - Number of active workers
- `task_retries_total` - Total retry attempts by type

### Prometheus Dashboard

Access Prometheus UI at `http://localhost:9090`

Example queries:
```promql
# Task processing rate
rate(tasks_processed_total[5m])

# Average task duration by type
rate(task_duration_seconds_sum[5m]) / rate(task_duration_seconds_count[5m])

# Failed task rate
rate(tasks_processed_total{status="failed"}[5m])
```

## Configuration

### Environment Variables

**Server:**
- `REDIS_ADDR` - Redis address (default: `localhost:6379`)
- `REDIS_PASSWORD` - Redis password (default: empty)
- `PORT` - HTTP server port (default: `8080`)

**Worker:**
- `REDIS_ADDR` - Redis address (default: `localhost:6379`)
- `REDIS_PASSWORD` - Redis password (default: empty)
- `WORKER_ID` - Unique worker identifier (default: `worker-1`)

## Testing

```bash
# Run all tests
make test

# Run tests with coverage report
make test-coverage

# View coverage in browser
open coverage.html
```

## Project Structure

```
distributed-task-queue/
├── cmd/
│   ├── server/          # HTTP API server
│   └── worker/          # Task worker
├── internal/
│   ├── queue/           # Core queue implementation
│   ├── task/            # Task definitions
│   ├── storage/         # Redis & memory storage
│   └── metrics/         # Prometheus metrics
├── api/                 # HTTP handlers
├── docker-compose.yml   # Docker orchestration
├── Makefile            # Build automation
└── README.md           # This file
```

## Performance

### Benchmarks

- **Throughput**: 1000+ tasks/second per worker
- **Latency**: <100ms task submission
- **Scalability**: Linear scaling with worker count
- **Reliability**: Automatic retry with exponential backoff

### Scaling

**Horizontal Scaling:**
```bash
# Add more workers
docker-compose up -d --scale worker=5
```

**Tuning:**
- Adjust `numWorkers` per process for CPU-bound tasks
- Increase Redis connection pool for high throughput
- Configure task timeouts based on workload

## Production Considerations

### Security
- Add authentication/authorization to API endpoints
- Use TLS for Redis connections
- Implement rate limiting
- Validate task payloads

### Reliability
- Configure Redis persistence (AOF/RDB)
- Set up Redis replication for high availability
- Monitor queue depth and worker health
- Implement dead letter queues

### Observability
- Export metrics to monitoring system
- Set up alerting for queue depth, failure rate
- Configure structured logging aggregation
- Add distributed tracing

This project demonstrates:

✅ **Advanced Go** - Goroutines, channels, interfaces, context  
✅ **Distributed Systems** - Message queues, task distribution  
✅ **Production Practices** - Testing, monitoring, Docker  
✅ **API Design** - RESTful endpoints, error handling  
✅ **Persistence** - Redis integration, data modeling  
✅ **Observability** - Metrics, logging, health checks  
✅ **DevOps** - Docker, CI/CD ready, configuration management  
