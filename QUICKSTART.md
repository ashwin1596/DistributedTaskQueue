# Quick Start Guide

Get the Distributed Task Queue running in 5 minutes!

## Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose
- Make (optional, but recommended)

## Option 1: Docker Compose (Easiest)

### 1. Start all services

```bash
# Clone the repository
git clone https://github.com/yourusername/distributed-task-queue.git
cd distributed-task-queue

# Start everything (Redis, Server, 3 Workers, Prometheus)
docker-compose up -d

# View logs
docker-compose logs -f
```

### 2. Verify it's running

```bash
# Check health
curl http://localhost:8080/health

# Should return: {"status":"healthy"}
```

### 3. Submit your first task

```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "type": "send_email",
    "priority": 2,
    "payload": {
      "recipient": "user@example.com",
      "subject": "Hello from Task Queue!",
      "body": "This is your first task"
    }
  }'
```

You'll get a response with a `task_id`:
```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "submitted"
}
```

### 4. Check task status

```bash
curl http://localhost:8080/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000
```

### 5. View metrics

Open http://localhost:8080/metrics in your browser to see Prometheus metrics.

Open http://localhost:9090 to access Prometheus dashboard.

## Option 2: Local Development

### 1. Install dependencies

```bash
go mod download
```

### 2. Start Redis

```bash
docker run -d --name redis -p 6379:6379 redis:7-alpine
```

### 3. Build the project

```bash
make build
```

### 4. Run the server

```bash
# In terminal 1
make run-server
```

### 5. Run a worker

```bash
# In terminal 2
make run-worker
```

### 6. Submit tasks

Use the same curl commands as above!

## Testing the System

### Run the example client script

```bash
chmod +x examples/client.sh
./examples/client.sh
```

This script will:
- Submit multiple tasks with different priorities
- Show task status
- Display queue statistics
- Check system health

## What's Running?

- **Server** (port 8080): API for task submission and management
- **Workers** (3 instances): Process tasks from the queue
- **Redis** (port 6379): Stores task data
- **Prometheus** (port 9090): Metrics collection and visualization

## Common Commands

```bash
# View all running services
docker-compose ps

# View logs for specific service
docker-compose logs -f server
docker-compose logs -f worker-1

# Stop all services
docker-compose down

# Restart services
docker-compose restart

# Scale workers
docker-compose up -d --scale worker=5

# Run tests
make test

# View test coverage
make test-coverage
```

## Example Tasks

### High Priority Email
```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "type": "send_email",
    "priority": 2,
    "payload": {
      "recipient": "urgent@example.com",
      "subject": "Urgent Alert"
    }
  }'
```

### Image Processing
```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "type": "process_image",
    "priority": 1,
    "payload": {
      "image_url": "https://example.com/photo.jpg",
      "operation": "resize",
      "width": 800,
      "height": 600
    }
  }'
```

### Batch Data Export
```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "type": "export_data",
    "priority": 0,
    "payload": {
      "format": "csv",
      "date_range": {
        "start": "2024-01-01",
        "end": "2024-01-31"
      }
    }
  }'
```

### Critical Webhook
```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "type": "call_webhook",
    "priority": 3,
    "payload": {
      "url": "https://example.com/webhook",
      "method": "POST"
    }
  }'
```

## Monitoring

### View Queue Statistics
```bash
curl http://localhost:8080/api/v1/stats
```

### View Metrics
```bash
# Raw Prometheus metrics
curl http://localhost:8080/metrics

# Or open in browser
open http://localhost:8080/metrics
```

### Prometheus Queries

Access Prometheus at http://localhost:9090 and try these queries:

```promql
# Task submission rate
rate(tasks_submitted_total[5m])

# Task completion rate
rate(tasks_processed_total{status="completed"}[5m])

# Average task duration
rate(task_duration_seconds_sum[5m]) / rate(task_duration_seconds_count[5m])

# Current queue size
queue_size

# Active workers
workers_active

# Failed task rate
rate(tasks_processed_total{status="failed"}[5m])
```

## Troubleshooting

### Redis connection issues
```bash
# Check if Redis is running
docker ps | grep redis

# Test Redis connection
redis-cli ping
```

### Port already in use
```bash
# Find process using port 8080
lsof -i :8080

# Kill the process
kill -9 <PID>
```

### Workers not processing tasks
```bash
# Check worker logs
docker-compose logs worker-1

# Restart workers
docker-compose restart worker-1 worker-2 worker-3
```

### Clean start
```bash
# Stop and remove everything
docker-compose down -v

# Start fresh
docker-compose up -d
```

## Next Steps

1. **Customize Task Handlers**: Edit `cmd/server/main.go` to add your own task types
2. **Configure Scaling**: Adjust worker count in `docker-compose.yml`
3. **Add Authentication**: Implement API key validation in `api/server.go`
4. **Set Up Monitoring**: Connect Prometheus to Grafana for dashboards
5. **Deploy to Production**: See deployment guide in docs/

## Need Help?

- Check the [README](README.md) for detailed documentation
- Review [CONTRIBUTING](CONTRIBUTING.md) for development guidelines
- Open an issue on GitHub for bugs or feature requests

Happy task queuing!
