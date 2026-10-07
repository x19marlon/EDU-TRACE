package compiler

import (
	"regexp"
	"strings"
)

// Prefijo "main.cpp:7:15: " que g++ antepone a cada diagnóstico.
var diagPrefix = regexp.MustCompile(`^[^:\s]+:\d+:\d+:\s*`)

// FirstError devuelve el primer mensaje de error de g++ sin la ruta ni la posición
// (por ejemplo "expected ';' before 'if'"), para agrupar errores frecuentes.
func FirstError(compilerOutput string) string {
	for _, line := range strings.Split(compilerOutput, "\n") {
		line = diagPrefix.ReplaceAllString(strings.TrimSpace(line), "")
		if rest, ok := strings.CutPrefix(line, "error: "); ok {
			rest = strings.TrimSpace(rest)
			if len(rest) > 160 {
				rest = rest[:160]
			}
			return strings.ToValidUTF8(rest, "")
		}
		if rest, ok := strings.CutPrefix(line, "fatal error: "); ok {
			return strings.ToValidUTF8(strings.TrimSpace(rest), "")
		}
	}
	return ""
}
