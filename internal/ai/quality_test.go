package ai

import (
	"strings"
	"testing"
)

func TestAnalyzeQuality(t *testing.T) {
	messy := `#include <iostream>
using namespace std;
int total;
int main() {
int a, b;
    cin >> a >> b;
    cout << "Resultado: " << a + b << endl;
    cout << "Resultado: " << a + b << endl;
    cout << "Resultado: " << a + b << endl;
    return 0;
}`
	q := AnalyzeQuality(messy)
	if !strings.Contains(q.Readability, "sangría de la línea 5") {
		t.Errorf("legibilidad: %q", q.Readability)
	}
	if !strings.Contains(q.Maintainability, "se repite 3 veces (las líneas 7, 8 y 9)") {
		t.Errorf("mantenibilidad: %q", q.Maintainability)
	}
	if !strings.Contains(q.Documentation, "no hay comentarios") {
		t.Errorf("documentación: %q", q.Documentation)
	}
	if len(q.Problems()) != 3 {
		t.Errorf("los tres son problemas concretos: %v", q.Problems())
	}

	clean := `#include <iostream>
using namespace std;
// Suma de 1 a n
int main() {
    int n, suma = 0;
    cin >> n;
    for (int i = 1; i <= n; i++) {
        suma += i;
    }
    cout << "Suma: " << suma << endl;
    return 0;
}`
	q = AnalyzeQuality(clean)
	if !strings.Contains(q.Readability, "consistente") || !strings.Contains(q.Maintainability, "fácil de modificar") || !strings.Contains(q.Documentation, "1 comentario;") {
		t.Errorf("código limpio: %+v", q)
	}
	if len(q.Problems()) != 0 {
		t.Errorf("el código limpio no tiene problemas que reportar al modelo: %v", q.Problems())
	}
	names := AnalyzeQuality("int main() {\n    double a, b, c;\n    int i;\n    return 0;\n}")
	if !strings.Contains(names.Readability, "«a», «b», «c»") || strings.Contains(names.Readability, "«i»") {
		t.Errorf("nombres cortos: %q", names.Readability)
	}
	global := AnalyzeQuality("#include <iostream>\nint contador = 0;\nconst int MAX = 5;\nint main() {\n    return 0;\n}")
	if !strings.Contains(global.Maintainability, "la línea 2 declara una variable global") {
		t.Errorf("variable global: %q", global.Maintainability)
	}
}

func TestEnsureQuality(t *testing.T) {
	q := QualityReport{Readability: "Legibilidad: bien.", Maintainability: "Mantenibilidad: bien.", Documentation: "Documentación: agrega comentarios."}
	full := "## ¿Cómo voy?\nTodo bien. La legibilidad es buena, la mantenibilidad también y falta documentación.\n## ¿Qué sigue?\nSigue."
	if got, ok := EnsureQuality(full, q); !ok || got != full {
		t.Fatalf("texto completo modificado: %q", got)
	}
	partial := "## ¿Hacia dónde voy?\nMeta.\n## ¿Cómo voy?\nTu código tiene buena legibilidad.\n## ¿Qué sigue?\nPrueba."
	got, ok := EnsureQuality(partial, q)
	if ok || strings.Contains(got, "Legibilidad: bien.") || !strings.Contains(got, "- Mantenibilidad: bien.") {
		t.Fatalf("solo debe agregar lo que falta: %q", got)
	}
	if strings.Index(got, "**Calidad del código**") > strings.Index(got, "## ¿Qué sigue?") {
		t.Fatalf("la calidad va dentro de ¿Cómo voy?, antes de ¿Qué sigue?: %q", got)
	}
	synonyms := "## ¿Cómo voy?\nLa sangría de la línea 4 está mal, la línea 6 se repite y faltan comentarios.\n## ¿Qué sigue?\nPrueba."
	if _, ok := EnsureQuality(synonyms, q); !ok {
		t.Fatal("sangría, repetición y comentarios cuentan como observaciones de calidad")
	}
	titled := "## ¿Cómo voy?\n**Calidad del código**: buena legibilidad.\n## ¿Qué sigue?\nPrueba."
	if got, _ := EnsureQuality(titled, q); strings.Count(got, "Calidad del código") != 1 {
		t.Fatalf("no debe repetir el título: %q", got)
	}
	fixed, checks := Finalize("Tu programa suma mal.", Outcome{Compiled: true, Success: true}, AnalyzeQuality("int main(){\nreturn 0;\n}"))
	if checks.SectionsOK || checks.QualityOK || len(MissingSections(fixed)) != 0 {
		t.Fatalf("Finalize: %+v %q", checks, fixed)
	}
	for _, k := range []string{"Legibilidad:", "Mantenibilidad:", "Documentación:"} {
		if !strings.Contains(fixed, k) {
			t.Errorf("falta %s en %q", k, fixed)
		}
	}
}
