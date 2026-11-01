package ai

import (
	"fmt"
	"regexp"
	"strings"
)

type QualityReport struct {
	Readability string
	Maintainability string
	Documentation string
	issues [3]bool
}

func (q QualityReport) Problems() []string {
	var out []string
	for i, line := range []string{q.Readability, q.Maintainability, q.Documentation} {
		if q.issues[i] {
			out = append(out, line)
		}
	}
	return out
}

var (
	declPattern = regexp.MustCompile(`^\s*(?:const\s+)?(?:unsigned\s+|long\s+)*(?:int|double|float|char|bool|string|long)\s+([^;()]+);`)
	identPattern = regexp.MustCompile(`^[A-Za-z_]\w*`)
	stringLiteral = regexp.MustCompile(`"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'`)
)

const (
	longLine = 100
	longMain = 50
	maxRefs = 3
)

func lineRefs(lines []int) string {
	if len(lines) > maxRefs {
		lines = lines[:maxRefs]
	}
	parts := make([]string, len(lines))
	for i, n := range lines {
		parts[i] = fmt.Sprint(n)
	}
	if len(parts) == 1 {
		return "la línea " + parts[0]
	}
	return "las líneas " + strings.Join(parts[:len(parts)-1], ", ") + " y " + parts[len(parts)-1]
}

func AnalyzeQuality(code string) QualityReport {
	lines := strings.Split(strings.ReplaceAll(strings.TrimRight(code, "\n"), "\r\n", "\n"), "\n")
	var unindented, long, shortNames, globals []int
	var shortList []string
	comments, depth, mainDepth, mainLines := 0, 0, -1, 0
	tabs, spaces, mainDone := false, false, false
	seen := map[string][]int{}
	for i, raw := range lines {
		n := i + 1
		if strings.Contains(raw, "//") || strings.Contains(raw, "/*") {
			comments++
		}
		stmt := stringLiteral.ReplaceAllString(strings.TrimSpace(raw), `""`)
		if idx := strings.Index(stmt, "//"); idx >= 0 {
			stmt = strings.TrimSpace(stmt[:idx])
		}
		indent := raw[:len(raw)-len(strings.TrimLeft(raw, " \t"))]
		tabs = tabs || strings.Contains(indent, "\t")
		spaces = spaces || strings.Contains(indent, " ")
		if stmt != "" && depth > 0 && indent == "" && !strings.HasPrefix(stmt, "}") && !strings.HasPrefix(stmt, "#") {
			unindented = append(unindented, n)
		}
		if len([]rune(raw)) > longLine {
			long = append(long, n)
		}
		if m := declPattern.FindStringSubmatch(stmt); m != nil {
			if depth == 0 && !strings.HasPrefix(stmt, "const") {
				globals = append(globals, n)
			}
			for _, part := range strings.Split(m[1], ",") {
				name := identPattern.FindString(strings.TrimSpace(part))
				if len(name) == 1 && !strings.Contains("ijkn", name) {
					shortNames = append(shortNames, n)
					shortList = append(shortList, name)
				}
			}
		}
		if len(stmt) >= 12 && stmt != "return 0;" && !strings.HasPrefix(stmt, "#") && !strings.HasPrefix(stmt, "using") {
			seen[stmt] = append(seen[stmt], n)
		}
		if mainDepth < 0 && strings.Contains(stmt, "main(") {
			mainDepth = depth
		}
		depth = max(0, depth+strings.Count(stmt, "{")-strings.Count(stmt, "}"))
		switch {
		case mainDepth < 0 || mainDone:
		case depth > mainDepth:
			mainLines++
		case mainLines > 0:
			mainDone = true
		}
	}

	var r QualityReport
	switch {
	case len(unindented) > 0:
		r.Readability = fmt.Sprintf("Legibilidad: revisa la sangría de %s; lo que va dentro de unas llaves se escribe más adentro para ver dónde empieza y termina cada bloque.", lineRefs(unindented))
	case tabs && spaces:
		r.Readability = "Legibilidad: mezclas tabulaciones y espacios en la sangría; usa uno solo para que el código se vea igual en cualquier editor."
	case len(shortNames) > 0:
		r.Readability = fmt.Sprintf("Legibilidad: nombres como «%s» (%s) no dicen qué guardan; un nombre descriptivo ayuda a entender el programa sin leerlo todo.", strings.Join(unique(shortList), "», «"), lineRefs(shortNames))
	case len(long) > 0:
		r.Readability = fmt.Sprintf("Legibilidad: %s es muy larga; dividirla hace más fácil leerla.", lineRefs(long))
	default:
		r.Readability = "Legibilidad: la sangría es consistente y los nombres se entienden."
	}
	r.issues[0] = len(unindented) > 0 || (tabs && spaces) || len(shortNames) > 0 || len(long) > 0

	var repeated []int
	repeatedText := ""
	for text, at := range seen {
		if len(at) >= 3 && len(at) > len(repeated) {
			repeated, repeatedText = at, text
		}
	}
	switch {
	case len(repeated) > 0:
		r.Maintainability = fmt.Sprintf("Mantenibilidad: «%s» se repite %d veces (%s); si tuvieras que cambiarla, tendrías que hacerlo en cada lugar. Busca una forma de escribirla una sola vez.", truncate(repeatedText, 60), len(repeated), lineRefs(repeated))
	case len(globals) > 0:
		r.Maintainability = fmt.Sprintf("Mantenibilidad: %s declara una variable global; declárala dentro de la función que la usa para que sea más fácil seguir quién la cambia.", lineRefs(globals))
	case mainLines > longMain:
		r.Maintainability = fmt.Sprintf("Mantenibilidad: main tiene %d líneas; separarlo en partes con una tarea cada una facilita modificarlo.", mainLines)
	default:
		r.Maintainability = "Mantenibilidad: no hay instrucciones repetidas ni variables globales, así que el programa es fácil de modificar."
	}
	r.issues[1] = len(repeated) > 0 || len(globals) > 0 || mainLines > longMain

	switch {
	case comments == 0 && len(lines) > 8:
		r.Documentation = "Documentación: no hay comentarios; agrega uno breve que diga qué resuelve el programa y otro antes de cada parte importante."
	case comments == 0:
		r.Documentation = "Documentación: con un comentario al inicio que diga qué resuelve el programa es suficiente."
	default:
		r.Documentation = fmt.Sprintf("Documentación: tienes %d %s; procura que expliquen el porqué de cada parte y no repitan lo que ya dice el código.", comments, plural(comments, "comentario", "comentarios"))
	}
	r.issues[2] = comments == 0 && len(lines) > 8
	return r
}

func unique(items []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range items {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

var qualityKeys = [3][]string{
	{"legib", "sangr", "indent"},
	{"mantenib", "repet", "repit", "global"},
	{"document", "comentario"},
}

func missingQuality(text string) [3]bool {
	folded := accentFolder.Replace(strings.ToLower(text))
	var missing [3]bool
	for i, keys := range qualityKeys {
		missing[i] = true
		for _, k := range keys {
			if strings.Contains(folded, k) {
				missing[i] = false
			}
		}
	}
	return missing
}

var qualityNames = [3]string{"legibilidad", "mantenibilidad", "documentación"}

func MissingQuality(text string) []string {
	var out []string
	for i, missing := range missingQuality(text) {
		if missing {
			out = append(out, qualityNames[i])
		}
	}
	return out
}

func EnsureQuality(text string, q QualityReport) (string, bool) {
	missing := missingQuality(text)
	if !missing[0] && !missing[1] && !missing[2] {
		return text, true
	}
	var block []string
	if !strings.Contains(accentFolder.Replace(strings.ToLower(text)), "calidad del codigo") {
		block = append(block, "**Calidad del código**")
	}
	for i, line := range []string{q.Readability, q.Maintainability, q.Documentation} {
		if missing[i] {
			block = append(block, "- "+line)
		}
	}
	block = append(block, "")
	lines := strings.Split(strings.TrimRight(text, "\n "), "\n")
	pos := findSections(lines)[2]
	if pos < 0 {
		return strings.Join(append(append(lines, ""), block...), "\n"), false
	}
	out := append([]string{}, lines[:pos]...)
	out = append(out, block...)
	out = append(out, lines[pos:]...)
	return strings.Join(out, "\n"), false
}

type Checks struct {
	SectionsOK bool
	QualityOK bool
}

func Finalize(text string, o Outcome, q QualityReport) (string, Checks) {
	fixed, sections := EnsureSections(text, FallbackSections(o))
	fixed, quality := EnsureQuality(fixed, q)
	return fixed, Checks{SectionsOK: sections, QualityOK: quality}
}

func writeStyle(b *strings.Builder, problems []string) {
	if len(problems) == 0 {
		return
	}
	b.WriteString("Revisión automática de estilo (datos ya verificados; úsalos en «Calidad del código»):\n")
	for _, p := range problems {
		b.WriteString("- " + p + "\n")
	}
	b.WriteString("\n")
}
