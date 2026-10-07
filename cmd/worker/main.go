package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xiatmabel/distributed-task-scheduler/internal/task"
)

type printPayload struct {
	Message string `json:"message"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	databaseURL := envOrDefault("DATABASE_URL", "postgres://scheduler:scheduler@localhost:5432/scheduler")
	workerID := envOrDefault("WORKER_ID", "worker-1")
	pollInterval, err := time.ParseDuration(envOrDefault("POLL_INTERVAL", "1s"))
	if err != nil || pollInterval <= 0 {
		logger.Error("invalid POLL_INTERVAL", "error", err)
		os.Exit(1)
	}

	db, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		logger.Error("configure database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	if err := db.Ping(context.Background()); err != nil {
		logger.Error("connect to database", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	store := task.NewPostgresStore(db)
	logger.Info("worker started", "worker_id", workerID)

	for {
		claimed, err := store.ClaimNext(ctx, workerID)
		if errors.Is(err, pgx.ErrNoRows) {
			if !wait(ctx, pollInterval) {
				return
			}
			continue
		}
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.Error("claim task", "error", err)
			if !wait(ctx, pollInterval) {
				return
			}
			continue
		}

		logger.Info("task claimed", "task_id", claimed.ID, "type", claimed.Type)
		executionError := execute(claimed)
		if err := store.Complete(ctx, claimed.ID, executionError); err != nil {
			logger.Error("complete task", "task_id", claimed.ID, "error", err)
			continue
		}
		if executionError != nil {
			logger.Error("task failed", "task_id", claimed.ID, "error", executionError)
		} else {
			logger.Info("task succeeded", "task_id", claimed.ID)
		}
	}
}

func execute(item task.Task) error {
	if item.Type != "print" {
		return fmt.Errorf("unsupported task type %q", item.Type)
	}
	var payload printPayload
	if err := json.Unmarshal(item.Payload, &payload); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}
	if payload.Message == "" {
		return errors.New("message is required")
	}
	slog.Info("print task", "message", payload.Message)
	return nil
}

func wait(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
