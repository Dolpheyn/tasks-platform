# Tasks Platform

A lightweight HTTP API for task queue management,
backed by [asynq](https://github.com/hibiken/asynq) and Redis.

## Overview

This project provides a simple REST API layer on top of asynq,
allowing any language to enqueue tasks and consume work without
requiring asynq client libraries. The service handles task queuing,
scheduling, and worker management through HTTP endpoints.

**Scalability**: Built for horizontal scaling with multiple instances.
asynq uses Redis atomic commands (BLPOP, ZADD, etc.) that are cluster-safe,
ensuring reliable task distribution across multiple workers and producers
without conflicts. Tasks are processed exactly once, even with concurrent
workers polling the same queues.

(context: it is a part of a take-home assignment)

## Architecture

```
┌─────────────────┐    POST /tasks      ┌─────────────────┐
│   Producer      │ ──────────────────► │   HTTP API      │
│   (any lang)    │                     │   (Gin)         │
└─────────────────┘                     └─────────────────┘
                                                │
                                                │ asynq.Client.Enqueue
                                                ▼
                                        ┌─────────────────┐
                                        │     Redis       │
                                        │   (asynq)       │
                                        └─────────────────┘
                                                ▲
                                                │ asynq.Server.Run
                                                ▼
                                        ┌─────────────────┐
                                        │  TaskManager    │
                                        │  (ProcessTask)  │
                                        └─────────────────┘
                                                ▲
                                                │ GET /poll
┌─────────────────┐                             │
│    Worker       │ ◄───────────────────────────┘
│   (any lang)    │
└─────────────────┘
```

**Components:**
- **HTTP Layer**: Gin web server with REST endpoints for task enqueueing and worker polling
- **TaskManager**: Bridge between asynq's pull-based queue and HTTP long-poll model for external workers
- **Queue Engine**: asynq handles persistence, retries, scheduling, and concurrent task distribution
- **Storage**: Redis for task queue and metadata

**Flow:**
1. Producer enqueues task via `POST /tasks` → asynq.Client pushes to Redis
2. asynq.Server pulls job → hands to TaskManager.ProcessTask (blocks until pickup)
3. Worker polls via `GET /poll` → TaskManager.TryConsumeTask signals pickup
4. TaskManager manages task lifecycle with heartbeats and completion signals

## Quick Start

```bash
# Using Docker Compose
docker compose up --build

# Test the API
curl -X POST localhost:8080/tasks \
     -H 'Content-Type: application/json' \
     -d '{"job_type":"demo","payload":{"hello":"world"}}'

curl "localhost:8080/poll?worker_id=myworker&task_type=demo"
```

## API Endpoints

- `POST /tasks` - Enqueue immediate task
- `GET /poll` - Long-poll for tasks (worker endpoint)
- `GET /health` - Health check

## Development

```bash
# Run locally (requires Redis)
go run ./cmd

# Test
go test ./...

# Build
go build ./cmd/...
```

## Configuration

Environment variables:
- `PORT` - HTTP server port (default: 8080)
- `REDIS_ADDR` - Redis address (default: localhost:6379)
- `REDIS_PASSWORD` - Redis password
- `REDIS_DB` - Redis database number (default: 0)

## Project Structure

```
cmd/        # Application entry point
internal/   # Private application code
  api/      # HTTP handlers and server
  config/   # Configuration management
pkg/        # Public packages
  dto/      # Data transfer objects
```

Built with Go, Gin, and asynq.
