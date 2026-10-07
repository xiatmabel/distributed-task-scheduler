package task

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	db *pgxpool.Pool
}

func NewPostgresStore(db *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) Create(ctx context.Context, input CreateInput) (Task, error) {
	const query = `
		INSERT INTO tasks (type, payload, scheduled_at)
		VALUES ($1, $2, COALESCE($3, NOW()))
		RETURNING id, type, payload, status, attempts, scheduled_at,
		          started_at, finished_at, worker_id, error, created_at`

	return scanTask(s.db.QueryRow(ctx, query, input.Type, input.Payload, input.ScheduledAt))
}

func (s *PostgresStore) Get(ctx context.Context, id int64) (Task, error) {
	const query = `
		SELECT id, type, payload, status, attempts, scheduled_at,
		       started_at, finished_at, worker_id, error, created_at
		FROM tasks
		WHERE id = $1`

	result, err := scanTask(s.db.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Task{}, ErrNotFound
	}
	return result, err
}

func (s *PostgresStore) ClaimNext(ctx context.Context, workerID string) (Task, error) {
	const query = `
		WITH candidate AS (
			SELECT id
			FROM tasks
			WHERE status = 'pending' AND scheduled_at <= NOW()
			ORDER BY scheduled_at, id
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		UPDATE tasks
		SET status = 'running',
		    started_at = NOW(),
		    worker_id = $1,
		    attempts = attempts + 1
		FROM candidate
		WHERE tasks.id = candidate.id
		RETURNING tasks.id, tasks.type, tasks.payload, tasks.status, tasks.attempts,
		          tasks.scheduled_at, tasks.started_at, tasks.finished_at,
		          tasks.worker_id, tasks.error, tasks.created_at`

	return scanTask(s.db.QueryRow(ctx, query, workerID))
}

func (s *PostgresStore) Complete(ctx context.Context, id int64, executionError error) error {
	status := "succeeded"
	var message *string
	if executionError != nil {
		status = "failed"
		text := executionError.Error()
		message = &text
	}

	command, err := s.db.Exec(ctx, `
		UPDATE tasks
		SET status = $2, finished_at = NOW(), error = $3
		WHERE id = $1 AND status = 'running'`,
		id, status, message,
	)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(...any) error
}

func scanTask(row rowScanner) (Task, error) {
	var result Task
	err := row.Scan(
		&result.ID,
		&result.Type,
		&result.Payload,
		&result.Status,
		&result.Attempts,
		&result.ScheduledAt,
		&result.StartedAt,
		&result.FinishedAt,
		&result.WorkerID,
		&result.Error,
		&result.CreatedAt,
	)
	return result, err
}
