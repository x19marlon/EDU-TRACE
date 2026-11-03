package ai

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/marlon/edu-trace/internal/model"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func testSyllabus(t *testing.T) *Syllabus {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "NO-ENVIAR")
	writeFile(t, filepath.Join(dir, "comun", "general.md"), "DOC-COMUN")
	writeFile(t, filepath.Join(dir, "intro", "materia.json"),
		`{"nombre": "Introducción a la Programación", "alias": ["Intro Programación"], "por_defecto": true}`)
	writeFile(t, filepath.Join(dir, "intro", "programa.md"), "DOC-INTRO")
	writeFile(t, filepath.Join(dir, "avanzada", "materia.json"), `{"nombre": "Programación Avanzada"}`)
	writeFile(t, filepath.Join(dir, "avanzada", "programa.md"), "DOC-AVANZADA")

	s, err := LoadSyllabus(dir)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestSyllabusFor(t *testing.T) {
	s := testSyllabus(t)
	tests := []struct {
		course, subject string
		want, notWant   []string
	}{
		{"Introducción a la Programación 2026-1", "Introducción a la Programación", []string{"DOC-COMUN", "DOC-INTRO"}, []string{"DOC-AVANZADA"}},
		{"INTRO PROGRAMACION - grupo a", "Introducción a la Programación", []string{"DOC-INTRO"}, []string{"DOC-AVANZADA"}},
		{"Programación Avanzada 2026-1", "Programación Avanzada", []string{"DOC-COMUN", "DOC-AVANZADA"}, []string{"DOC-INTRO"}},
		{"", "Introducción a la Programación", []string{"DOC-INTRO"}, []string{"DOC-AVANZADA"}},
		{"Bases de Datos", "", []string{"DOC-COMUN"}, []string{"DOC-INTRO", "DOC-AVANZADA"}},
	}
	for _, tt := range tests {
		docs, subject := s.For(tt.course)
		if subject != tt.subject {
			t.Errorf("For(%q) subject = %q, want %q", tt.course, subject, tt.subject)
		}
		for _, w := range tt.want {
			if !strings.Contains(docs, w) {
				t.Errorf("For(%q) missing %s", tt.course, w)
			}
		}
		for _, nw := range append(tt.notWant, "NO-ENVIAR") {
			if strings.Contains(docs, nw) {
				t.Errorf("For(%q) should not include %s", tt.course, nw)
			}
		}
	}
}

func TestSyllabusForWithoutDocs(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "intro", "materia.json"), `{"nombre": "Intro"}`)
	writeFile(t, filepath.Join(dir, "intro", "programa.md"), "DOC-INTRO")
	s, err := LoadSyllabus(dir)
	if err != nil {
		t.Fatal(err)
	}
	if docs, _ := s.For("Bases de Datos"); docs != noDocsText {
		t.Errorf("unmatched course without comun/ = %q, want the no-docs notice", docs)
	}
}

func TestLoadSyllabusErrors(t *testing.T) {
	t.Run("missing materia.json", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "intro", "programa.md"), "x")
		if _, err := LoadSyllabus(dir); err == nil {
			t.Error("expected error")
		}
	})
	t.Run("two defaults", func(t *testing.T) {
		dir := t.TempDir()
		for _, name := range []string{"a", "b"} {
			writeFile(t, filepath.Join(dir, name, "materia.json"), `{"nombre": "`+name+`", "por_defecto": true}`)
			writeFile(t, filepath.Join(dir, name, "doc.md"), "x")
		}
		if _, err := LoadSyllabus(dir); err == nil {
			t.Error("expected error")
		}
	})
}

func TestRepoSyllabusLoads(t *testing.T) {
	s, err := LoadSyllabus(filepath.Join("..", "..", "syllabus"))
	if err != nil {
		t.Fatal(err)
	}
	if _, subject := s.For("Introducción a la Programación 2026-1"); subject != "Introducción a la Programación" {
		t.Errorf("seed course matched %q", subject)
	}
	if _, subject := s.For("Programación Avanzada 2026-1"); subject != "" {
		t.Errorf("advanced course should not match a subject yet, got %q", subject)
	}
}

func TestMessagesUseTheDocsOfTheCourse(t *testing.T) {
	s := testSyllabus(t)
	o := &OllamaClient{syllabus: s}
	system := func(course string) string {
		return o.BuildMessages(model.FeedbackRequest{Mode: ModeFormal, Code: "int main(){}", Course: course}, "", "", nil)[0].Content
	}
	if got := system("Introducción a la Programación 2026-1"); !strings.Contains(got, "DOC-INTRO") || strings.Contains(got, "DOC-AVANZADA") {
		t.Errorf("curso de Intro: %q", got)
	}
	if got := system("Programación Avanzada 2026-1"); !strings.Contains(got, "DOC-AVANZADA") || strings.Contains(got, "DOC-INTRO") {
		t.Errorf("curso de Avanzada: %q", got)
	}

	// El curso lo resuelve el servidor: el cliente no puede elegir otros documentos.
	var req model.FeedbackRequest
	if err := json.Unmarshal([]byte(`{"course":"Programación Avanzada","Course":"Programación Avanzada"}`), &req); err != nil {
		t.Fatal(err)
	}
	if req.Course != "" {
		t.Errorf("el cliente fijó el curso: %q", req.Course)
	}
}
