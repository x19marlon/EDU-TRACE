package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/marlon/edu-trace/internal/auth"
	"github.com/marlon/edu-trace/internal/compiler"
	"github.com/marlon/edu-trace/internal/model"
)

// ClassesHandler agrupa materias, clases, inscripciones y envíos.
type ClassesHandler struct {
	store      *auth.Store
	compiler   *compiler.Compiler
	joinLimits *auth.LoginLimiter
}

func NewClassesHandler(store *auth.Store, comp *compiler.Compiler) *ClassesHandler {
	return &ClassesHandler{
		store:      store,
		compiler:   comp,
		joinLimits: auth.NewLoginLimiter(10, 15*time.Minute),
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

// ---------- Docente ----------

// Courses maneja GET /api/teacher/courses.
func (h *ClassesHandler) Courses(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.store.TeacherCourses(auth.UserFrom(r.Context()).ID))
}

// CreateCourse maneja POST /api/teacher/courses.
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

// CreateGroup maneja POST /api/teacher/courses/{id}/groups.
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

// Group maneja GET /api/teacher/groups/{id}.
func (h *ClassesHandler) Group(w http.ResponseWriter, r *http.Request) {
	g, err := h.store.GroupForTeacher(auth.UserFrom(r.Context()).ID, r.PathValue("id"))
	if err != nil {
		writeStoreErr(w, err, "cargar la clase")
		return
	}
	writeJSON(w, http.StatusOK, g)
}

// GroupSubmissions maneja GET /api/teacher/groups/{id}/submissions?student=...
func (h *ClassesHandler) GroupSubmissions(w http.ResponseWriter, r *http.Request) {
	subs, err := h.store.GroupSubmissions(auth.UserFrom(r.Context()).ID, r.PathValue("id"), r.URL.Query().Get("student"))
	if err != nil {
		writeStoreErr(w, err, "cargar los envíos")
		return
	}
	writeJSON(w, http.StatusOK, subs)
}

// Submission maneja GET /api/teacher/submissions/{id}.
func (h *ClassesHandler) Submission(w http.ResponseWriter, r *http.Request) {
	sub, err := h.store.SubmissionForTeacher(auth.UserFrom(r.Context()).ID, r.PathValue("id"))
	if err != nil {
		writeStoreErr(w, err, "cargar el envío")
		return
	}
	writeJSON(w, http.StatusOK, sub)
}

// ---------- Estudiante ----------

// MyGroups maneja GET /api/student/groups.
func (h *ClassesHandler) MyGroups(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.store.StudentGroups(auth.UserFrom(r.Context()).ID))
}

// Join maneja POST /api/student/groups/join.
func (h *ClassesHandler) Join(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	user := auth.UserFrom(r.Context())
	// Limitar intentos fallidos para que no se puedan adivinar códigos.
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

// MySubmissions maneja GET /api/student/submissions.
func (h *ClassesHandler) MySubmissions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.store.StudentSubmissions(auth.UserFrom(r.Context()).ID, 20))
}

// Submit maneja POST /api/student/submissions. El servidor compila y ejecuta el
// código en el sandbox: el docente ve el resultado real, no uno enviado por el cliente.
func (h *ClassesHandler) Submit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GroupID    string `json:"group_id"`
		Code       string `json:"code"`
		Stdin      string `json:"stdin"`
		AIFeedback string `json:"ai_feedback"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4*maxCodeSize)
	if !decodeJSON(w, r, &req) {
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
	case len(req.AIFeedback) > maxCodeSize:
		req.AIFeedback = strings.ToValidUTF8(req.AIFeedback[:maxCodeSize], "")
	}

	user := auth.UserFrom(r.Context())
	if !h.store.IsEnrolled(user.ID, req.GroupID) {
		writeJSON(w, http.StatusForbidden, model.ErrorResponse{Error: auth.ErrNotEnrolled.Error()})
		return
	}

	result := h.compiler.Compile(r.Context(), req.Code, req.Stdin)
	h.store.RecordCompile(user.ID)
	sub := &auth.Submission{
		SubmissionMeta: auth.SubmissionMeta{
			GroupID:   req.GroupID,
			StudentID: user.ID,
			Success:   result.Success,
			RunError:  result.Error,
			ExitCode:  result.ExitCode,
		},
		Code:           req.Code,
		Stdin:          req.Stdin,
		CompilerOutput: result.CompilerOutput,
		ProgramOutput:  result.ProgramOutput,
		AIFeedback:     req.AIFeedback,
	}
	if err := h.store.AddSubmission(sub); err != nil {
		writeStoreErr(w, err, "guardar el envío")
		return
	}
	slog.Info("submission saved", "user_id", user.ID, "group_id", req.GroupID, "submission_id", sub.ID)
	writeJSON(w, http.StatusCreated, sub.SubmissionMeta)
}
