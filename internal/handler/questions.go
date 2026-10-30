package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/marlon/edu-trace/internal/auth"
	"github.com/marlon/edu-trace/internal/model"
)

const (
	maxQuestionLen = 1000
	maxAnswerLen = 5000
	maxRatingComment = 500
)

func textField(w http.ResponseWriter, text string, max int, what string) (string, bool) {
	text = strings.TrimSpace(text)
	if n := utf8.RuneCountInString(text); n < 2 || n > max {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "escribe " + what + " (máximo " + strconv.Itoa(max) + " caracteres)"})
		return "", false
	}
	return text, true
}

func (h *ClassesHandler) AskQuestion(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Body string `json:"body"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	body, ok := textField(w, req.Body, maxQuestionLen, "tu pregunta")
	if !ok {
		return
	}
	user := auth.UserFrom(r.Context())
	q, err := h.store.AskQuestion(user.ID, r.PathValue("id"), body)
	if errors.Is(err, auth.ErrTooManyQuestions) {
		writeJSON(w, http.StatusConflict, model.ErrorResponse{Error: err.Error()})
		return
	}
	if err != nil {
		writeStoreErr(w, err, "enviar tu pregunta")
		return
	}
	slog.Info("feedback question asked", "user_id", user.ID, "submission_id", q.SubmissionID)
	writeJSON(w, http.StatusCreated, q)
}

func (h *ClassesHandler) AnswerQuestion(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Answer string `json:"answer"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	answer, ok := textField(w, req.Answer, maxAnswerLen, "la respuesta")
	if !ok {
		return
	}
	user := auth.UserFrom(r.Context())
	q, err := h.store.AnswerQuestion(user, r.PathValue("id"), answer)
	if err != nil {
		writeStoreErr(w, err, "responder la pregunta")
		return
	}
	slog.Info("feedback question answered", "user_id", user.ID, "question_id", q.ID)
	writeJSON(w, http.StatusOK, q)
}

func (h *ClassesHandler) OpenQuestions(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.TeacherOpenQuestions(auth.UserFrom(r.Context()).ID)
	if err != nil {
		writeStoreErr(w, err, "cargar las preguntas")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *ClassesHandler) RateFeedback(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Kind string `json:"kind"`
		SubmissionID string `json:"submission_id"`
		AssignmentID string `json:"assignment_id"`
		Understood string `json:"understood"`
		Comment string `json:"comment"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if utf8.RuneCountInString(req.Comment) > maxRatingComment {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "el comentario tiene máximo 500 caracteres"})
		return
	}
	user := auth.UserFrom(r.Context())
	var err error
	if req.Kind == auth.RatingReview {
		err = h.store.RateReview(user.ID, req.SubmissionID, req.Understood, req.Comment)
	} else {
		err = h.store.RateInstant(user.ID, req.Kind, req.AssignmentID, req.Understood, req.Comment)
	}
	if errors.Is(err, auth.ErrInvalidRating) {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}
	if err != nil {
		writeStoreErr(w, err, "guardar tu respuesta")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
