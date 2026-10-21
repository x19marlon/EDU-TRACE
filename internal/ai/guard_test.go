package ai

import (
	"strings"
	"testing"
)

const guardStudentCode = `#include <iostream>
using namespace std;

int main() {
    int a, b, c;
    cin >> a >> b >> c;
    float promedio = a + b + c / 3;
    int suma;
    for (int i = 1; i < 10; i++) {
        suma = suma + i;
    }
    cout << "Promedio: " << promedio << endl;
    return 0;
}`

// guardRun filtra text llegando en trozos de size bytes, como en el streaming.
func guardRun(text string, size int) (string, int) {
	var b strings.Builder
	g := NewSolutionGuard(func(s string) { b.WriteString(s) }, guardStudentCode, "Calcula el promedio de tres notas")
	for len(text) > 0 {
		n := min(size, len(text))
		g.Write(text[:n])
		text = text[n:]
	}
	g.Close()
	return b.String(), g.Omitted()
}

func TestSolutionGuard(t *testing.T) {
	cases := []struct {
		name string
		in string
		omitted int
		keep []string 
		drop []string 
	}{
		{
			name: "texto con títulos y viñetas pasa igual",
			in: "## ¿Cómo voy?\n- Tu programa compila.\n1. Revisa la línea 7.\nOjo - la variable suma no empieza en cero.\n¿Qué valor tiene `suma` al empezar?\n",
			keep: []string{"## ¿Cómo voy?", "- Tu programa compila.", "1. Revisa la línea 7.", "Ojo - la variable", "`suma`"},
		},
		{
			name: "línea corregida en un bloque",
			in: "Prueba así:\n```cpp\nfloat promedio = (a + b + c) / 3;\n```\nY compila de nuevo.\n",
			omitted: 1,
			keep: []string{"Prueba así:", omittedBlock, "Y compila de nuevo."},
			drop: []string{"(a + b + c) / 3"},
		},
		{
			name: "programa completo con nombres nuevos",
			in: "```cpp\n#include <iostream>\nint main() {\n    std::cout << 1;\n}\n```\n",
			omitted: 1,
			drop: []string{"int main"},
		},
		{
			name: "cita textual de su código",
			in: "Mira estas líneas:\n```cpp\nfor (int i = 1; i < 10; i++) {\n    suma = suma + i;\n}\n```\n",
			keep: []string{"suma = suma + i;"},
		},
		{
			name: "cita con números de línea",
			in: "```\n7 | float promedio = a + b + c / 3;\n```\n",
			keep: []string{"7 | float promedio"},
		},
		{
			name: "corrección con números de línea fuera de bloque",
			in: "Cambia esto:\n7 | float promedio = (a + b + c) / 3;\nListo.\n",
			omitted: 1,
			drop: []string{"(a + b + c)"},
			keep: []string{"Listo."},
		},
		{
			name: "ejemplo genérico corto",
			in: "Un ciclo se ve así:\n```cpp\nfor (int k = 0; k < 3; k++) {\n    cout << k;\n}\n```\n",
			keep: []string{"for (int k = 0; k < 3; k++) {"},
		},
		{
			name: "bloque genérico largo",
			in: "```cpp\nint x = 4;\nint y = 5;\nint z = x * y;\ncout << z;\ncout << x;\n```\n",
			omitted: 1,
			drop: []string{"int z"},
		},
		{
			name: "fragmento en línea que corrige",
			in: "En la línea 9 cambia la condición a `i <= 10`, o mejor `(a + b + c) / 3` y `int suma = 0`.\n",
			omitted: 3,
			keep: []string{"En la línea 9 cambia la condición a " + omittedInline},
			drop: []string{"i <= 10", "(a + b + c)", "suma = 0"},
		},
		{
			name: "fragmentos en línea permitidos",
			in: "Usa `%` para el residuo; la variable `promedio` y `i < 10` son tuyas; prueba `k++`.\n",
			keep: []string{"`%`", "`promedio`", "`i < 10`", "`k++`"},
		},
		{
			name: "código suelto con sangría",
			in: "Haz esto:\n\n    suma = 0;\n    promedio = (a + b + c) / 3.0;\n\nY listo.\n",
			omitted: 1,
			drop: []string{"3.0"},
			keep: []string{"Haz esto:", "Y listo."},
		},
		{
			name: "bloque sin cerrar al final",
			in: "Así:\n```cpp\nfloat promedio = (a + b + c) / 3;",
			omitted: 1,
			drop: []string{"(a + b + c)"},
		},
		{
			name: "palabra del enunciado en un ejemplo nuevo",
			in: "```cpp\nfloat notas = x / 3;\n```\n",
			omitted: 1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			whole, n := guardRun(tc.in, len(tc.in)+1)
			if n != tc.omitted {
				t.Errorf("omitidos = %d, quería %d\n%s", n, tc.omitted, whole)
			}
			for _, s := range tc.keep {
				if !strings.Contains(whole, s) {
					t.Errorf("falta %q en:\n%s", s, whole)
				}
			}
			for _, s := range tc.drop {
				if strings.Contains(whole, s) {
					t.Errorf("no se quitó %q:\n%s", s, whole)
				}
			}
			if tc.omitted == 0 && whole != tc.in {
				t.Errorf("cambió un texto que debía pasar igual:\n%q\n%q", tc.in, whole)
			}
			
			for _, size := range []int{1, 2, 3, 7, 16} {
				if got, _ := guardRun(tc.in, size); got != whole {
					t.Errorf("con trozos de %d bytes:\n%q\nquería\n%q", size, got, whole)
				}
			}
		})
	}
}

func TestFilterSolution(t *testing.T) {
	out, n := FilterSolution("Revisa:\n```cpp\nsuma = 0;\n```\n", "int suma;\nsuma = suma + 1;")
	if n != 1 || strings.Contains(out, "suma = 0") {
		t.Fatalf("no filtró: %d %q", n, out)
	}
}
