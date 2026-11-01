package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/marlon/edu-trace/internal/auth"
	"github.com/marlon/edu-trace/internal/model"
	"github.com/marlon/edu-trace/internal/practice"
	"github.com/marlon/edu-trace/internal/similarity"
)

const similarityThreshold = 0.5

func (h *ClassesHandler) AddCoTeacher(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	user := auth.UserFrom(r.Context())
	t, err := h.store.AddCoTeacher(user.ID, r.PathValue("id"), strings.TrimSpace(req.Email))
	switch {
	case errors.Is(err, auth.ErrTeacherNotFound):
		writeJSON(w, http.StatusNotFound, model.ErrorResponse{Error: err.Error()})
		return
	case errors.Is(err, auth.ErrAlreadyTeacher):
		writeJSON(w, http.StatusConflict, model.ErrorResponse{Error: err.Error()})
		return
	case err != nil:
		writeStoreErr(w, err, "agregar al profesor")
		return
	}
	slog.Info("co-teacher added", "user_id", user.ID, "course_id", r.PathValue("id"), "teacher_id", t.ID)
	writeJSON(w, http.StatusCreated, t)
}

func (h *ClassesHandler) RemoveCoTeacher(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFrom(r.Context())
	if err := h.store.RemoveCoTeacher(user.ID, r.PathValue("id"), r.PathValue("teacher")); err != nil {
		writeStoreErr(w, err, "retirar al profesor")
		return
	}
	slog.Info("co-teacher removed", "user_id", user.ID, "course_id", r.PathValue("id"), "teacher_id", r.PathValue("teacher"))
	w.WriteHeader(http.StatusNoContent)
}

func (h *ClassesHandler) CloneCourse(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		CloseOld bool `json:"close_old"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Name != "" {
		name, ok := validName(w, req.Name, "materia")
		if !ok {
			return
		}
		req.Name = name
	}
	user := auth.UserFrom(r.Context())
	id, err := h.store.CloneCourse(user.ID, r.PathValue("id"), req.Name, req.CloseOld)
	if err != nil && id == "" {
		writeStoreErr(w, err, "copiar la materia")
		return
	}
	if err != nil {
		slog.Error("cerrar la materia copiada", "course_id", r.PathValue("id"), "error", err)
	}
	slog.Info("course cloned", "user_id", user.ID, "from", r.PathValue("id"), "to", id)
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *ClassesHandler) StudentAlerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := h.store.StudentAlerts(auth.UserFrom(r.Context()).ID, r.PathValue("id"), time.Now())
	if err != nil {
		writeStoreErr(w, err, "calcular las alertas")
		return
	}
	writeJSON(w, http.StatusOK, alerts)
}

func (h *ClassesHandler) StudentReport(w http.ResponseWriter, r *http.Request) {
	rep, err := h.store.StudentReport(auth.UserFrom(r.Context()).ID, r.PathValue("id"), r.PathValue("student"), time.Now())
	if err != nil {
		writeStoreErr(w, err, "preparar el reporte")
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

type similarStudent struct {
	StudentID string `json:"student_id"`
	Name string `json:"name"`
	SubmissionID string `json:"submission_id"`
}

type similarPair struct {
	A similarStudent `json:"a"`
	B similarStudent `json:"b"`
	Score float64 `json:"score"`
}

func (h *ClassesHandler) Similarity(w http.ResponseWriter, r *http.Request) {
	assignment := r.URL.Query().Get("assignment")
	if assignment == "" {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "elige un taller"})
		return
	}
	docs, err := h.store.SimilarityInput(auth.UserFrom(r.Context()).ID, r.PathValue("id"), assignment)
	if err != nil {
		writeStoreErr(w, err, "comparar las entregas")
		return
	}
	byID := map[string]auth.SimilarityDoc{}
	codes := map[string]string{}
	for _, d := range docs {
		byID[d.SubmissionID] = d
		codes[d.SubmissionID] = d.Code
	}
	out := []similarPair{}
	for _, p := range similarity.Compare(codes, similarityThreshold) {
		a, b := byID[p.A], byID[p.B]
		out = append(out, similarPair{
			A: similarStudent{StudentID: a.StudentID, Name: a.StudentName, SubmissionID: a.SubmissionID},
			B: similarStudent{StudentID: b.StudentID, Name: b.StudentName, SubmissionID: b.SubmissionID},
			Score: p.Score,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"compared": len(docs), "pairs": out})
}

func (h *ClassesHandler) MyProgress(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFrom(r.Context())
	groups, err := h.store.StudentProgress(user.ID)
	if err != nil {
		writeStoreErr(w, err, "cargar tu progreso")
		return
	}
	results, err := h.store.PracticeResults(user.ID)
	if err != nil {
		writeStoreErr(w, err, "cargar tu práctica")
		return
	}
	solved := 0
	for _, res := range results {
		if res.SolvedAt != nil {
			solved++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"groups": groups, "practice": map[string]int{"solved": solved, "total": len(practice.All())}})
}

type exerciseView struct {
	practice.Exercise
	Result *auth.PracticeResult `json:"result,omitempty"`
}

func (h *ClassesHandler) PracticeList(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFrom(r.Context())
	results := map[string]auth.PracticeResult{}
	if user.Role == auth.RoleStudent {
		var err error
		if results, err = h.store.PracticeResults(user.ID); err != nil {
			writeStoreErr(w, err, "cargar tu práctica")
			return
		}
	}
	out := []exerciseView{}
	for _, e := range practice.All() {
		v := exerciseView{Exercise: e}
		if res, ok := results[e.ID]; ok {
			v.Result = &res
		}
		out = append(out, v)
	}
	writeJSON(w, http.StatusOK, out)
}

func ownTests(in []model.TestCase) ([]model.TestCase, error) {
	if len(in) == 0 {
		return nil, nil
	}
	tests, err := auth.NormalizeTests(in)
	if err != nil {
		return nil, err
	}
	for i := range tests {
		tests[i].ID, tests[i].Own, tests[i].Hidden = fmt.Sprintf("propio-%d", i+1), true, false
	}
	return tests, nil
}

func exerciseStatement(id string) string {
	if e, ok := practice.Get(id); ok {
		return e.Title + "\n" + e.Statement
	}
	return ""
}
