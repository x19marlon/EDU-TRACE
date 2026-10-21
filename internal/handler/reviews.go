package handler

import (
	"github.com/marlon/edu-trace/internal/ai"
	"github.com/marlon/edu-trace/internal/auth"
	"github.com/marlon/edu-trace/internal/model"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"unicode/utf8"
)

const maxReviewFeedbackLen = 20000

type ReviewsHandler struct {
	store  *auth.Store
	ollama *ai.OllamaClient // nil si Ollama no está disponible
}

func NewReviewsHandler(store *auth.Store, ollama *ai.OllamaClient) *ReviewsHandler {
	return &ReviewsHandler{store: store, ollama: ollama}
}

type reviewResponse struct {
	*auth.Review
	GradeFrom string `json:"grade_source"`
}

func respondReview(w http.ResponseWriter, rev *auth.Review) {
	writeJSON(w, http.StatusOK, reviewResponse{Review: rev, GradeFrom: rev.GradeSource()})
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
	rev, err := h.store.ReviewForTeacher(auth.UserFrom(r.Context()).ID, r.PathValue("id"))
	if err != nil {
		writeStoreErr(w, err, "cargar la revision")
		return
	}
	respondReview(w, rev)
}

func (h *ReviewsHandler) Draft(w http.ResponseWriter, r *http.Request) {
	if h.ollama == nil {
		writeJSON(w, http.StatusServiceUnavailable, model.ErrorResponse{Error: "Ollama no está configurado. La propuesta con IA no está disponible."})
		return
	}
	teacher := auth.UserFrom(r.Context())
	id := r.PathValue("id")
	sub, err := h.store.SubmissionForTeacher(teacher.ID, id)
	if err != nil {
		writeStoreErr(w, err, "cargar la entrega")
		return
	}
	if sub.IsAttempt() {
		writeJSON(w, http.StatusNotFound, model.ErrorResponse{Error: "solo se revisan las entregas oficiales"})
		return
	}

	in := ai.ReviewInput{
		Code:           sub.Code,
		Stdin:          sub.Stdin,
		Success:        sub.Success,
		CompilerOutput: sub.CompilerOutput,
		ProgramOutput:  sub.ProgramOutput,
		RunError:       sub.RunError,
		ExitCode:       sub.ExitCode,
	}
	if sub.AssignmentID != "" {
		if statement, rubric, err := h.store.AssignmentForGrading(teacher.ID, sub.AssignmentID); err == nil {
			in.Statement, in.Rubric = statement, rubric
		}
	}

	sug, err := h.ollama.SuggestReview(r.Context(), in)
	if err != nil {
		slog.Error("review draft failed", "submission_id", id, "error", err)
		writeJSON(w, http.StatusBadGateway, model.ErrorResponse{Error: err.Error()})
		return
	}

	grade, ok := validGrade(&sug.Grade)
	if !ok {
		grade = nil
	}
	rev, err := h.store.SaveAIDraft(teacher.ID, id, sug.Feedback, grade, sug.Reason, sug.Confidence)
	if err != nil {
		writeStoreErr(w, err, "guardar la propuesta")
		return
	}
	slog.Info("review draft created", "submission_id", id, "ai_grade", sug.Grade, "confidence", sug.Confidence)
	respondReview(w, rev)
}

func (h *ReviewsHandler) Save(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Feedback string   `json:"feedback"`
		Grade    *float64 `json:"grade"`
		Return   bool     `json:"return"`
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

	teacher := auth.UserFrom(r.Context())
	rev, err := h.store.SaveTeacherReview(teacher.ID, r.PathValue("id"), feedback, grade, req.Return)
	if err != nil {
		writeStoreErr(w, err, "guardar la revisión")
		return
	}
	slog.Info("review saved", "submission_id", rev.SubmissionID, "returned", req.Return, "grade_source", rev.GradeSource())
	respondReview(w, rev)
}
