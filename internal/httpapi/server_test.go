package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/xiatmabel/distributed-task-scheduler/internal/task"
)

type fakeStore struct {
	created task.Task
	found   task.Task
	getErr  error
}

func (f *fakeStore) Create(_ context.Context, input task.CreateInput) (task.Task, error) {
	f.created = task.Task{
		ID:          1,
		Type:        input.Type,
		Payload:     input.Payload,
		Status:      "pending",
		ScheduledAt: time.Now(),
		CreatedAt:   time.Now(),
	}
	return f.created, nil
}

func (f *fakeStore) Get(_ context.Context, _ int64) (task.Task, error) {
	return f.found, f.getErr
}

func TestCreateTask(t *testing.T) {
	store := &fakeStore{}
	server := New(store, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewBufferString(
		`{"type":"print","payload":{"message":"hello"}}`,
	))
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, response.Code)
	}
	var created task.Task
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Type != "print" || created.Status != "pending" {
		t.Fatalf("unexpected task: %+v", created)
	}
}

func TestIndex(t *testing.T) {
	server := New(&fakeStore{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"service":"distributed-task-scheduler"`)) {
		t.Fatalf("unexpected response: %s", response.Body.String())
	}
}

func TestCreateTaskRejectsMissingType(t *testing.T) {
	server := New(&fakeStore{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewBufferString(
		`{"payload":{"message":"hello"}}`,
	))
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestGetTaskNotFound(t *testing.T) {
	server := New(
		&fakeStore{getErr: task.ErrNotFound},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	request := httptest.NewRequest(http.MethodGet, "/tasks/99", nil)
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, response.Code)
	}
}
