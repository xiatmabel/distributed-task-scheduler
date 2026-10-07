package task

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var ErrNotFound = errors.New("task not found")

type Task struct {
	ID          int64           `json:"id"`
	Type        string          `json:"type"`
	Payload     json.RawMessage `json:"payload"`
	Status      string          `json:"status"`
	Attempts    int             `json:"attempts"`
	ScheduledAt time.Time       `json:"scheduled_at"`
	StartedAt   *time.Time      `json:"started_at,omitempty"`
	FinishedAt  *time.Time      `json:"finished_at,omitempty"`
	WorkerID    *string         `json:"worker_id,omitempty"`
	Error       *string         `json:"error,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

type CreateInput struct {
	Type        string          `json:"type"`
	Payload     json.RawMessage `json:"payload"`
	ScheduledAt *time.Time      `json:"scheduled_at,omitempty"`
}

type Store interface {
	Create(context.Context, CreateInput) (Task, error)
	Get(context.Context, int64) (Task, error)
}
