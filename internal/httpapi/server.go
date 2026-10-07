package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/xiatmabel/distributed-task-scheduler/internal/task"
)

type Server struct {
	store  task.Store
	logger *slog.Logger
}

func New(store task.Store, logger *slog.Logger) http.Handler {
	server := &Server{store: store, logger: logger}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", server.health)
	mux.HandleFunc("POST /tasks", server.createTask)
	mux.HandleFunc("GET /tasks/{id}", server.getTask)
	return mux
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	var input task.CreateInput
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(input.Type) == "" {
		writeError(w, http.StatusBadRequest, "type is required")
		return
	}
	if len(input.Payload) == 0 || !json.Valid(input.Payload) {
		writeError(w, http.StatusBadRequest, "payload must be valid JSON")
		return
	}

	created, err := s.store.Create(r.Context(), input)
	if err != nil {
		s.logger.Error("create task", "error", err)
		writeError(w, http.StatusInternalServerError, "could not create task")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) getTask(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	found, err := s.store.Get(r.Context(), id)
	if errors.Is(err, task.ErrNotFound) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		s.logger.Error("get task", "task_id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "could not get task")
		return
	}
	writeJSON(w, http.StatusOK, found)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Error("encode response", "error", err)
	}
}
