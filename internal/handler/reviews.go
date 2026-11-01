package handler

import (
	"context"
	"errors"
	"fmt"
	"github.com/marlon/edu-trace/internal/ai"
	"github.com/marlon/edu-trace/internal/auth"
	"github.com/marlon/edu-trace/internal/model"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

const maxReviewFeedbackLen = 20000

type ReviewsHandler struct {
	store *auth.Store
	ollama *ai.OllamaClient // nil si Ollama no está disponible
}

func NewReviewsHandler(store *auth.Store, ollama *ai.OllamaClient) *ReviewsHandler {
	return &ReviewsHandler{store: store, ollama: ollama}
}

type reviewResponse struct {
	*auth.Review
	GradeFrom string `json:"grade_source"`
	AIJob *auth.JobState `json:"ai_job,omitempty"`
	AIAvailable bool `json:"ai_available"`
}

func validGrade(g *float64) (*float64, bool) {
	if g == nil {
		return nil, true
	}
	if math.IsNaN(*g) || math.IsInf(*g, 0) {
		return nil, false
	}
	v := math.Round(*g*10) / 10
	if v < 0 || v > ai.GradeMax {
		return nil, false
	}
	return &v, true
}

func (h *ReviewsHandler) Get(w http.ResponseWriter, r *http.Request) {
	teacher := auth.UserFrom(r.Context())
	rev, err := h.store.ReviewForTeacher(teacher.ID, r.PathValue("id"))
	if err != nil {
		writeStoreErr(w, err, "cargar la revision")
		return
	}
	h.respond(w, teacher.ID, rev)
}

func (h *ReviewsHandler) Draft(w http.ResponseWriter, r *http.Request) {
	if h.ollama == nil {
		writeJSON(w, http.StatusServiceUnavailable, model.ErrorResponse{Error: "Ollama no está configurado. La propuesta con IA no está disponible."})
		return
	}
	teacher := auth.UserFrom(r.Context())
	id := r.PathValue("id")
	start := time.Now()
	rev, err := h.generate(r.Context(), teacher.ID, id, false)
	h.store.RecordAIRequest(auth.AIRequest{Kind: auth.AIKindDraft, GroupID: h.store.SubmissionGroup(id), Role: teacher.Role,
		TotalMs: time.Since(start).Milliseconds(), OK: err == nil, Error: errText(err)})
	var aiErr *aiError
	switch {
	case errors.As(err, &aiErr):
		slog.Error("review draft failed", "submission_id", id, "error", err)
		writeJSON(w, http.StatusBadGateway, model.ErrorResponse{Error: aiErr.Error()})
		return
	case errors.Is(err, errNotOfficial):
		writeJSON(w, http.StatusNotFound, model.ErrorResponse{Error: err.Error()})
		return
	case err != nil:
		writeStoreErr(w, err, "generar la propuesta")
		return
	}
	h.respond(w, teacher.ID, rev)
}

type aiError struct{ err error }

func (e *aiError) Error() string { return e.err.Error() }
func (e *aiError) Unwrap() error { return e.err }

var errNotOfficial = errors.New("solo se revisan las entregas oficiales")

func errText(err error) string {
	if err == nil {
		return ""
	}
	return auth.JobError(err)
}

func (h *ReviewsHandler) generate(ctx context.Context, teacherID, id string, auto bool) (*auth.Review, error) {
	sub, err := h.store.SubmissionForTeacher(teacherID, id)
	if err != nil {
		return nil, err
	}
	if sub.IsAttempt() {
		return nil, errNotOfficial
	}
	current, err := h.store.ReviewForTeacher(teacherID, id)
	if err != nil {
		return nil, err
	}
	in := ai.ReviewInput{
		Code: sub.Code,
		Stdin: sub.Stdin,
		Success: sub.Success,
		CompilerOutput: sub.CompilerOutput,
		ProgramOutput: sub.ProgramOutput,
		RunError: sub.RunError,
		ExitCode: sub.ExitCode,
		Tests: sub.Tests,
	}
	for _, r := range current.RAEs {
		in.RAEs = append(in.RAEs, ai.RAEInput{Code: r.Code, Description: r.Description})
	}
	if sub.AssignmentID != "" {
		if statement, rubric, err := h.store.AssignmentForGrading(teacherID, sub.AssignmentID); err == nil {
			in.Statement, in.Rubric = statement, rubric
		}
	}

	sug, err := h.ollama.SuggestReview(ctx, in)
	if err != nil {
		return nil, &aiError{err}
	}

	grade, ok := validGrade(&sug.Grade)
	if !ok {
		grade = nil
	}
	levels := map[string]string{}
	for i, r := range current.RAEs {
		levels[r.RAEID] = sug.Levels[fmt.Sprintf("rae_%d", i+1)]
	}
	save := h.store.SaveAIDraft
	if auto {
		save = h.store.SaveAutoAIDraft
	}
	rev, err := save(teacherID, id, sug.Feedback, grade, sug.Reason, sug.Confidence, levels)
	if err != nil {
		return nil, err
	}
	if err := h.store.SetAIMinutes(teacherID, id, sug.Minutes); err != nil {
		slog.Error("estimación de tiempo de revisión", "submission_id", id, "error", err)
	} else if sug.Minutes > 0 {
		rev.AIMinutes = &sug.Minutes
	} else {
		rev.AIMinutes = nil
	}
	slog.Info("review draft created", "submission_id", id, "auto", auto, "ai_grade", sug.Grade, "confidence", sug.Confidence, "minutes", sug.Minutes, "sections_fixed", sug.SectionsFixed)
	return rev, nil
}

func (h *ReviewsHandler) respond(w http.ResponseWriter, teacherID string, rev *auth.Review) {
	job, err := h.store.JobFor(teacherID, rev.SubmissionID)
	if err != nil {
		slog.Error("estado de la cola de IA", "submission_id", rev.SubmissionID, "error", err)
	}
	writeJSON(w, http.StatusOK, reviewResponse{Review: rev, GradeFrom: rev.GradeSource(), AIJob: job, AIAvailable: h.ollama != nil})
}

func (h *ReviewsHandler) Discard(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ElapsedMs int64 `json:"elapsed_ms"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	teacher := auth.UserFrom(r.Context())
	id := r.PathValue("id")
	rev, err := h.store.DiscardAIDraft(teacher.ID, id)
	if errors.Is(err, auth.ErrAlreadyReturned) || errors.Is(err, auth.ErrNoAIDraft) {
		writeJSON(w, http.StatusConflict, model.ErrorResponse{Error: err.Error()})
		return
	}
	if err != nil {
		writeStoreErr(w, err, "descartar la propuesta")
		return
	}
	h.addTime(teacher.ID, rev, req.ElapsedMs)
	slog.Info("review draft discarded", "submission_id", id)
	h.respond(w, teacher.ID, rev)
}

func (h *ReviewsHandler) addTime(teacherID string, rev *auth.Review, ms int64) {
	added, err := h.store.AddReviewTime(teacherID, rev.SubmissionID, ms)
	if err != nil {
		slog.Error("tiempo de revisión", "submission_id", rev.SubmissionID, "error", err)
	}
	rev.ReviewMs += added
}

func (h *ReviewsHandler) AddTime(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ElapsedMs int64 `json:"elapsed_ms"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if _, err := h.store.AddReviewTime(auth.UserFrom(r.Context()).ID, r.PathValue("id"), req.ElapsedMs); err != nil {
		writeStoreErr(w, err, "guardar el tiempo de revisión")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ReviewsHandler) Audit(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.SubmissionAudit(auth.UserFrom(r.Context()).ID, r.PathValue("id"))
	if err != nil {
		writeStoreErr(w, err, "cargar la bitácora")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *ReviewsHandler) Save(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Feedback string `json:"feedback"`
		Grade *float64 `json:"grade"`
		Return bool `json:"return"`
		RAELevels map[string]string `json:"rae_levels"`
		ElapsedMs int64 `json:"elapsed_ms"`
	}
	if !decodeJSONLimit(w, r, &req, 256*1024) {
		return
	}

	feedback := strings.TrimSpace(req.Feedback)
	if utf8.RuneCountInString(feedback) > maxReviewFeedbackLen {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "la retroalimentación supera los 20.000 caracteres"})
		return
	}
	if req.Return && feedback == "" {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "escribe la retroalimentación antes de devolverla"})
		return
	}
	grade, ok := validGrade(req.Grade)
	if !ok {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "la nota debe estar entre 0.0 y 5.0"})
		return
	}

	if len(req.RAELevels) > 50 {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "demasiados RAE"})
		return
	}
	for _, l := range req.RAELevels {
		if !auth.ValidLevel(l) {
			writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "el nivel de cada RAE debe ser bajo, medio o alto"})
			return
		}
	}

	teacher := auth.UserFrom(r.Context())
	rev, err := h.store.SaveTeacherReview(teacher.ID, r.PathValue("id"), feedback, grade, req.Return, req.RAELevels)
	if err != nil {
		writeStoreErr(w, err, "guardar la revisión")
		return
	}
	h.addTime(teacher.ID, rev, req.ElapsedMs)
	slog.Info("review saved", "submission_id", rev.SubmissionID, "returned", req.Return, "grade_source", rev.GradeSource())
	h.respond(w, teacher.ID, rev)
}
