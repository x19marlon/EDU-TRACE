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

	user := auth.UserFrom(r.Context())
	h.store.RecordFeedback(user.ID)

	slog.Info("streaming feedback request received",
		"user_id", user.ID,
		"code_size", len(req.Code),
		"compiled", req.Success,
	)

	// Set SSE headers.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, model.ErrorResponse{
			Error: "streaming not supported",
		})
		return
	}

	start := time.Now()

	err := h.ollama.StreamFeedback(r.Context(), req, func(token string) {
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

	// Send done event with metadata.
	donePayload, _ := json.Marshal(map[string]any{
		"model":   h.modelName,
		"time_ms": elapsed,
	})
	fmt.Fprintf(w, "event: done\ndata: %s\n\n", donePayload)
	flusher.Flush()

	slog.Info("streaming feedback completed", "time_ms", elapsed)
}
