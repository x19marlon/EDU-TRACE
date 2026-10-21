package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/marlon/edu-trace/internal/ai"
	"github.com/marlon/edu-trace/internal/auth"
	"github.com/marlon/edu-trace/internal/model"
)

// ChatHandler handles interactive tutoring chat requests with SSE streaming.
type ChatHandler struct {
	ollama *ai.OllamaClient
	modelName string
	store *auth.Store
}

// NewChatHandler creates a new ChatHandler.
func NewChatHandler(ollama *ai.OllamaClient, modelName string, store *auth.Store) *ChatHandler {
	return &ChatHandler{ollama: ollama, modelName: modelName, store: store}
}

func (h *ChatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req model.ChatRequest
	r.Body = http.MaxBytesReader(w, r.Body, 4*maxFeedbackCodeSize)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{
			Error: "cuerpo de solicitud inválido",
			Details: err.Error(),
		})
		return
	}

	if len(req.Messages) == 0 {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{
			Error: "se requiere al menos un mensaje en el chat",
		})
		return
	}

	user := auth.UserFrom(r.Context())

	var statement string
	if req.AssignmentID != "" {
		st, err := h.store.AssignmentStatement(user, req.AssignmentID)
		if err == nil {
			statement = st
		}
	}

	slog.Info("chat message received",
		"user_id", user.ID,
		"messages_count", len(req.Messages),
		"code_size", len(req.Code),
	)

	streamSSE(w, h.modelName, func(onToken func(string)) error {
		return h.ollama.StreamChat(r.Context(), req, statement, onToken)
	})
}
