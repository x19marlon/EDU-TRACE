package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/marlon/edu-trace/internal/auth"
	"github.com/marlon/edu-trace/internal/compiler"
	"github.com/marlon/edu-trace/internal/model"
)

const (
	maxCodeSize  = 50 * 1024 // 50 KB
	maxStdinSize = 64 * 1024 // 64 KB
)

// CompileHandler handles POST /api/compile requests.
type CompileHandler struct {
	compiler *compiler.Compiler
	store    *auth.Store
}

// NewCompileHandler creates a handler backed by the given compiler.
func NewCompileHandler(c *compiler.Compiler, store *auth.Store) *CompileHandler {
	return &CompileHandler{compiler: c, store: store}
}

func (h *CompileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req model.CompileRequest

	r.Body = http.MaxBytesReader(w, r.Body, 2*(maxCodeSize+maxStdinSize))
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

	if len(req.Code) > maxCodeSize {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{
			Error: "code exceeds maximum size (50 KB)",
		})
		return
	}

	if len(req.Stdin) > maxStdinSize {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{
			Error: "stdin exceeds maximum size (64 KB)",
		})
		return
	}

	user := auth.UserFrom(r.Context())
	slog.Info("compile request received", "user_id", user.ID, "code_size", len(req.Code), "has_stdin", len(req.Stdin) > 0)
	h.store.RecordCompile(user.ID)

	result := h.compiler.Compile(r.Context(), req.Code, req.Stdin)

	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to write JSON response", "error", err)
	}
}
