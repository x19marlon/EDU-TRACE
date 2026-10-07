package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/marlon/edu-trace/internal/ai"
	"github.com/marlon/edu-trace/internal/auth"
	"github.com/marlon/edu-trace/internal/model"
)

const maxFeedbackCodeSize = 50 * 1024 // 50 KB

// FeedbackHandler handles POST /api/feedback requests using Server-Sent Events.
type FeedbackHandler struct {
	ollama    *ai.OllamaClient
	modelName string
	store     *auth.Store
}

// NewFeedbackHandler creates a handler backed by the given Ollama client.
func NewFeedbackHandler(ollama *ai.OllamaClient, modelName string, store *auth.Store) *FeedbackHandler {
	return &FeedbackHandler{ollama: ollama, modelName: modelName, store: store}
}

func (h *FeedbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req model.FeedbackRequest

	r.Body = http.MaxBytesReader(w, r.Body, 4*maxFeedbackCodeSize)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{
			Error:   "invalid request body",
			Details: err.Error(),
		})
		return
	}

	if len(req.Code) == 0 {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{
			Error: "code cannot be empty",
		})
		return
	}

	if len(req.Code) > maxFeedbackCodeSize {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{
			Error: "code exceeds maximum size (50 KB)",
		})
		return
	}

	switch req.Mode {
	case "":
		req.Mode = ai.ModeFormal
	case ai.ModeFormal, ai.ModeInformal:
	default:
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "tipo de retroalimentación inválido"})
		return
	}

	user := auth.UserFrom(r.Context())

	// El enunciado del taller solo se usa si el usuario tiene acceso a ese taller.
	var statement string
	if req.AssignmentID != "" {
		st, err := h.store.AssignmentStatement(user, req.AssignmentID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, model.ErrorResponse{Error: "taller no encontrado"})
			return
		}
		statement = st
	}

	h.store.RecordFeedback(user.ID)

	slog.Info("streaming feedback request received",
		"user_id", user.ID,
		"mode", req.Mode,
		"code_size", len(req.Code),
		"compiled", req.Success,
	)

	streamSSE(w, h.modelName, func(onToken func(string)) error {
		return h.ollama.StreamFeedback(r.Context(), req, statement, onToken)
	})
}

// streamSSE envía la respuesta del modelo como Server-Sent Events: un evento por
// trozo de texto, "error" si falla y "done" con el tiempo empleado.
func streamSSE(w http.ResponseWriter, modelName string, run func(onToken func(string)) error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, model.ErrorResponse{Error: "streaming not supported"})
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	start := time.Now()
	err := run(func(token string) {
		payload, _ := json.Marshal(map[string]string{"token": token})
		fmt.Fprintf(w, "data: %s\n\n", payload)
		flusher.Flush()
	})
	elapsed := time.Since(start).Milliseconds()

	if err != nil {
		slog.Error("Ollama streaming failed", "error", err)
		errPayload, _ := json.Marshal(map[string]string{"error": err.Error()})
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", errPayload)
		flusher.Flush()
	}

	donePayload, _ := json.Marshal(map[string]any{"model": modelName, "time_ms": elapsed})
	fmt.Fprintf(w, "event: done\ndata: %s\n\n", donePayload)
	flusher.Flush()
	slog.Info("streaming completed", "time_ms", elapsed)
}
