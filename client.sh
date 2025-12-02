#!/bin/bash

# Example client script to interact with the task queue API
# Usage: ./examples/client.sh

API_URL="${API_URL:-http://localhost:8080}"

echo "🚀 Distributed Task Queue - Example Client"
echo "=========================================="
echo ""

# Function to submit a task
submit_task() {
    local task_type=$1
    local priority=$2
    local payload=$3
    
    echo "📤 Submitting $task_type task (priority: $priority)..."
    
    response=$(curl -s -X POST "$API_URL/api/v1/tasks" \
        -H "Content-Type: application/json" \
        -d "{
            \"type\": \"$task_type\",
            \"priority\": $priority,
            \"payload\": $payload,
            \"max_retries\": 3
        }")
    
    task_id=$(echo "$response" | grep -o '"task_id":"[^"]*' | cut -d'"' -f4)
    echo "✅ Task submitted: $task_id"
    echo ""
    echo "$task_id"
}

# Function to get task status
get_task() {
    local task_id=$1
    
    echo "🔍 Checking task status: $task_id"
    
    curl -s "$API_URL/api/v1/tasks/$task_id" | jq '.'
    echo ""
}

# Function to get stats
get_stats() {
    echo "📊 Queue Statistics:"
    curl -s "$API_URL/api/v1/stats" | jq '.'
    echo ""
}

# Function to check health
check_health() {
    echo "❤️  Health Check:"
    curl -s "$API_URL/health" | jq '.'
    echo ""
}

# Main execution
echo "1️⃣  Submitting Email Task (High Priority)"
email_task=$(submit_task "send_email" 2 '{
    "recipient": "user@example.com",
    "subject": "Test Email",
    "body": "This is a test email from the task queue"
}')

echo "2️⃣  Submitting Image Processing Task (Medium Priority)"
image_task=$(submit_task "process_image" 1 '{
    "image_url": "https://example.com/image.jpg",
    "operation": "resize",
    "width": 800,
    "height": 600
}')

echo "3️⃣  Submitting Data Export Task (Low Priority)"
export_task=$(submit_task "export_data" 0 '{
    "format": "csv",
    "date_range": {
        "start": "2024-01-01",
        "end": "2024-01-31"
    }
}')

echo "4️⃣  Submitting Webhook Task (Critical Priority)"
webhook_task=$(submit_task "call_webhook" 3 '{
    "url": "https://example.com/webhook",
    "method": "POST",
    "headers": {
        "Authorization": "Bearer token123"
    },
    "body": {
        "event": "task.completed",
        "timestamp": "2024-01-15T10:30:00Z"
    }
}')

sleep 2

echo "=========================================="
get_stats
check_health

echo "=========================================="
echo "📋 Task Details"
echo "=========================================="
echo ""

get_task "$email_task"
get_task "$image_task"

echo "=========================================="
echo "✨ Demo Complete!"
echo ""
echo "To view metrics: $API_URL/metrics"
echo "To view Prometheus: http://localhost:9090"
echo ""
