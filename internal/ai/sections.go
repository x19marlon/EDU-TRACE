package ai

import (
	"fmt"
	"strings"

	"github.com/marlon/edu-trace/internal/model"
)

var sectionTitles = [3]string{"¿Hacia dónde voy?", "¿Cómo voy?", "¿Qué sigue?"}

var accentFolder = strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n")

func headingKey(line string) string {
	line = accentFolder.Replace(strings.ToLower(strings.TrimSpace(line)))
	line = strings.TrimLeft(line, "#*-_> 0123456789.)")
	var b strings.Builder
	for _, r := range line {
		if r >= 'a' && r <= 'z' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

var sectionKeys = func() [3]string {
	var keys [3]string
	for i, t := range sectionTitles {
		keys[i] = headingKey(t)
	}
	return keys
}()

func findSections(lines []string) [3]int {
	pos := [3]int{-1, -1, -1}
	for i, line := range lines {
		if len(line) > 60 {
			continue
		}
		key := headingKey(line)
		for j, k := range sectionKeys {
			if key == k && pos[j] < 0 {
				pos[j] = i
			}
		}
	}
	return pos
}

func MissingSections(text string) []string {
	var missing []string
	for i, p := range findSections(strings.Split(text, "\n")) {
		if p < 0 {
			missing = append(missing, sectionTitles[i])
		}
	}
	return missing
}

func EnsureSections(text string, fallback [3]string) (string, bool) {
	text = strings.TrimRight(text, "\n ")
	lines := strings.Split(text, "\n")
	pos := findSections(lines)
	if pos[0] >= 0 && pos[1] >= 0 && pos[2] >= 0 {
		return text, true
	}
	block := func(i int) []string { return []string{"## " + sectionTitles[i], fallback[i], ""} }
	if pos[0] < 0 && pos[1] < 0 && pos[2] < 0 {
		out := append(block(0), "## "+sectionTitles[1])
		out = append(out, lines...)
		out = append(out, "")
		out = append(out, block(2)...)
		return strings.TrimRight(strings.Join(out, "\n"), "\n"), false
	}
	insertAt := map[int][]string{}
	var tail []string
	for i := range pos {
		if pos[i] >= 0 {
			continue
		}
		next := -1
		for j := i + 1; j < len(pos); j++ {
			if pos[j] >= 0 {
				next = pos[j]
				break
			}
		}
		if next < 0 {
			tail = append(tail, append([]string{""}, block(i)...)...)
		} else {
			insertAt[next] = append(insertAt[next], block(i)...)
		}
	}
	var out []string
	for i, line := range lines {
		out = append(out, insertAt[i]...)
		out = append(out, line)
	}
	out = append(out, tail...)
	return strings.TrimRight(strings.Join(out, "\n"), "\n"), false
}

type Outcome struct {
	Statement bool
	Compiled bool
	Success bool
	RunFailed bool
	TestsPassed int
	TestsTotal int
}

func OutcomeOf(fb model.FeedbackRequest, statement string) Outcome {
	o := Outcome{Statement: strings.TrimSpace(statement) != "", Compiled: fb.Compiled, Success: fb.Success,
		RunFailed: fb.RunError != "" || fb.ExitCode != 0}
	o.TestsPassed, o.TestsTotal = countPassed(fb.Tests)
	return o
}

func countPassed(results []model.TestResult) (int, int) {
	passed := 0
	for _, r := range results {
		if r.Passed {
			passed++
		}
	}
	return passed, len(results)
}

func FallbackSections(o Outcome) [3]string {
	var f [3]string
	if o.Statement {
		f[0] = "Tu meta es que el programa haga lo que pide el enunciado del taller. Este trabajo desarrolla el RAE 4: implementar en C++."
	} else {
		f[0] = "Tu meta es que el programa resuelva correctamente el problema que te propusiste. Este trabajo desarrolla el RAE 4: implementar en C++."
	}
	testsFailing := o.TestsTotal > 0 && o.TestsPassed < o.TestsTotal
	switch {
	case !o.Compiled:
		f[1] = "Todavía no compilaste esta versión, así que aún no sabemos si tiene errores."
		f[2] = "Compila tu código y lee con calma el primer mensaje que aparezca antes de cambiar otra cosa."
	case !o.Success:
		f[1] = "Tu código todavía no compila: el compilador encontró al menos un error."
		f[2] = "Lee el primer error del compilador, ubica la línea que indica y pregúntate qué falta o sobra en ella; luego vuelve a compilar."
	case o.RunFailed:
		f[1] = "Tu programa compila, pero la ejecución terminó con un problema."
		f[2] = "Ejecuta tu programa con una entrada pequeña y muestra con cout el valor de tus variables en cada paso para ver dónde deja de pasar lo que esperas."
	case testsFailing:
		f[1] = fmt.Sprintf("Tu programa compila y pasa %d de %d casos de prueba del profesor.", o.TestsPassed, o.TestsTotal)
		f[2] = "Ejecuta tu programa con la entrada del caso que no pasa y compara su salida, línea por línea, con la salida esperada: ¿en qué punto empiezan a diferir?"
	default:
		f[1] = "Tu programa compila y se ejecuta."
		if o.TestsTotal > 0 {
			f[1] = fmt.Sprintf("Tu programa compila y pasa los %d casos de prueba del profesor.", o.TestsTotal)
		}
		f[2] = "Prueba tu programa con casos borde (por ejemplo, cero, un número negativo o el valor más grande permitido) y comprueba que la salida siga siendo correcta."
	}
	return f
}

type SectionTracker struct {
	out func(string)
	b strings.Builder
}

func NewSectionTracker(out func(string)) *SectionTracker { return &SectionTracker{out: out} }

func (t *SectionTracker) Write(s string) {
	t.b.WriteString(s)
	t.out(s)
}

func (t *SectionTracker) Text() string { return t.b.String() }

const maxTestDetails = 2

func writeTests(b *strings.Builder, results []model.TestResult) {
	if len(results) == 0 {
		return
	}
	passed, total := countPassed(results)
	fmt.Fprintf(b, "Casos de prueba del profesor: pasa %d de %d.\n", passed, total)
	shown := 0
	for i, r := range results {
		if r.Passed {
			continue
		}
		name := r.Name
		if name == "" {
			name = fmt.Sprintf("Caso %d", i+1)
		}
		if r.Hidden {
			fmt.Fprintf(b, "- %s (oculto para el estudiante): no pasa. No reveles nada de este caso.\n", name)
			continue
		}
		if shown >= maxTestDetails {
			fmt.Fprintf(b, "- %s: no pasa.\n", name)
			continue
		}
		shown++
		fmt.Fprintf(b, "- %s: no pasa. Entrada: %q. Salida esperada: %q. Salida obtenida: %q.", name,
			truncate(r.Stdin, 200), truncate(r.Expected, 200), truncate(r.Output, 200))
		if r.Error != "" {
			fmt.Fprintf(b, " Problema: %s.", r.Error)
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
}
