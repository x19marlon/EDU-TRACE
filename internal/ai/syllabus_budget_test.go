package ai

import (
	"fmt"
	"path/filepath"
	"testing"
)

// Los documentos van en cada petición: si crecen, le quitan espacio al código del
// estudiante en el contexto del modelo (OLLAMA_CONTEXT_SIZE, 6144 por defecto).
func TestCourseDocsLeaveRoomForStudentCode(t *testing.T) {
	s, err := LoadSyllabus(filepath.Join("..", "..", "syllabus"))
	if err != nil {
		t.Fatal(err)
	}
	docs, _ := s.For("Introducción a la Programación 2026-1")
	const contextSize, minForStudent = 6144, 2000
	for name, prompt := range map[string]string{"formal": systemPromptFormal, "informal": systemPromptInformal, "chat": systemPromptChat, "revisión": systemPromptReview} {
		system := estimateTokens([]chatMessage{{Content: fmt.Sprintf(prompt, docs)}})
		if free := contextSize - reservedTokens - system; free < minForStudent {
			t.Errorf("%s: quedan %d tokens para el estudiante; se necesitan al menos %d", name, free, minForStudent)
		}
	}
}
