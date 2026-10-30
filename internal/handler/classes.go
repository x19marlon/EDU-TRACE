package handler

import (
	"errors"
	"github.com/marlon/edu-trace/internal/auth"
	"github.com/marlon/edu-trace/internal/compiler"
	"github.com/marlon/edu-trace/internal/model"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

type ClassesHandler struct {
	store *auth.Store
	compiler *compiler.Compiler
	joinLimits *auth.LoginLimiter
	drafts *DraftQueue
}

func NewClassesHandler(store *auth.Store, comp *compiler.Compiler, drafts *DraftQueue) *ClassesHandler {
	return &ClassesHandler{
		store: store,
		compiler: comp,
		joinLimits: auth.NewLoginLimiter(10, 15*time.Minute),
		drafts: drafts,
	}
}

func validName(w http.ResponseWriter, name, what string) (string, bool) {
	name = strings.TrimSpace(name)
	if n := utf8.RuneCountInString(name); n < 2 || n > 80 {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "el nombre de la " + what + " debe tener entre 2 y 80 caracteres"})
		return "", false
	}
	return name, true
}

func writeStoreErr(w http.ResponseWriter, err error, action string) {
	switch {
	case errors.Is(err, auth.ErrNotFound):
		writeJSON(w, http.StatusNotFound, model.ErrorResponse{Error: "no encontrado"})
	case errors.Is(err, auth.ErrNotEnrolled):
		writeJSON(w, http.StatusForbidden, model.ErrorResponse{Error: err.Error()})
	default:
		slog.Error(action+" failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, model.ErrorResponse{Error: "no se pudo " + action})
	}
}

func (h *ClassesHandler) Courses(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.store.TeacherCourses(auth.UserFrom(r.Context()).ID))
}

func (h *ClassesHandler) CreateCourse(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	name, ok := validName(w, req.Name, "materia")
	if !ok {
		return
	}
	c, err := h.store.CreateCourse(auth.UserFrom(r.Context()).ID, name)
	if err != nil {
		writeStoreErr(w, err, "crear la materia")
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (h *ClassesHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	name, ok := validName(w, req.Name, "clase")
	if !ok {
		return
	}
	g, err := h.store.CreateGroup(auth.UserFrom(r.Context()).ID, r.PathValue("id"), name)
	if err != nil {
		writeStoreErr(w, err, "crear la clase")
		return
	}
	writeJSON(w, http.StatusCreated, g)
}

func (h *ClassesHandler) Group(w http.ResponseWriter, r *http.Request) {
	g, err := h.store.GroupForTeacher(auth.UserFrom(r.Context()).ID, r.PathValue("id"))
	if err != nil {
		writeStoreErr(w, err, "cargar la clase")
		return
	}
	writeJSON(w, http.StatusOK, g)
}

func (h *ClassesHandler) GroupSubmissions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	subs, err := h.store.GroupSubmissions(auth.UserFrom(r.Context()).ID, r.PathValue("id"), q.Get("student"), q.Get("kind"))
	if err != nil {
		writeStoreErr(w, err, "cargar los envíos")
		return
	}
	writeJSON(w, http.StatusOK, submissionItems(h.store, subs, true))
}

func (h *ClassesHandler) Submission(w http.ResponseWriter, r *http.Request) {
	sub, err := h.store.SubmissionForTeacher(auth.UserFrom(r.Context()).ID, r.PathValue("id"))
	if err != nil {
		writeStoreErr(w, err, "cargar el envío")
		return
	}
	user := auth.UserFrom(r.Context())
	writeJSON(w, http.StatusOK, submissionView{Submission: sub, MonitorFeedback: h.store.MonitorFeedbackFor(sub.ID), Annotations: h.store.AnnotationsFor(user, sub.ID)})
}

func (h *ClassesHandler) MyGroups(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.store.StudentGroups(auth.UserFrom(r.Context()).ID))
}

func (h *ClassesHandler) Join(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	user := auth.UserFrom(r.Context())

	if h.joinLimits.Blocked(user.ID) {
		writeJSON(w, http.StatusTooManyRequests, model.ErrorResponse{Error: "demasiados intentos, espera unos minutos"})
		return
	}
	g, err := h.store.JoinGroup(user.ID, req.Code)
	if errors.Is(err, auth.ErrInvalidCode) {
		h.joinLimits.Fail(user.ID)
		writeJSON(w, http.StatusNotFound, model.ErrorResponse{Error: err.Error()})
		return
	}
	if err != nil {
		writeStoreErr(w, err, "unirte a la clase")
		return
	}
	slog.Info("student joined group", "user_id", user.ID, "group_id", g.ID)
	writeJSON(w, http.StatusOK, g)
}

func (h *ClassesHandler) MySubmissions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.store.StudentSubmissions(auth.UserFrom(r.Context()).ID, 20))
}

func (h *ClassesHandler) Submit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GroupID string `json:"group_id"`
		AssignmentID string `json:"assignment_id"`
		Code string `json:"code"`
		Stdin string `json:"stdin"`
		AIFeedback string `json:"ai_feedback"`
		AIFeedbackInformal string `json:"ai_feedback_informal"`
	}

	if !decodeJSONLimit(w, r, &req, 2*(2*maxCodeSize+maxStdinSize+maxCodeSize)) {
		return
	}
	switch {
	case strings.TrimSpace(req.Code) == "":
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "el código está vacío"})
		return
	case len(req.Code) > maxCodeSize:
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "el código supera el tamaño máximo (50 KB)"})
		return
	case len(req.Stdin) > maxStdinSize:
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "la entrada supera el tamaño máximo (64 KB)"})
		return
	}
	req.AIFeedback = clip(req.AIFeedback, maxCodeSize)
	req.AIFeedbackInformal = clip(req.AIFeedbackInformal, maxCodeSize)

	user := auth.UserFrom(r.Context())
	if !h.store.IsEnrolled(user.ID, req.GroupID) {
		writeJSON(w, http.StatusForbidden, model.ErrorResponse{Error: auth.ErrNotEnrolled.Error()})
		return
	}
	if req.AssignmentID != "" && !h.store.AssignmentInGroup(req.AssignmentID, req.GroupID) {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "ese taller no pertenece a la clase elegida"})
		return
	}

	tests, err := h.store.TestsForRun(req.AssignmentID)
	if err != nil {
		writeStoreErr(w, err, "cargar los casos de prueba")
		return
	}
	result := h.compiler.CompileAndTest(r.Context(), req.Code, req.Stdin, tests)
	h.store.RecordCompile(user.ID)
	sub := &auth.Submission{
		SubmissionMeta: auth.SubmissionMeta{
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
		AIFeedback: req.AIFeedback,
		AIFeedbackInf: req.AIFeedbackInformal,
		Tests: result.Tests,
	}
	if err := h.store.AddSubmission(sub); err != nil {
		var closed *auth.DeadlineError
		if errors.As(err, &closed) {
			writeJSON(w, http.StatusConflict, map[string]any{"error": closed.Error(), "closed_at": closed.ClosedAt})
			return
		}
		writeStoreErr(w, err, "guardar el envío")
		return
	}
	h.drafts.Wake()
	slog.Info("submission saved", "user_id", user.ID, "group_id", req.GroupID, "submission_id", sub.ID)
	writeJSON(w, http.StatusCreated, submittedView{SubmissionMeta: sub.SubmissionMeta, Tests: model.PublicResults(sub.Tests)})
}

type submittedView struct {
	auth.SubmissionMeta
	Tests []model.TestResult `json:"tests,omitempty"`
}

func clip(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return strings.ToValidUTF8(s[:max], "")
}

func (h *ClassesHandler) MyHistory(w http.ResponseWriter, r *http.Request) {
	hist, err := h.store.History(auth.UserFrom(r.Context()).ID)
	if err != nil {
		writeStoreErr(w, err, "cargar tu historial")
		return
	}
	writeJSON(w, http.StatusOK, hist)
}

func (h *ClassesHandler) MySubmission(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFrom(r.Context())
	sub, err := h.store.OwnSubmission(user.ID, r.PathValue("id"))
	if err != nil {
		writeStoreErr(w, err, "cargar la entrega")
		return
	}
	writeJSON(w, http.StatusOK, submissionView{Submission: sub, Annotations: h.store.AnnotationsFor(user, sub.ID)})
}
