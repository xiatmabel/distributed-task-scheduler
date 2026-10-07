# Distributed Task Scheduler

[English](README.md) | [简体中文](README.zh-CN.md)

A single-node task scheduler MVP for learning task state machines, workers, database-backed queues, and concurrent task claiming through a practical project.

## Features

- Create and query tasks through an HTTP API
- Persist task state in PostgreSQL
- Poll and execute `print` tasks with a worker
- Atomically claim tasks with `FOR UPDATE SKIP LOCKED`
- Start the complete stack with Docker Compose

## Quick start

Docker Desktop is required.

```powershell
docker compose up --build
```

Open <http://localhost:8080> to view service information and available endpoints.

Create a task:

```powershell
$task = Invoke-RestMethod `
  -Method Post `
  -Uri http://localhost:8080/tasks `
  -ContentType application/json `
  -Body '{"type":"print","payload":{"message":"Hello, distributed systems!"}}'

$task
```

Query its execution result:

```powershell
Invoke-RestMethod "http://localhost:8080/tasks/$($task.id)"
```

The task status should eventually become `succeeded`, and the worker logs will contain the message.

Stop the stack and delete its local data:

```powershell
docker compose down --volumes
```

## API

### `POST /tasks`

```json
{
  "type": "print",
  "payload": {
    "message": "Hello"
  }
}
```

Use the optional `scheduled_at` field to schedule a task for the future:

```json
{
  "type": "print",
  "payload": {
    "message": "Run later"
  },
  "scheduled_at": "2026-10-08T10:00:00Z"
}
```

### `GET /tasks/{id}`

Returns the task, its status, attempt count, worker, and timestamps.

### `GET /healthz`

Returns the health status of the API process.

## Project structure

```text
cmd/api/             HTTP API entry point
cmd/worker/          Worker entry point
internal/httpapi/    HTTP routes and handlers
internal/task/       Task model and PostgreSQL storage
migrations/          Database initialization scripts
compose.yaml         Local runtime environment
```

## Learning focus

This version uses a database transaction to prevent multiple workers from claiming the same task concurrently. It does not yet recover tasks when a worker crashes during execution. The next stage will introduce leases, timeout recovery, retries, and idempotency.

## Tests

Run the tests through Docker without installing Go:

```powershell
docker run --rm -v "${PWD}:/src" -w /src golang:1.24-alpine go test ./...
```
