# System Architecture

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           Client Applications                            │
│                                                                           │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐                │
│  │   Web    │  │  Mobile  │  │   CLI    │  │  Service │                │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘                │
│       │             │              │             │                       │
└───────┼─────────────┼──────────────┼─────────────┼───────────────────────┘
        │             │              │             │
        └─────────────┴──────────────┴─────────────┘
                           │
                           ▼
                  ┌─────────────────┐
                  │   Load Balancer │  (Optional)
                  └────────┬────────┘
                           │
        ┏━━━━━━━━━━━━━━━━━┻━━━━━━━━━━━━━━━━━┓
        ▼                                    ▼
┌───────────────┐                    ┌───────────────┐
│  API Server 1 │                    │  API Server N │
│               │                    │               │
│  Port: 8080   │                    │  Port: 8080   │
│               │                    │               │
│  ┌─────────┐  │                    │  ┌─────────┐  │
│  │   API   │  │                    │  │   API   │  │
│  │ Handlers│  │                    │  │ Handlers│  │
│  └────┬────┘  │                    │  └────┬────┘  │
│       │       │                    │       │       │
│  ┌────▼────┐  │                    │  ┌────▼────┐  │
│  │ Metrics │  │                    │  │ Metrics │  │
│  └─────────┘  │                    │  └─────────┘  │
└───────┬───────┘                    └───────┬───────┘
        │                                    │
        └────────────────┬───────────────────┘
                         │
                         ▼
              ┌─────────────────────┐
              │       Redis         │
              │   (Task Storage)    │
              │                     │
              │  ┌───────────────┐  │
              │  │  Task Queue   │  │
              │  │  - Priority   │  │
              │  │  - Status     │  │
              │  │  - Metadata   │  │
              │  └───────────────┘  │
              └──────────┬──────────┘
                         │
        ┏━━━━━━━━━━━━━━━━┻━━━━━━━━━━━━━━━━┓
        ▼                ▼                 ▼
┌───────────────┐ ┌───────────────┐ ┌───────────────┐
│   Worker 1    │ │   Worker 2    │ │   Worker N    │
│               │ │               │ │               │
│  ┌─────────┐  │ │  ┌─────────┐  │ │  ┌─────────┐  │
│  │ Poller  │  │ │  │ Poller  │  │ │  │ Poller  │  │
│  └────┬────┘  │ │  └────┬────┘  │ │  └────┬────┘  │
│       │       │ │       │       │ │       │       │
│  ┌────▼────┐  │ │  ┌────▼────┐  │ │  ┌────▼────┐  │
│  │Critical │  │ │  │Critical │  │ │  │Critical │  │
│  │  Queue  │  │ │  │  Queue  │  │ │  │  Queue  │  │
│  └────┬────┘  │ │  └────┬────┘  │ │  └────┬────┘  │
│       │       │ │       │       │ │       │       │
│  ┌────▼────┐  │ │  ┌────▼────┐  │ │  ┌────▼────┐  │
│  │  High   │  │ │  │  High   │  │ │  │  High   │  │
│  │  Queue  │  │ │  │  Queue  │  │ │  │  Queue  │  │
│  └────┬────┘  │ │  └────┬────┘  │ │  └────┬────┘  │
│       │       │ │       │       │ │       │       │
│  ┌────▼────┐  │ │  ┌────▼────┐  │ │  ┌────▼────┐  │
│  │ Medium  │  │ │  │ Medium  │  │ │  │ Medium  │  │
│  │  Queue  │  │ │  │  Queue  │  │ │  │  Queue  │  │
│  └────┬────┘  │ │  └────┬────┘  │ │  └────┬────┘  │
│       │       │ │       │       │ │       │       │
│  ┌────▼────┐  │ │  ┌────▼────┐  │ │  ┌────▼────┐  │
│  │   Low   │  │ │  │   Low   │  │ │  │   Low   │  │
│  │  Queue  │  │ │  │  Queue  │  │ │  │  Queue  │  │
│  └────┬────┘  │ │  └────┬────┘  │ │  └────┬────┘  │
│       │       │ │       │       │ │       │       │
│  ┌────▼─────┐ │ │  ┌────▼─────┐ │ │  ┌────▼─────┐ │
│  │ Handlers │ │ │  │ Handlers │ │ │  │ Handlers │ │
│  └──────────┘ │ │  └──────────┘ │ │  └──────────┘ │
└───────────────┘ └───────────────┘ └───────────────┘
        │                │                 │
        └────────────────┴─────────────────┘
                         │
                         ▼
                ┌─────────────────┐
                │   Prometheus    │
                │    (Metrics)    │
                └─────────────────┘
```

## Component Flow

### Task Submission Flow
```
1. Client → API Server
2. API Server → Validate Request
3. API Server → Create Task
4. API Server → Save to Redis
5. API Server → Push to Priority Channel
6. API Server → Return task_id to Client
```

### Task Processing Flow
```
1. Worker → Poll Redis (or receive from channel)
2. Worker → Fetch Task
3. Worker → Mark as "Processing"
4. Worker → Execute Handler
5a. Success → Mark as "Completed"
5b. Failure → Check Retry Count
    → If retries available → Mark as "Retrying" → Requeue
    → If max retries → Mark as "Failed"
6. Worker → Update Metrics
7. Worker → Poll for next task
```

## Data Model

### Task Structure
```
Task {
    id: UUID
    type: string
    priority: 0-3 (Low, Medium, High, Critical)
    status: pending | processing | completed | failed | retrying
    payload: map[string]interface{}
    max_retries: int
    retry_count: int
    created_at: timestamp
    started_at: timestamp?
    completed_at: timestamp?
    error: string?
    worker_id: string?
}
```

### Redis Keys
```
task:<task_id>                    # Task data (JSON)
tasks:status:<status>             # Sorted set by priority+timestamp
```

## Concurrency Model

### Per Worker
```
Main Goroutine
    │
    ├─── Poller Goroutine (polls Redis every 1s)
    │
    ├─── Critical Priority Workers (N goroutines)
    │      └─── Process tasks from critical channel
    │
    ├─── High Priority Workers (N goroutines)
    │      └─── Process tasks from high channel
    │
    ├─── Medium Priority Workers (N goroutines)
    │      └─── Process tasks from medium channel
    │
    └─── Low Priority Workers (N goroutines)
           └─── Process tasks from low channel
```

## Scaling Strategy

### Horizontal Scaling
```
Single Machine:
├─── 1 Server
└─── 3 Workers

Scaled:
├─── 3 Servers (behind load balancer)
└─── 10 Workers
     ├─── Worker 1-3 (Machine A)
     ├─── Worker 4-6 (Machine B)
     └─── Worker 7-10 (Machine C)
```

### Vertical Scaling
```
Increase per-worker concurrency:
- More goroutines per priority level
- Larger channel buffers
- Redis connection pool size
```

## Monitoring Points

```
┌─────────────┐
│   Metrics   │
├─────────────┤
│ • tasks_submitted_total
│ • tasks_processed_total
│ • task_duration_seconds
│ • queue_size
│ • workers_active
│ • task_retries_total
└─────────────┘
       │
       ▼
┌─────────────┐
│ Prometheus  │
└─────────────┘
       │
       ▼
┌─────────────┐
│   Grafana   │ (Optional)
│ (Dashboard) │
└─────────────┘
```

## Error Handling

### Retry Strategy
```
Attempt 1: Immediate
Attempt 2: 1 second delay
Attempt 3: 4 seconds delay
Attempt 4: 9 seconds delay
Failed: Mark as permanently failed
```

### Failure Recovery
```
Worker Crash:
├─── Tasks in "processing" status remain in Redis
├─── Timeout mechanism can requeue stale tasks
└─── New worker picks up from queue

Redis Crash:
├─── Workers continue with in-memory channel
├─── Graceful degradation
└─── Resume from Redis when available
```

## Security Considerations

### Production Additions Needed
```
1. Authentication
   ├─── API Key validation
   ├─── JWT tokens
   └─── mTLS for service-to-service

2. Authorization
   ├─── Role-based access control
   └─── Task ownership validation

3. Network Security
   ├─── TLS for Redis connections
   ├─── HTTPS for API
   └─── Network policies (K8s)

4. Input Validation
   ├─── Payload size limits
   ├─── Schema validation
   └─── Rate limiting per client
```

## Deployment Architecture

### Docker Compose (Development)
```
Services:
├─── redis
├─── server
├─── worker-1
├─── worker-2
├─── worker-3
└─── prometheus
```

### Kubernetes (Production)
```
Deployments:
├─── redis-deployment (StatefulSet)
├─── server-deployment (3 replicas)
├─── worker-deployment (10 replicas)
└─── prometheus-deployment

Services:
├─── redis-service (ClusterIP)
├─── server-service (LoadBalancer)
└─── prometheus-service (ClusterIP)

Ingress:
└─── task-queue-ingress (HTTPS)
```

## Performance Characteristics

### Throughput
- Single worker: ~250 tasks/second
- 4 workers: ~1000 tasks/second
- Linear scaling with worker count

### Latency
- Task submission: <10ms (p99)
- Task pickup: <100ms (p99)
- Task execution: varies by handler

### Resource Usage
- Server: ~50MB RAM, minimal CPU
- Worker: ~100MB RAM, CPU varies by workload
- Redis: ~100MB RAM baseline + task data
