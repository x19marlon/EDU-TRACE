package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/marlon/edu-trace/internal/auth"
	"github.com/marlon/edu-trace/internal/model"
)

const maxMonitorFeedbackLen = 5000

type MonitorHandler struct {
	store *auth.Store
}

func NewMonitorHandler(store *auth.Store) *MonitorHandler {
	return &MonitorHandler{store: store}
}

type submissionItem struct {
	auth.SubmissionMeta
	ReviewState *auth.ReviewState `json:"review_state,omitempty"`
	MonitorFeedbackCount int `json:"monitor_feedback_count,omitempty"`
	AIJob string `json:"ai_job,omitempty"`
}

func submissionItems(store *auth.Store, subs []auth.SubmissionMeta, withReview bool) []submissionItem {
	ids := make([]string, len(subs))
	for i, m := range subs {
		ids[i] = m.ID
	}
	states, jobs := map[string]auth.ReviewState{}, map[string]string{}
	if withReview {
		states, jobs = store.ReviewStates(ids), store.JobStates(ids)
	}
	counts := store.MonitorFeedbackCounts(ids)
	out := make([]submissionItem, len(subs))
	for i, m := range subs {
		out[i] = submissionItem{SubmissionMeta: m, MonitorFeedbackCount: counts[m.ID], AIJob: jobs[m.ID]}
		if st, ok := states[m.ID]; ok {
			out[i].ReviewState = &st
		}
	}
	return out
}

type submissionView struct {
	*auth.Submission
	MonitorFeedback []auth.MonitorFeedback `json:"monitor_feedback,omitempty"`
	Annotations []auth.Annotation `json:"annotations"`
}

func (h *MonitorHandler) Groups(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.store.MonitorGroups(auth.UserFrom(r.Context()).ID))
}

func (h *MonitorHandler) Group(w http.ResponseWriter, r *http.Request) {
	g, err := h.store.GroupForMonitor(auth.UserFrom(r.Context()).ID, r.PathValue("id"))
	if err != nil {
		writeStoreErr(w, err, "cargar la clase")
		return
	}
	writeJSON(w, http.StatusOK, g)
}

func (h *MonitorHandler) Submissions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	subs, err := h.store.MonitorGroupSubmissions(auth.UserFrom(r.Context()).ID, r.PathValue("id"), q.Get("student"), q.Get("kind"))
	if err != nil {
		writeStoreErr(w, err, "cargar las entregas")
		return
	}
	writeJSON(w, http.StatusOK, submissionItems(h.store, subs, false))
}

func (h *MonitorHandler) Assignments(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.MonitorGroupAssignments(auth.UserFrom(r.Context()).ID, r.PathValue("id"))
	if err != nil {
		writeStoreErr(w, err, "cargar los talleres")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *MonitorHandler) Submission(w http.ResponseWriter, r *http.Request) {
	sub, err := h.store.SubmissionForMonitor(auth.UserFrom(r.Context()).ID, r.PathValue("id"))
	if err != nil {
		writeStoreErr(w, err, "cargar la entrega")
		return
	}
	user := auth.UserFrom(r.Context())
	writeJSON(w, http.StatusOK, submissionView{Submission: sub, MonitorFeedback: h.store.MonitorFeedbackFor(sub.ID), Annotations: h.store.AnnotationsFor(user, sub.ID)})
}

func (h *MonitorHandler) AddFeedback(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Body string `json:"body"`
	}
	if !decodeJSONLimit(w, r, &req, 64*1024) {
		return
	}
	body := strings.TrimSpace(req.Body)
	if n := utf8.RuneCountInString(body); n == 0 || n > maxMonitorFeedbackLen {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "escribe la retroalimentación (máximo 5000 caracteres)"})
		return
	}
	user := auth.UserFrom(r.Context())
	f, err := h.store.AddMonitorFeedback(user.ID, r.PathValue("id"), body)
	if errors.Is(err, auth.ErrAttemptFeedback) {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}
	if err != nil {
		writeStoreErr(w, err, "guardar la retroalimentación")
		return
	}
	slog.Info("monitor feedback added", "monitor_id", user.ID, "submission_id", f.SubmissionID)
	writeJSON(w, http.StatusCreated, f)
}

func (h *MonitorHandler) DeleteFeedback(w http.ResponseWriter, r *http.Request) {
	if err := h.store.DeleteMonitorFeedback(auth.UserFrom(r.Context()), r.PathValue("id")); err != nil {
		writeStoreErr(w, err, "eliminar la retroalimentación")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
