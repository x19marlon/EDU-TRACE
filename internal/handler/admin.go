package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/marlon/edu-trace/internal/auth"
	"github.com/marlon/edu-trace/internal/model"
)

func (h *ClassesHandler) RenameCourse(w http.ResponseWriter, r *http.Request) {
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
	if err := h.store.RenameCourse(auth.UserFrom(r.Context()).ID, r.PathValue("id"), name); err != nil {
		writeStoreErr(w, err, "renombrar la materia")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"id": r.PathValue("id"), "name": name})
}

func (h *ClassesHandler) DeleteCourse(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFrom(r.Context())
	if err := h.store.DeleteCourse(user.ID, r.PathValue("id")); err != nil {
		writeStoreErr(w, err, "eliminar la materia")
		return
	}
	slog.Info("course deleted", "user_id", user.ID, "course_id", r.PathValue("id"))
	w.WriteHeader(http.StatusNoContent)
}

func (h *ClassesHandler) RenameGroup(w http.ResponseWriter, r *http.Request) {
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
	if err := h.store.RenameGroup(auth.UserFrom(r.Context()).ID, r.PathValue("id"), name); err != nil {
		writeStoreErr(w, err, "renombrar la clase")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"id": r.PathValue("id"), "name": name})
}

func (h *ClassesHandler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFrom(r.Context())
	if err := h.store.DeleteGroup(user.ID, r.PathValue("id")); err != nil {
		writeStoreErr(w, err, "eliminar la clase")
		return
	}
	slog.Info("group deleted", "user_id", user.ID, "group_id", r.PathValue("id"))
	w.WriteHeader(http.StatusNoContent)
}

func (h *ClassesHandler) RegenerateJoinCode(w http.ResponseWriter, r *http.Request) {
	code, err := h.store.RegenerateJoinCode(auth.UserFrom(r.Context()).ID, r.PathValue("id"))
	if err != nil {
		writeStoreErr(w, err, "generar un código nuevo")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"join_code": code})
}

func (h *ClassesHandler) RemoveStudent(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFrom(r.Context())
	if err := h.store.RemoveStudent(user.ID, r.PathValue("id"), r.PathValue("student")); err != nil {
		writeStoreErr(w, err, "quitar al estudiante")
		return
	}
	slog.Info("student removed", "user_id", user.ID, "group_id", r.PathValue("id"), "student_id", r.PathValue("student"))
	w.WriteHeader(http.StatusNoContent)
}

func (h *ClassesHandler) IssueResetCode(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFrom(r.Context())
	code, expires, err := h.store.IssueResetCode(user.ID, r.PathValue("id"), r.PathValue("student"))
	if err != nil {
		writeStoreErr(w, err, "generar el código")
		return
	}
	slog.Info("reset code issued", "user_id", user.ID, "group_id", r.PathValue("id"), "student_id", r.PathValue("student"))
	writeJSON(w, http.StatusOK, map[string]any{"code": code, "expires_at": expires})
}

func (h *ClassesHandler) CloseCourse(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFrom(r.Context())
	purge, err := h.store.CloseCourse(user.ID, r.PathValue("id"))
	if err != nil {
		writeStoreErr(w, err, "cerrar el periodo")
		return
	}
	slog.Info("course closed", "user_id", user.ID, "course_id", r.PathValue("id"), "purge_at", purge)
	writeJSON(w, http.StatusOK, map[string]any{"purge_at": purge})
}

func (h *ClassesHandler) ReopenCourse(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFrom(r.Context())
	if err := h.store.ReopenCourse(user.ID, r.PathValue("id")); err != nil {
		writeStoreErr(w, err, "reabrir el periodo")
		return
	}
	slog.Info("course reopened", "user_id", user.ID, "course_id", r.PathValue("id"))
	w.WriteHeader(http.StatusNoContent)
}

func (h *ClassesHandler) AddMonitor(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	email := strings.TrimSpace(req.Email)
	if email == "" || len(email) > 254 {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "escribe el email del monitor"})
		return
	}
	m, err := h.store.AddMonitor(auth.UserFrom(r.Context()).ID, r.PathValue("id"), email)
	if errors.Is(err, auth.ErrMonitorNotFound) {
		writeJSON(w, http.StatusNotFound, model.ErrorResponse{Error: err.Error() + "; pídele que se registre como monitor"})
		return
	}
	if err != nil {
		writeStoreErr(w, err, "agregar al monitor")
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

func (h *ClassesHandler) RemoveMonitor(w http.ResponseWriter, r *http.Request) {
	if err := h.store.RemoveMonitor(auth.UserFrom(r.Context()).ID, r.PathValue("id"), r.PathValue("monitor")); err != nil {
		writeStoreErr(w, err, "quitar al monitor")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
