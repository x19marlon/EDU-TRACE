package ai

import (
	"strings"
	"testing"

	"github.com/marlon/edu-trace/internal/model"
)

func TestEnsureSections(t *testing.T) {
	fb := FallbackSections(Outcome{Compiled: true, Success: true, TestsPassed: 1, TestsTotal: 3})
	complete := "## ¿Hacia dónde voy?\nMeta.\n\n## ¿Cómo voy?\nBien.\n\n## ¿Qué sigue?\nPrueba."
	if got, ok := EnsureSections(complete, fb); !ok || got != complete {
		t.Fatalf("texto completo modificado: %v %q", ok, got)
	}
	variants := "**¿Hacia donde voy?**\nMeta.\n### 2. ¿Como voy?\nBien.\n¿QUÉ SIGUE?:\nPrueba."
	if _, ok := EnsureSections(variants, fb); !ok {
		t.Fatal("no reconoce títulos con negrita, numeración o sin tildes")
	}

	noNext := "## ¿Hacia dónde voy?\nMeta.\n\n## ¿Cómo voy?\nBien."
	got, ok := EnsureSections(noNext, fb)
	if ok || !strings.HasSuffix(got, "## ¿Qué sigue?\n"+fb[2]) || !strings.Contains(fb[1], "1 de 3") {
		t.Fatalf("debe agregar ¿Qué sigue? al final: %q", got)
	}
	if len(MissingSections(got)) != 0 {
		t.Fatalf("siguen faltando secciones: %v", MissingSections(got))
	}

	noMiddle := "## ¿Hacia dónde voy?\nMeta.\n## ¿Qué sigue?\nPrueba."
	got, _ = EnsureSections(noMiddle, fb)
	if strings.Index(got, "¿Cómo voy?") > strings.Index(got, "¿Qué sigue?") || strings.Index(got, "¿Cómo voy?") < 0 {
		t.Fatalf("¿Cómo voy? debe quedar antes de ¿Qué sigue?: %q", got)
	}

	plain := "Tu programa suma mal en la línea 7."
	got, ok = EnsureSections(plain, fb)
	lines := strings.Split(got, "\n")
	if ok || lines[0] != "## ¿Hacia dónde voy?" || !strings.Contains(got, "## ¿Cómo voy?\n"+plain) || len(MissingSections(got)) != 0 {
		t.Fatalf("texto sin secciones: %q", got)
	}
}

func TestFallbackByOutcome(t *testing.T) {
	cases := []struct {
		o Outcome
		want string
	}{
		{Outcome{}, "Compila tu código"},
		{Outcome{Compiled: true}, "primer error del compilador"},
		{Outcome{Compiled: true, Success: true, RunFailed: true}, "valor de tus variables"},
		{Outcome{Compiled: true, Success: true, TestsPassed: 2, TestsTotal: 3}, "caso que no pasa"},
		{Outcome{Compiled: true, Success: true}, "casos borde"},
	}
	for _, c := range cases {
		if f := FallbackSections(c.o); !strings.Contains(f[2], c.want) || f[0] == "" || f[1] == "" {
			t.Errorf("%+v: %q", c.o, f)
		}
	}
}

func TestWriteTestsHidesHiddenCases(t *testing.T) {
	var b strings.Builder
	writeTests(&b, []model.TestResult{
		{Name: "Básico", Passed: true},
		{Name: "Cero", Stdin: "0", Expected: "Suma: 0", Output: "Suma: 1"},
		{Name: "Secreto", Hidden: true, Stdin: "99", Expected: "Suma: 4950"},
	})
	out := b.String()
	if !strings.Contains(out, "pasa 1 de 3") || !strings.Contains(out, `"Suma: 1"`) || strings.Contains(out, "4950") || strings.Contains(out, `"99"`) {
		t.Fatalf("resumen de casos: %q", out)
	}
}
