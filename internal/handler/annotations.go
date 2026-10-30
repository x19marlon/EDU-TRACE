package handler

import (
	"errors"
	"net/http"

	"github.com/marlon/edu-trace/internal/auth"
	"github.com/marlon/edu-trace/internal/model"
)

type AnnotationsHandler struct {
	store *auth.Store
}

func NewAnnotationsHandler(store *auth.Store) *AnnotationsHandler {
	return &AnnotationsHandler{store: store}
}

func (h *AnnotationsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in auth.AnnotationInput
	if !decodeJSONLimit(w, r, &in, 512*1024) {
		return
	}
	a, err := h.store.AddAnnotation(auth.UserFrom(r.Context()), r.PathValue("id"), in)
	if errors.Is(err, auth.ErrInvalidAnnotation) {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "la anotación no es válida: revisa las líneas, el comentario o el dibujo"})
		return
	}
	if err != nil {
		writeStoreErr(w, err, "guardar la anotación")
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

func (h *AnnotationsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.store.DeleteAnnotation(auth.UserFrom(r.Context()), r.PathValue("id")); err != nil {
		writeStoreErr(w, err, "eliminar la anotación")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AnnotationsHandler) Send(w http.ResponseWriter, r *http.Request) {
	n, at, err := h.store.SendMonitorAnnotations(auth.UserFrom(r.Context()), r.PathValue("id"))
	if err != nil {
		writeStoreErr(w, err, "enviar las anotaciones")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sent": n, "sent_at": at})
}
