package handler

import (
	"errors"
	"github.com/marlon/edu-trace/internal/auth"
	"github.com/marlon/edu-trace/internal/model"
	"log/slog"
	"math"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxAssignmentFiles = auth.MaxAssignmentFiles
	maxAssignmentFileSize = 10 << 20
	maxAssignmentUpload = 30 << 20
	maxStatementLen = 20000
	maxRubricLen = 5000
	multipartMemoryLimit = 8 << 20
	assignmentTitleMaxLen = 120
	assignmentTitleMinLen = 2
	assignmentStatementMin = 1
)

type assignmentForm struct {
	title string
	statement string
	rubric string
	dueAt *time.Time
	policy auth.LatePolicy
	files []*multipart.FileHeader
}

func parseAssignmentForm(w http.ResponseWriter, r *http.Request) (*assignmentForm, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxAssignmentUpload+1<<20)
	if err := r.ParseMultipartForm(multipartMemoryLimit); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			writeJSON(w, http.StatusRequestEntityTooLarge, model.ErrorResponse{Error: "los archivos superan el máximo de 30 MB por taller"})
			return nil, false
		}
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "formulario inválido"})
		return nil, false
	}
	f := &assignmentForm{
		title: strings.TrimSpace(r.FormValue("title")),
		statement: strings.TrimSpace(r.FormValue("statement")),
		rubric: strings.TrimSpace(r.FormValue("rubric")),
		files: r.MultipartForm.File["files"],
	}
	fail := func(status int, msg string) (*assignmentForm, bool) {
		r.MultipartForm.RemoveAll()
		writeJSON(w, status, model.ErrorResponse{Error: msg})
		return nil, false
	}
	if n := utf8.RuneCountInString(f.title); n < assignmentTitleMinLen || n > assignmentTitleMaxLen {
		return fail(http.StatusBadRequest, "el título debe tener entre 2 y 120 caracteres")
	}
	if n := utf8.RuneCountInString(f.statement); n < assignmentStatementMin || n > maxStatementLen {
		return fail(http.StatusBadRequest, "escribe el enunciado (máximo 20.000 caracteres)")
	}
	if utf8.RuneCountInString(f.rubric) > maxRubricLen {
		return fail(http.StatusBadRequest, "los criterios de evaluacion superan los 5000 caracteres")
	}
	if v := strings.TrimSpace(r.FormValue("due_at")); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return fail(http.StatusBadRequest, "fecha límite inválida")
		}
		t = t.UTC()
		f.dueAt = &t
	}
	f.policy.AllowLate = r.FormValue("allow_late") != "false"
	if v := strings.TrimSpace(r.FormValue("late_until")); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return fail(http.StatusBadRequest, "fecha de cierre de entregas tardías inválida")
		}
		t = t.UTC()
		f.policy.LateUntil = &t
	}
	if v := strings.TrimSpace(strings.Replace(r.FormValue("late_penalty"), ",", ".", 1)); v != "" {
		p, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return fail(http.StatusBadRequest, "penalización por día inválida")
		}
		p = math.Round(p*10) / 10
		f.policy.LatePenalty = &p
	}
	if len(f.files) > maxAssignmentFiles {
		return fail(http.StatusBadRequest, "máximo 10 archivos por taller")
	}
	for _, fh := range f.files {
		if fh.Size > maxAssignmentFileSize {
			return fail(http.StatusBadRequest, "«"+fh.Filename+"» supera el máximo de 10 MB")
		}
	}
	return f, true
}

func openUploads(w http.ResponseWriter, headers []*multipart.FileHeader) ([]auth.FileUpload, func(), bool) {
	var opened []multipart.File
	closeAll := func() {
		for _, f := range opened {
			f.Close()
		}
	}
	uploads := make([]auth.FileUpload, 0, len(headers))
	for _, fh := range headers {
		f, err := fh.Open()
		if err != nil {
			closeAll()
			writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "no se pudo leer «" + fh.Filename + "»"})
			return nil, nil, false
		}
		opened = append(opened, f)
		uploads = append(uploads, auth.FileUpload{Name: fh.Filename, ContentType: fh.Header.Get("Content-Type"), Body: f})
	}
	return uploads, closeAll, true
}

func (h *ClassesHandler) CreateAssignment(w http.ResponseWriter, r *http.Request) {
	form, ok := parseAssignmentForm(w, r)
	if !ok {
		return
	}
	defer r.MultipartForm.RemoveAll()
	uploads, closeAll, ok := openUploads(w, form.files)
	if !ok {
		return
	}
	defer closeAll()

	user := auth.UserFrom(r.Context())
	a, err := h.store.CreateAssignment(user.ID, r.PathValue("id"), form.title, form.statement, form.rubric, form.dueAt, form.policy, uploads)
	if errors.Is(err, auth.ErrInvalidDeadline) {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}
	if err != nil {
		writeStoreErr(w, err, "crear el taller")
		return
	}
	slog.Info("assignment created", "user_id", user.ID, "group_id", a.GroupID, "assignment_id", a.ID, "files", len(a.Files))
	writeJSON(w, http.StatusCreated, a)
}

func (h *ClassesHandler) UpdateAssignment(w http.ResponseWriter, r *http.Request) {
	form, ok := parseAssignmentForm(w, r)
	if !ok {
		return
	}
	defer r.MultipartForm.RemoveAll()
	uploads, closeAll, ok := openUploads(w, form.files)
	if !ok {
		return
	}
	defer closeAll()

	user := auth.UserFrom(r.Context())
	a, err := h.store.UpdateAssignment(user.ID, r.PathValue("id"), form.title, form.statement, form.rubric, form.dueAt,
		form.policy, r.MultipartForm.Value["remove_files"], uploads)
	if errors.Is(err, auth.ErrTooManyFiles) || errors.Is(err, auth.ErrInvalidDeadline) {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}
	if err != nil {
		writeStoreErr(w, err, "guardar el taller")
		return
	}
	slog.Info("assignment updated", "user_id", user.ID, "assignment_id", a.ID, "files", len(a.Files))
	writeJSON(w, http.StatusOK, a)
}

func (h *ClassesHandler) DeleteAssignment(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFrom(r.Context())
	if err := h.store.DeleteAssignment(user.ID, r.PathValue("id")); err != nil {
		writeStoreErr(w, err, "eliminar el taller")
		return
	}
	slog.Info("assignment deleted", "user_id", user.ID, "assignment_id", r.PathValue("id"))
	w.WriteHeader(http.StatusNoContent)
}

func (h *ClassesHandler) GroupAssignments(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.GroupAssignments(auth.UserFrom(r.Context()).ID, r.PathValue("id"))
	if err != nil {
		writeStoreErr(w, err, "cargar los talleres")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *ClassesHandler) MyAssignments(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.store.StudentAssignments(auth.UserFrom(r.Context()).ID))
}

func (h *ClassesHandler) AssignmentFile(w http.ResponseWriter, r *http.Request) {
	path, meta, err := h.store.AssignmentFilePath(auth.UserFrom(r.Context()), r.PathValue("id"), r.PathValue("file"))
	if err != nil {
		writeStoreErr(w, err, "descargar el archivo")
		return
	}
	f, err := os.Open(path)
	if err != nil {
		writeStoreErr(w, err, "descargar el archivo")
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		writeStoreErr(w, err, "descargar el archivo")
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": meta.Name}))
	http.ServeContent(w, r, "", st.ModTime(), f)
}
