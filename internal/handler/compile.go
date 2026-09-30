package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/marlon/edu-trace/internal/compiler"
	"github.com/marlon/edu-trace/internal/model"
)

const maxCodeSize = 50 * 1024 // 50 KB

// CompileHandler handles POST /api/compile requests.
type CompileHandler struct {
	compiler *compiler.Compiler
}

// NewCompileHandler creates a handler backed by the given compiler.
func NewCompileHandler(c *compiler.Compiler) *CompileHandler {
	return &CompileHandler{compiler: c}
}

func (h *CompileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req model.CompileRequest

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

	slog.Info("compile request received", "code_size", len(req.Code), "has_stdin", len(req.Stdin) > 0)

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
