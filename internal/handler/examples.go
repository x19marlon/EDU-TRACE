package handler

import (
	"net/http"

	"github.com/marlon/edu-trace/internal/auth"
	"github.com/marlon/edu-trace/internal/examples"
	"github.com/marlon/edu-trace/internal/model"
)

type groupExampleView struct {
	examples.Example
	Enabled bool `json:"enabled"`
}

func (h *ClassesHandler) writeGroupExamples(w http.ResponseWriter, teacherID, groupID string) {
	ids, err := h.store.GroupExampleIDs(teacherID, groupID)
	if err != nil {
		writeStoreErr(w, err, "cargar los ejemplos")
		return
	}
	enabled := make(map[string]bool, len(ids))
	for _, id := range ids {
		enabled[id] = true
	}
	out := []groupExampleView{}
	for _, e := range examples.All() {
		out = append(out, groupExampleView{Example: e, Enabled: enabled[e.ID]})
	}
	writeJSON(w, http.StatusOK, out)
}

// GroupExamples maneja GET /api/teacher/groups/{id}/examples: el catálogo completo,
// con los ejemplos activos en la clase marcados.
func (h *ClassesHandler) GroupExamples(w http.ResponseWriter, r *http.Request) {
	h.writeGroupExamples(w, auth.UserFrom(r.Context()).ID, r.PathValue("id"))
}

// SetGroupExamples maneja PATCH /api/teacher/groups/{id}/examples: agrega o quita
// uno o varios ejemplos de la clase.
func (h *ClassesHandler) SetGroupExamples(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs     []string `json:"ids"`
		Enabled bool     `json:"enabled"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if len(req.IDs) == 0 {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "elige al menos un ejemplo"})
		return
	}
	for _, id := range req.IDs {
		if !examples.Exists(id) {
			writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "ese ejemplo no existe"})
			return
		}
	}
	teacher := auth.UserFrom(r.Context())
	if err := h.store.SetGroupExamples(teacher.ID, r.PathValue("id"), req.IDs, req.Enabled); err != nil {
		writeStoreErr(w, err, "cambiar los ejemplos")
		return
	}
	h.writeGroupExamples(w, teacher.ID, r.PathValue("id"))
}

type learnerGroupView struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	CourseName string             `json:"course_name"`
	Examples   []examples.Example `json:"examples"`
}

// MyExamples maneja GET /api/student/examples: los ejemplos que los docentes dejaron
// en las clases del estudiante.
func (h *ClassesHandler) MyExamples(w http.ResponseWriter, r *http.Request) {
	groups, err := h.store.LearnerExamples(auth.UserFrom(r.Context()).ID)
	if err != nil {
		writeStoreErr(w, err, "cargar los ejemplos")
		return
	}
	out := make([]learnerGroupView, len(groups))
	for i, g := range groups {
		out[i] = learnerGroupView{ID: g.ID, Name: g.Name, CourseName: g.CourseName, Examples: examples.Filter(g.ExampleIDs)}
	}
	writeJSON(w, http.StatusOK, out)
}
