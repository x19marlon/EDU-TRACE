package compiler

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/marlon/edu-trace/internal/model"
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

var quoted = regexp.MustCompile(`[‘'"]([^’'"]*)[’'"]`)

var keepNames = map[string]bool{"cout": true, "cin": true, "endl": true, "string": true, "getline": true,
	"setprecision": true, "fixed": true, "sqrt": true, "pow": true, "main": true, "else": true, "if": true, "return": true,
	"for": true, "while": true, "do": true, "switch": true, "case": true, "break": true, "int": true, "double": true,
	"float": true, "char": true, "bool": true, "void": true}

func NormalizeError(msg string) string {
	return quoted.ReplaceAllStringFunc(strings.TrimSpace(msg), func(m string) string {
		inner := quoted.FindStringSubmatch(m)[1]
		if keepNames[inner] || strings.HasPrefix(inner, "operator") || !strings.ContainsAny(strings.ToLower(inner), "abcdefghijklmnopqrstuvwxyz_0123456789") {
			return "'" + inner + "'"
		}
		return "'…'"
	})
}

var explanations = []struct{ pattern, text string }{
	{"'cout' was not declared", "Falta #include <iostream> o using namespace std; antes de usar cout."},
	{"'cin' was not declared", "Falta #include <iostream> o using namespace std; antes de usar cin."},
	{"'endl' was not declared", "Falta using namespace std; (o escribir std::endl)."},
	{"'string' was not declared", "Falta #include <string> o using namespace std; para usar string."},
	{"'setprecision' was not declared", "Falta #include <iomanip> para usar setprecision."},
	{"expected ';'", "Falta un punto y coma al final de la instrucción anterior."},
	{"or ';' before", "Falta un punto y coma al final de la instrucción anterior."},
	{"was not declared in this scope", "Se usa una variable o función que no está declarada, está escrita distinto o se declaró dentro de otro bloque."},
	{"expected '}' at end of input", "Falta cerrar una llave: cada { necesita su }."},
	{"expected primary-expression", "Hay un operador, paréntesis o coma de más o en el lugar equivocado."},
	{"expected ')'", "Falta cerrar un paréntesis."},
	{"expected unqualified-id", "Hay una instrucción fuera de una función o una llave de más."},
	{"'else' without a previous 'if'", "Un else quedó sin su if: revisa las llaves y que no haya un ; justo después del if."},
	{"no match for 'operator>>'", "Se usó >> con algo que no se puede leer (por ejemplo, cout >> x en lugar de cin >> x)."},
	{"no match for 'operator<<'", "Se usó << con algo que no se puede mostrar (por ejemplo, cin << x en lugar de cout << x)."},
	{"lvalue required as left operand of assignment", "Se usó = donde se quería comparar con ==, o se asigna a algo que no es una variable."},
	{"redeclaration of", "La misma variable se declaró dos veces en el mismo bloque."},
	{"conflicting declaration", "La misma variable se declaró dos veces con tipos distintos."},
	{"invalid conversion", "Se asigna un valor de un tipo a una variable de otro tipo incompatible."},
	{"cannot convert", "Se asigna un valor de un tipo a una variable de otro tipo incompatible."},
	{"does not name a type", "Un tipo está mal escrito o falta el #include que lo define."},
	{"missing terminating", "Falta cerrar unas comillas."},
	{"stray", "Hay un carácter extraño en el código, por ejemplo comillas tipográficas copiadas de un documento."},
	{"too few arguments", "La función se llama con menos datos de los que pide."},
	{"too many arguments", "La función se llama con más datos de los que pide."},
	{"expected initializer", "Una declaración está incompleta o le falta un punto y coma a la línea anterior."},
	{"invalid operands", "Se aplica un operador a tipos que no lo admiten (por ejemplo, % con números con decimales)."},
}

func ExplainError(normalized string) string {
	for _, e := range explanations {
		if strings.Contains(normalized, e.pattern) {
			return e.text
		}
	}
	return ""
}

func DescribeRuntime(runError string, exitCode int) (string, string) {
	switch {
	case strings.Contains(runError, "tiempo"):
		return "Se agotó el tiempo de ejecución", "Un ciclo que no termina o un programa que espera una entrada que no recibe."
	case exitCode == 136:
		return "División entera por cero", "Se dividió (o se sacó módulo) entre una variable que valía 0."
	case exitCode == 139:
		return "Acceso a memoria inválida", "Un índice fuera del arreglo o la matriz, o un ciclo que se pasa del tamaño."
	case exitCode == 134:
		return "El programa abortó", "Una operación inválida en tiempo de ejecución, por ejemplo convertir un texto que no es número."
	case exitCode < 0:
		return "El programa terminó de forma anormal", "Suele ser un índice fuera del arreglo o una división por cero."
	default:
		return fmt.Sprintf("Terminó con código de salida %d", exitCode), "main devolvió un valor distinto de 0 o el programa se detuvo con un error."
	}
}

var diagLine = regexp.MustCompile(`^[^:\s]+:(\d+):\d+:\s*(?:fatal )?error:\s*(.+)$`)

const maxExplanations = 5

func Explain(result *model.CompileResult) []model.ErrorExplanation {
	var out []model.ErrorExplanation
	seen := map[string]bool{}
	for _, line := range strings.Split(result.CompilerOutput, "\n") {
		m := diagLine.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		text := ExplainError(NormalizeError(m[2]))
		key := m[1] + text
		if text == "" || seen[key] {
			continue
		}
		seen[key] = true
		n, _ := strconv.Atoi(m[1])
		out = append(out, model.ErrorExplanation{Line: n, Message: strings.ToValidUTF8(m[2], ""), Explanation: text})
		if len(out) == maxExplanations {
			break
		}
	}
	if result.Success && (result.ExitCode != 0 || result.Error != "") {
		title, text := DescribeRuntime(result.Error, result.ExitCode)
		out = append(out, model.ErrorExplanation{Message: title, Explanation: text})
	}
	return out
}
