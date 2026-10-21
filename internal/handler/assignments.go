package handler

import (
	"errors"
	"github.com/marlon/edu-trace/internal/auth"
	"github.com/marlon/edu-trace/internal/model"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxAssignmentFiles     = 10
	maxAssignmentFileSize  = 10 << 20
	maxAssignmentUpload    = 30 << 20
	maxStatementLen        = 20000
	maxRubricLen           = 5000
	multipartMemoryLimit   = 8 << 20
	assignmentTitleMaxLen  = 120
	assignmentTitleMinLen  = 2
	assignmentStatementMin = 1
)

func (h *ClassesHandler) CreateAssignment(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxAssignmentUpload+1<<20)
	if err := r.ParseMultipartForm(multipartMemoryLimit); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			writeJSON(w, http.StatusRequestEntityTooLarge, model.ErrorResponse{Error: "los archivos superan el máximo de 30 MB por taller"})
			return
		}
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "formulario inválido"})
		return
	}
	defer r.MultipartForm.RemoveAll()

	title := strings.TrimSpace(r.FormValue("title"))
	statement := strings.TrimSpace(r.FormValue("statement"))
	rubric := strings.TrimSpace(r.FormValue("rubric"))
	if n := utf8.RuneCountInString(title); n < assignmentTitleMinLen || n > assignmentTitleMaxLen {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "el título debe tener entre 2 y 120 caracteres"})
		return
	}
	if n := utf8.RuneCountInString(statement); n < assignmentStatementMin || n > maxStatementLen {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "escribe el enunciado (máximo 20.000 caracteres)"})
		return
	}
	if utf8.RuneCountInString(rubric) > maxRubricLen {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "los criterios de evaluacion superan los 5000 caracteres"})
		return
	}
	var dueAt *time.Time
	if v := strings.TrimSpace(r.FormValue("due_at")); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "fecha límite inválida"})
			return
		}
		t = t.UTC()
		dueAt = &t
	}

	headers := r.MultipartForm.File["files"]
	if len(headers) > maxAssignmentFiles {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "máximo 10 archivos por taller"})
		return
	}
	uploads := make([]auth.FileUpload, 0, len(headers))
	for _, fh := range headers {
		if fh.Size > maxAssignmentFileSize {
			writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "«" + fh.Filename + "» supera el máximo de 10 MB"})
			return
		}
		f, err := fh.Open()
		if err != nil {
			writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "no se pudo leer «" + fh.Filename + "»"})
			return
		}
		defer f.Close()
		uploads = append(uploads, auth.FileUpload{Name: fh.Filename, ContentType: fh.Header.Get("Content-Type"), Body: f})
	}

	user := auth.UserFrom(r.Context())
	a, err := h.store.CreateAssignment(user.ID, r.PathValue("id"), title, statement, rubric, dueAt, uploads)
	if err != nil {
		writeStoreErr(w, err, "crear el taller")
		return
	}
	slog.Info("assignment created", "user_id", user.ID, "group_id", a.GroupID, "assignment_id", a.ID, "files", len(a.Files))
	writeJSON(w, http.StatusCreated, a)
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
