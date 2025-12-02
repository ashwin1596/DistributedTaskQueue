.PHONY: build test run clean docker-build docker-up docker-down help

# Variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
DOCKER=docker-compose

# Build targets
build: build-server build-worker

build-server:
	@echo "Building server..."
	$(GOBUILD) -o bin/server ./cmd/server

build-worker:
	@echo "Building worker..."
	$(GOBUILD) -o bin/worker ./cmd/worker

# Test targets
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run targets
run-server: build-server
	@echo "Starting server..."
	./bin/server

run-worker: build-worker
	@echo "Starting worker..."
	./bin/worker

# Docker targets
docker-build:
	@echo "Building Docker images..."
	$(DOCKER) build

docker-up:
	@echo "Starting services..."
	$(DOCKER) up -d

docker-down:
	@echo "Stopping services..."
	$(DOCKER) down

docker-logs:
	$(DOCKER) logs -f

docker-restart: docker-down docker-up

# Development
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

lint:
	@echo "Running linter..."
	golangci-lint run ./...

# Cleanup
clean:
	@echo "Cleaning up..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	$(DOCKER) down -v

# Help
help:
	@echo "Available targets:"
	@echo "  build           - Build server and worker binaries"
	@echo "  build-server    - Build server binary"
	@echo "  build-worker    - Build worker binary"
	@echo "  test            - Run tests"
	@echo "  test-coverage   - Run tests with coverage report"
	@echo "  run-server      - Run the server"
	@echo "  run-worker      - Run a worker"
	@echo "  docker-build    - Build Docker images"
	@echo "  docker-up       - Start all services with Docker Compose"
	@echo "  docker-down     - Stop all services"
	@echo "  docker-logs     - View logs from all services"
	@echo "  docker-restart  - Restart all services"
	@echo "  deps            - Download and tidy dependencies"
	@echo "  fmt             - Format code"
	@echo "  lint            - Run linter"
	@echo "  clean           - Clean build artifacts and Docker volumes"
	@echo "  help            - Show this help message"
