package handler

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/marlon/edu-trace/internal/auth"
	"github.com/marlon/edu-trace/internal/model"
)

type raeRequest struct {
	Code string `json:"code"`
	Description string `json:"description"`
}

func validRAE(w http.ResponseWriter, r *http.Request) (*raeRequest, bool) {
	var req raeRequest
	if !decodeJSON(w, r, &req) {
		return nil, false
	}
	req.Code, req.Description = strings.TrimSpace(req.Code), strings.TrimSpace(req.Description)
	if n := utf8.RuneCountInString(req.Code); n < 1 || n > 20 {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "el código del RAE debe tener entre 1 y 20 caracteres (p. ej. RAE 1)"})
		return nil, false
	}
	if n := utf8.RuneCountInString(req.Description); n < 3 || n > 500 {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "la descripción del RAE debe tener entre 3 y 500 caracteres"})
		return nil, false
	}
	return &req, true
}

func (h *ClassesHandler) CourseRAEs(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.CourseRAEs(auth.UserFrom(r.Context()).ID, r.PathValue("id"))
	if err != nil {
		writeStoreErr(w, err, "cargar los RAE")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *ClassesHandler) CreateRAE(w http.ResponseWriter, r *http.Request) {
	req, ok := validRAE(w, r)
	if !ok {
		return
	}
	rae, err := h.store.CreateRAE(auth.UserFrom(r.Context()).ID, r.PathValue("id"), req.Code, req.Description)
	if err != nil {
		writeStoreErr(w, err, "crear el RAE")
		return
	}
	writeJSON(w, http.StatusCreated, rae)
}

func (h *ClassesHandler) UpdateRAE(w http.ResponseWriter, r *http.Request) {
	req, ok := validRAE(w, r)
	if !ok {
		return
	}
	if err := h.store.UpdateRAE(auth.UserFrom(r.Context()).ID, r.PathValue("id"), req.Code, req.Description); err != nil {
		writeStoreErr(w, err, "guardar el RAE")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"id": r.PathValue("id"), "code": req.Code, "description": req.Description})
}

func (h *ClassesHandler) DeleteRAE(w http.ResponseWriter, r *http.Request) {
	if err := h.store.DeleteRAE(auth.UserFrom(r.Context()).ID, r.PathValue("id")); err != nil {
		writeStoreErr(w, err, "eliminar el RAE")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ClassesHandler) RAEMatrix(w http.ResponseWriter, r *http.Request) {
	m, err := h.store.GroupRAEMatrix(auth.UserFrom(r.Context()).ID, r.PathValue("id"))
	if err != nil {
		writeStoreErr(w, err, "cargar el rendimiento por RAE")
		return
	}
	writeJSON(w, http.StatusOK, m)
}
