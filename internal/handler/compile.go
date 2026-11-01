package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/marlon/edu-trace/internal/auth"
	"github.com/marlon/edu-trace/internal/compiler"
	"github.com/marlon/edu-trace/internal/model"
	"github.com/marlon/edu-trace/internal/practice"
)

const (
	maxCodeSize = 50 * 1024 // 50 KB
	maxStdinSize = 64 * 1024 // 64 KB
)

// CompileHandler handles POST /api/compile requests.
type CompileHandler struct {
	compiler *compiler.Compiler
	store *auth.Store
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
			Error: "invalid request body",
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

	own, err := ownTests(req.OwnTests)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "tus casos de prueba no son válidos: " + err.Error()})
		return
	}
	var tests []model.TestCase
	exercise, isExercise := practice.Get(req.ExerciseID)
	enrolled := false
	switch {
	case req.ExerciseID != "" && !isExercise:
		writeJSON(w, http.StatusNotFound, model.ErrorResponse{Error: "ese ejercicio no existe"})
		return
	case isExercise:
		tests = exercise.Tests
	default:
		enrolled = user.Role == auth.RoleStudent && req.GroupID != "" && h.store.IsEnrolled(user.ID, req.GroupID)
		if !enrolled || (req.AssignmentID != "" && !h.store.AssignmentInGroup(req.AssignmentID, req.GroupID)) {
			req.AssignmentID = ""
		}
		if tests, err = h.store.TestsForRun(req.AssignmentID); err != nil {
			slog.Error("load tests failed", "assignment_id", req.AssignmentID, "error", err)
		}
	}
	result := h.compiler.CompileAndTest(r.Context(), req.Code, req.Stdin, append(tests, own...))
	if enrolled {
		official := *result
		official.Tests = result.Tests[:len(tests)]
		h.saveAttempt(user, req, &official)
	}
	if isExercise && user.Role == auth.RoleStudent {
		passed := 0
		for _, t := range result.Tests[:len(tests)] {
			if t.Passed {
				passed++
			}
		}
		if err := h.store.RecordPractice(user.ID, exercise.ID, passed, len(tests)); err != nil {
			slog.Error("guardar el resultado de la práctica", "user_id", user.ID, "exercise", exercise.ID, "error", err)
		}
	}
	result.Tests = model.PublicResults(result.Tests)
	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to write JSON response", "error", err)
	}
}

// saveAttempt guarda la compilación como intento si el estudiante eligió una clase
// en la que está inscrito. Nunca hace fallar la compilación.
func (h *CompileHandler) saveAttempt(user *auth.User, req model.CompileRequest, result *model.CompileResult) {
	err := h.store.AddSubmission(&auth.Submission{
		SubmissionMeta: auth.SubmissionMeta{
			Kind: auth.KindAttempt,
			GroupID: req.GroupID,
			AssignmentID: req.AssignmentID,
			StudentID: user.ID,
			Success: result.Success,
			RunError: result.Error,
			ExitCode: result.ExitCode,
			ErrorSummary: compiler.FirstError(result.CompilerOutput),
		},
		Code: req.Code,
		Stdin: req.Stdin,
		CompilerOutput: result.CompilerOutput,
		ProgramOutput: result.ProgramOutput,
		Tests: result.Tests,
	})
	if err != nil && !errors.Is(err, auth.ErrDuplicateAttempt) && !errors.Is(err, auth.ErrCourseClosed) {
		slog.Error("save attempt failed", "user_id", user.ID, "error", err)
	}
}
