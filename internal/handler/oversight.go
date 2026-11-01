package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/marlon/edu-trace/internal/auth"
	"github.com/marlon/edu-trace/internal/model"
)

func (h *ClassesHandler) SetAIMode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AIMode string `json:"ai_mode"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	teacher := auth.UserFrom(r.Context())
	err := h.store.SetGroupAIMode(teacher.ID, r.PathValue("id"), req.AIMode)
	if errors.Is(err, auth.ErrInvalidAIMode) {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}
	if err != nil {
		writeStoreErr(w, err, "cambiar el modo de la IA")
		return
	}
	slog.Info("ai mode changed", "user_id", teacher.ID, "group_id", r.PathValue("id"), "ai_mode", req.AIMode)
	writeJSON(w, http.StatusOK, map[string]string{"ai_mode": req.AIMode})
}

func (h *ClassesHandler) Metrics(w http.ResponseWriter, r *http.Request) {
	m, err := h.store.GroupMetrics(auth.UserFrom(r.Context()).ID, r.PathValue("id"))
	if err != nil {
		writeStoreErr(w, err, "calcular las métricas")
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (h *ClassesHandler) SetThresholds(w http.ResponseWriter, r *http.Request) {
	var req auth.Thresholds
	if !decodeJSON(w, r, &req) {
		return
	}
	err := h.store.SetCourseThresholds(auth.UserFrom(r.Context()).ID, r.PathValue("id"), req)
	if errors.Is(err, auth.ErrInvalidThreshold) {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}
	if err != nil {
		writeStoreErr(w, err, "guardar los umbrales")
		return
	}
	writeJSON(w, http.StatusOK, req)
}

func (h *ClassesHandler) GroupAudit(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	list, err := h.store.GroupAudit(auth.UserFrom(r.Context()).ID, r.PathValue("id"), limit)
	if err != nil {
		writeStoreErr(w, err, "cargar la bitácora")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *ClassesHandler) RetryDrafts(w http.ResponseWriter, r *http.Request) {
	n, err := h.store.RetryFailedJobs(auth.UserFrom(r.Context()).ID, r.PathValue("id"))
	if err != nil {
		writeStoreErr(w, err, "reintentar las propuestas")
		return
	}
	h.drafts.Wake()
	writeJSON(w, http.StatusOK, map[string]int64{"retried": n})
}

func (h *ClassesHandler) RerunTests(w http.ResponseWriter, r *http.Request) {
	teacher := auth.UserFrom(r.Context())
	sub, err := h.store.SubmissionForTeacher(teacher.ID, r.PathValue("id"))
	if err != nil {
		writeStoreErr(w, err, "cargar la entrega")
		return
	}
	tests, err := h.store.TestsForRun(sub.AssignmentID)
	if err != nil {
		writeStoreErr(w, err, "cargar los casos de prueba")
		return
	}
	if len(tests) == 0 {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "el taller de esta entrega no tiene casos de prueba"})
		return
	}
	result := h.compiler.CompileAndTest(r.Context(), sub.Code, sub.Stdin, tests)
	if err := h.store.SaveSubmissionTests(teacher.ID, sub.ID, result.Tests); err != nil {
		writeStoreErr(w, err, "guardar los resultados")
		return
	}
	writeJSON(w, http.StatusOK, result.Tests)
}

func (h *ClassesHandler) ReviewPlan(w http.ResponseWriter, r *http.Request) {
	plan, err := h.store.ReviewPlan(auth.UserFrom(r.Context()).ID, r.PathValue("id"))
	if err != nil {
		writeStoreErr(w, err, "estimar el tiempo de revisión")
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (h *ClassesHandler) FrequentErrors(w http.ResponseWriter, r *http.Request) {
	report, err := h.store.FrequentErrors(auth.UserFrom(r.Context()).ID, r.PathValue("id"), r.URL.Query().Get("alcance") == "materia")
	if err != nil {
		writeStoreErr(w, err, "calcular los errores frecuentes")
		return
	}
	writeJSON(w, http.StatusOK, report)
}
