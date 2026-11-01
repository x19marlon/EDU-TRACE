package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/marlon/edu-trace/internal/ai"
	"github.com/marlon/edu-trace/internal/auth"
	"github.com/marlon/edu-trace/internal/model"
)

const maxFeedbackCodeSize = 50 * 1024 // 50 KB

// FeedbackHandler handles POST /api/feedback requests using Server-Sent Events.
type FeedbackHandler struct {
	ollama *ai.OllamaClient
	modelName string
	store *auth.Store
	limit *auth.LoginLimiter
}

// NewFeedbackHandler creates a handler backed by the given Ollama client.
func NewFeedbackHandler(ollama *ai.OllamaClient, modelName string, store *auth.Store, limit *auth.LoginLimiter) *FeedbackHandler {
	return &FeedbackHandler{ollama: ollama, modelName: modelName, store: store, limit: limit}
}

func (h *FeedbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req model.FeedbackRequest

	r.Body = http.MaxBytesReader(w, r.Body, 4*maxFeedbackCodeSize)
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
	case ai.ModeHint:
		if req.Level < 1 || req.Level > ai.MaxHintLevel {
			writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "el nivel de la pista va de 1 a 3"})
			return
		}
		if len(req.Previous) > ai.MaxHintLevel-1 {
			req.Previous = req.Previous[:ai.MaxHintLevel-1]
		}
	default:
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "tipo de retroalimentación inválido"})
		return
	}

	user := auth.UserFrom(r.Context())
	if !aiAllowed(w, h.store, h.limit, user, req.AssignmentID) {
		return
	}

	// El enunciado del taller solo se usa si el usuario tiene acceso a ese taller.
	var statement, rubric string
	if req.AssignmentID != "" {
		st, rb, err := h.store.AssignmentContext(user, req.AssignmentID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, model.ErrorResponse{Error: "taller no encontrado"})
			return
		}
		statement, rubric = st, rb
	} else if req.ExerciseID != "" {
		statement = exerciseStatement(req.ExerciseID)
	}

	h.store.RecordFeedback(user.ID)

	slog.Info("streaming feedback request received",
		"user_id", user.ID,
		"mode", req.Mode,
		"code_size", len(req.Code),
		"compiled", req.Success,
	)

	if req.Mode == ai.ModeHint {
		stats := streamSSE(w, h.modelName, func(onToken, _ func(string)) error {
			return h.ollama.StreamHint(r.Context(), req, statement, onToken)
		})
		h.store.RecordAIRequest(stats.record(h.ollama, auth.AIKindHint, h.store.AssignmentGroup(req.AssignmentID), user.Role))
		return
	}

	var checks ai.Checks
	stats := streamSSE(w, h.modelName, func(onToken, onReplace func(string)) error {
		c, err := h.ollama.StreamFeedback(r.Context(), req, statement, rubric, onToken, onReplace)
		checks = c
		return err
	})
	kind := auth.AIKindFormal
	if req.Mode == ai.ModeInformal {
		kind = auth.AIKindInformal
	}
	rec := stats.record(h.ollama, kind, h.store.AssignmentGroup(req.AssignmentID), user.Role)
	if stats.err == nil {
		rec.SectionsOK, rec.QualityOK = &checks.SectionsOK, &checks.QualityOK
	}
	h.store.RecordAIRequest(rec)
}

const supervisedMessage = "En tu clase la retroalimentación de la IA pasa primero por tu profesor: envía tu código y la verás en «Mis entregas» después de su revisión y aprobación."

func evaluationMessage(e *auth.OpenEvaluation) string {
	return "Hay una evaluación en curso («" + e.Title + "»): la IA y el chat están desactivados hasta el " +
		e.Until.In(time.Local).Format("02/01/2006 a las 15:04") + "."
}

func aiAllowed(w http.ResponseWriter, store *auth.Store, limit *auth.LoginLimiter, user *auth.User, assignmentID string) bool {
	if user.Role != auth.RoleStudent {
		return true
	}
	block, err := store.StudentAIBlock(user.ID, assignmentID)
	if err != nil {
		writeStoreErr(w, err, "consultar el modo de la IA")
		return false
	}
	switch {
	case block == nil:
		if limit == nil {
			return true
		}
		wait, ok := limit.Take(user.ID)
		if ok {
			return true
		}
		w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(wait.Seconds()))))
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"code": "limite",
			"error": fmt.Sprintf("Llegaste al límite de %d pedidos a la IA por hora. Podrás pedir otro en %d min; mientras tanto, revisa los mensajes del compilador y los casos de prueba.",
				limit.Max(), int(math.Ceil(wait.Minutes())))})
	case block.Code == auth.AIBlockEvaluation:
		writeJSON(w, http.StatusForbidden, map[string]string{"error": evaluationMessage(block.Evaluation), "code": block.Code})
	default:
		writeJSON(w, http.StatusForbidden, map[string]string{"error": supervisedMessage, "code": block.Code})
	}
	return false
}

type streamStats struct {
	firstMs int64
	totalMs int64
	err error
}

func (st streamStats) record(o *ai.OllamaClient, kind, groupID string, role auth.Role) auth.AIRequest {
	rec := auth.AIRequest{Kind: kind, GroupID: groupID, Role: role, FirstMs: st.firstMs, TotalMs: st.totalMs, OK: st.err == nil,
		Model: o.Model(), PromptVersion: o.PromptVersion()}
	if st.err != nil {
		rec.Error = auth.JobError(st.err)
	}
	return rec
}

// streamSSE envía la respuesta del modelo como Server-Sent Events: un evento por
// trozo de texto, "error" si falla y "done" con el tiempo empleado.
func streamSSE(w http.ResponseWriter, modelName string, run func(onToken, onReplace func(string)) error) streamStats {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, model.ErrorResponse{Error: "streaming not supported"})
		return streamStats{err: errors.New("streaming not supported")}
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	start := time.Now()
	var first int64
	err := run(func(token string) {
		if first == 0 {
			first = time.Since(start).Milliseconds() + 1
		}
		payload, _ := json.Marshal(map[string]string{"token": token})
		fmt.Fprintf(w, "data: %s\n\n", payload)
		flusher.Flush()
	}, func(text string) {
		payload, _ := json.Marshal(map[string]string{"text": text})
		fmt.Fprintf(w, "event: replace\ndata: %s\n\n", payload)
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
	return streamStats{firstMs: first, totalMs: elapsed, err: err}
}
