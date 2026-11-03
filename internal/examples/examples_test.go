package examples

import (
	"context"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/marlon/edu-trace/internal/compiler"
	"github.com/marlon/edu-trace/internal/config"
	"github.com/marlon/edu-trace/internal/model"
)

func TestCatalogCoversCourseTopics(t *testing.T) {
	topics := []string{}
	seen := map[string]bool{}
	for _, e := range All() {
		if len(topics) == 0 || topics[len(topics)-1] != e.Topic {
			if seen[e.Topic] {
				t.Errorf("el tema %q no está agrupado", e.Topic)
			}
			seen[e.Topic] = true
			topics = append(topics, e.Topic)
		}
		if e.ID == "" || e.Title == "" || e.Explanation == "" || e.Code == "" || e.Output == "" {
			t.Errorf("%q: ejemplo incompleto", e.ID)
		}
	}
	want := "Estructura y variables,Palabras reservadas,Operadores,Condicionales,Ciclos,Funciones,Arreglos,Matrices,Estructuras (struct),Archivos de texto,Librería cmath"
	if got := strings.Join(topics, ","); got != want {
		t.Fatalf("temas: %s", got)
	}
}

// El programa corre en un sandbox donde solo /tmp admite escritura, y el curso no ve
// punteros, memoria dinámica, clases, plantillas ni excepciones.
func TestExamplesStayInsideCourseAndSandbox(t *testing.T) {
	outside := regexp.MustCompile(`\b(class|new|delete|template|try|catch|throw|auto|nullptr|vector|map)\b|\bint\s*\*`)
	file := regexp.MustCompile(`(?:of|if|f)stream\s+\w+\("([^"]*)"`)
	for _, e := range All() {
		code := regexp.MustCompile(`//.*`).ReplaceAllString(e.Code, "")
		if m := outside.FindString(code); m != "" {
			t.Errorf("%s: usa %q, que está fuera del curso", e.ID, m)
		}
		for _, m := range file.FindAllStringSubmatch(e.Code, -1) {
			if !strings.HasPrefix(m[1], "/tmp/") {
				t.Errorf("%s: abre %q; en el sandbox solo se puede escribir en /tmp", e.ID, m[1])
			}
		}
	}
}

func TestExamplesCompileAndPrintTheirOutput(t *testing.T) {
	if _, err := exec.LookPath("g++"); err != nil {
		t.Skip("g++ no está instalado")
	}
	c := compiler.New(&config.Config{GppPath: "g++", Sandbox: "none", CompileTimeout: 30 * time.Second, RunTimeout: 3 * time.Second, MaxOutputSize: 10240})
	for _, e := range All() {
		res := c.CompileAndTest(context.Background(), e.Code, e.Stdin, []model.TestCase{{
			ID: e.ID, Name: e.Title, Stdin: e.Stdin, Expected: e.Output, Match: model.MatchExact,
		}})
		if !res.Success {
			t.Errorf("%s no compila: %s", e.ID, res.CompilerOutput)
			continue
		}
		if strings.TrimSpace(res.CompilerOutput) != "" {
			t.Errorf("%s compila con avisos: %s", e.ID, res.CompilerOutput)
		}
		for _, r := range res.Tests {
			if !r.Passed {
				t.Errorf("%s: la salida no coincide (salida %q, esperada %q, error %q)", e.ID, r.Output, r.Expected, r.Error)
			}
		}
	}
}

func TestFilterKeepsCatalogOrderAndIgnoresUnknown(t *testing.T) {
	got := Filter([]string{"cmath-distancia", "no-existe", "ciclo-for"})
	if len(got) != 2 || got[0].ID != "ciclo-for" || got[1].ID != "cmath-distancia" {
		t.Fatalf("Filter = %+v", got)
	}
	if Exists("no-existe") || !Exists("ciclo-for") {
		t.Fatal("Exists no coincide con el catálogo")
	}
}
