package handler

import (
	"net/http"

	"github.com/marlon/edu-trace/internal/ai"
	"github.com/marlon/edu-trace/internal/auth"
)

// Registros (intentos + entregas) más recientes que se envían al modelo.
const analysisMaxEntries = 20

// AnalysisHandler genera para el profesor un análisis del proceso de un estudiante.
type AnalysisHandler struct {
	ollama    *ai.OllamaClient
	modelName string
	store     *auth.Store
}

func NewAnalysisHandler(ollama *ai.OllamaClient, modelName string, store *auth.Store) *AnalysisHandler {
	return &AnalysisHandler{ollama: ollama, modelName: modelName, store: store}
}

// ServeHTTP maneja POST /api/teacher/groups/{id}/students/{student}/analysis (SSE).
func (h *AnalysisHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	teacher := auth.UserFrom(r.Context())
	groupID := r.PathValue("id")

	// Solo clases propias y estudiantes inscritos en ellas.
	_, subs, err := h.store.StudentProcess(teacher.ID, groupID, r.PathValue("student"), analysisMaxEntries)
	if err != nil {
		writeStoreErr(w, err, "cargar el proceso del estudiante")
		return
	}
	assignments, err := h.store.GroupAssignments(teacher.ID, groupID)
	if err != nil {
		writeStoreErr(w, err, "cargar los talleres")
		return
	}
	titles := map[string]string{}
	for _, a := range assignments {
		titles[a.ID] = a.Title
	}

	// El nombre del estudiante no se envía al modelo: no hace falta para el análisis.
	entries := make([]ai.ProcessEntry, 0, len(subs))
	for _, s := range subs {
		entries = append(entries, ai.ProcessEntry{
			When:            s.CreatedAt,
			Official:        !s.IsAttempt(),
			AssignmentTitle: titles[s.AssignmentID],
			Success:         s.Success,
			RunError:        s.RunError,
			ExitCode:        s.ExitCode,
			CompilerOutput:  s.CompilerOutput,
			Stdin:           s.Stdin,
			ProgramOutput:   s.ProgramOutput,
			Code:            s.Code,
		})
	}

	streamSSE(w, h.modelName, func(onToken func(string)) error {
		return h.ollama.StreamAnalysis(r.Context(), entries, onToken)
	})
}
