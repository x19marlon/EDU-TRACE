package ai

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)


const (
	maxExampleLines = 3 
	omittedBlock = "Código omitido por EduTrace Aquí la IA escribió código que podría resolver tu ejercicio. Inténtalo tú con las pistas, si te atascas, pregúntale a tu docente.\n"
	omittedInline = "**omitido**"
)

var (
	identRe = regexp.MustCompile(`[a-z_][a-z0-9_]*`)
	quotedRe = regexp.MustCompile(`"(\\.|[^"\\])*"|'(\\.|[^'\\])*'`)
	lineNumRe = regexp.MustCompile(`^\d+\s*\|`) 
	accents = strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n")
)


var cppWords = wordSet(`int float double char bool void long short unsigned signed const auto string
	if else for while do switch case default break continue return true false
	include define iostream iomanip cmath cstdlib ctime using namespace std cout cin endl main
	getline setprecision fixed sqrt pow abs max min size length sizeof new delete struct  private nullptr null printf scanf rand srand time`)

func wordSet(s string) map[string]bool {
	set := map[string]bool{}
	for _, w := range strings.Fields(s) {
		set[w] = true
	}
	return set
}


func withoutComment(line string) string {
	if i := strings.Index(line, "//"); i >= 0 {
		return line[:i]
	}
	return line
}


func flatten(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return -1
			}
			return r
		}, withoutComment(line))
	}
	return strings.Join(lines, "\n")
}


func names(s string, minLen int) []string {
	var out []string
	seen := map[string]bool{}
	for _, line := range strings.Split(s, "\n") {
		line = quotedRe.ReplaceAllString(withoutComment(line), "")
		for _, n := range identRe.FindAllString(accents.Replace(strings.ToLower(line)), -1) {
			if len(n) >= minLen && !cppWords[n] && !seen[n] {
				seen[n] = true
				out = append(out, n)
			}
		}
	}
	return out
}


type studentRef struct {
	flat string
	names map[string]bool
}

func newStudentRef(code string, texts []string) *studentRef {
	r := &studentRef{flat: flatten(code), names: map[string]bool{}}
	for _, n := range names(code, 1) {
		r.names[n] = true
	}
	for _, t := range texts {
		r.flat += "\n" + flatten(t)
		for _, n := range names(t, 3) { 
			r.names[n] = true
		}
	}
	return r
}


func (r *studentRef) quotes(s string) bool {
	f := flatten(s)
	return f == "" || strings.Contains(r.flat, f)
}


func (r *studentRef) rewrites(s string) bool {
	if r.quotes(s) {
		return false
	}
	short, fresh := 0, false
	for _, n := range names(s, 1) {
		switch {
		case !r.names[n]:
			fresh = true
		case len(n) > 1:
			return true
		default:
			short++
		}
	}
	return short > 1 || !fresh && r.nearCopy(flatten(s))
}


func (r *studentRef) nearCopy(f string) bool {
	if len(f) < 4 || len(f) > 200 {
		return false
	}
	limit := max(1, len(f)/4)
	prev, cur := make([]int, len(f)+1), make([]int, len(f)+1)
	for _, line := range strings.Split(r.flat, "\n") {
		
		for j := range prev {
			prev[j] = j
		}
		for i := 0; i < len(line); i++ {
			cur[0] = 0
			for j := 1; j <= len(f); j++ {
				cost := 1
				if line[i] == f[j-1] {
					cost = 0
				}
				cur[j] = min(prev[j-1]+cost, prev[j]+1, cur[j-1]+1)
			}
			if cur[len(f)] <= limit {
				return true
			}
			prev, cur = cur, prev
		}
	}
	return false
}


func (r *studentRef) blockLeaks(lines []string) bool {
	fresh, program := 0, false
	for _, line := range lines {
		t := strings.TrimSpace(lineNumRe.ReplaceAllString(strings.TrimSpace(line), ""))
		if strings.HasPrefix(t, "```") || strings.Trim(t, "{}();") == "" {
			continue 
		}
		if strings.Contains(flatten(t), "main(") {
			program = true
		}
		if r.quotes(t) {
			continue 
		}
		if r.rewrites(t) {
			return true
		}
		fresh++
	}
	return fresh > maxExampleLines || program && fresh > 0
}

func codeStart(t string, final bool) (code, ready bool) {
	if t == "" {
		return false, final
	}
	c := t[0]
	switch {
	case c == '#':
		if len(t) == 1 {
			return false, final
		}
		return t[1] != '#' && t[1] != ' ', true // 
	case strings.IndexByte("{}/;", c) >= 0:
		return true, true
	case '0' <= c && c <= '9':
		rest := strings.TrimLeft(t, "0123456789 ")
		if rest == "" {
			return false, final
		}
		return rest[0] == '|', true // 
	case c != '_' && !unicode.IsLetter(firstRune(t)):
		return false, true 
	}
	end := strings.IndexFunc(t, func(r rune) bool { return r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	if end < 0 {
		return cppWords[strings.ToLower(t)], final
	}
	word, rest := strings.ToLower(t[:end]), t[end:]
	if cppWords[word] || strings.HasPrefix(rest, "::") || strings.IndexByte("[(", rest[0]) >= 0 {
		return true, true 
	}
	rest = strings.TrimLeft(rest, " \t")
	if rest == "" {
		return false, final
	}
	for _, op := range []string{"=", "+=", "-=", "*=", "/=", "%=", "++", "--", "<<", ">>"} {
		if strings.HasPrefix(rest, op) {
			return true, true 
		}
	}
	return false, true
}

func firstRune(s string) rune {
	r, _ := utf8.DecodeRuneInString(s)
	return r
}


type SolutionGuard struct {
	ref *studentRef
	emit func(string)
	buf string 
	prose bool 
	fence bool
	span bool 
	spanText string
	held []string 
	omitted int
}


func NewSolutionGuard(emit func(string), code string, texts ...string) *SolutionGuard {
	return &SolutionGuard{ref: newStudentRef(code, texts), emit: emit}
}


func (g *SolutionGuard) Omitted() int { return g.omitted }


func (g *SolutionGuard) Write(token string) {
	g.buf += token
	g.process(false)
}


func (g *SolutionGuard) Close() {
	g.process(true)
	if g.span {
		g.out("`" + g.spanText)
		g.span = false
	}
	g.fence = false
	g.release()
}

func (g *SolutionGuard) out(s string) {
	if s != "" {
		g.emit(s)
	}
}

func (g *SolutionGuard) process(final bool) {
	for g.buf != "" {
		switch {
		case g.prose:
			g.proseText()
		case g.fence:
			if !g.fenceLine(final) {
				return
			}
		default:
			if !g.lineStart(final) {
				return
			}
		}
	}
}


func (g *SolutionGuard) nextLine(final bool) (string, bool) {
	if nl := strings.IndexByte(g.buf, '\n'); nl >= 0 {
		return g.buf[:nl+1], true
	}
	return g.buf, final
}


func (g *SolutionGuard) lineStart(final bool) bool {
	line, complete := g.nextLine(final)
	t := strings.TrimSpace(line)
	if strings.HasPrefix(t, "```") || strings.HasPrefix("```", t) {
		if !complete {
			return false 
		}
		if t == "" { 
			if len(g.held) > 0 {
				g.held = append(g.held, line)
			} else {
				g.out(line)
			}
			g.buf = g.buf[len(line):]
			return true
		}
		if strings.HasPrefix(t, "```") {
			g.release()
			g.fence = true
			g.held = []string{line}
			g.buf = g.buf[len(line):]
			return true
		}
	}
	code, ready := codeStart(t, complete)
	if !ready || code && !complete {
		return false
	}
	if code {
		g.held = append(g.held, line)
		g.buf = g.buf[len(line):]
		return true
	}
	g.release()
	g.prose = true
	return true
}

func (g *SolutionGuard) fenceLine(final bool) bool {
	line, complete := g.nextLine(final)
	if !complete {
		return false
	}
	g.held = append(g.held, line)
	g.buf = g.buf[len(line):]
	if strings.HasPrefix(strings.TrimSpace(line), "```") {
		g.fence = false
		g.release()
	}
	return true
}

func (g *SolutionGuard) proseText() {
	for g.buf != "" {
		i := strings.IndexAny(g.buf, "`\n")
		if i < 0 {
			if g.span {
				g.spanText += g.buf
			} else {
				g.out(g.buf)
			}
			g.buf = ""
			return
		}
		c := g.buf[i]
		switch {
		case g.span && c == '`':
			g.closeSpan(g.spanText + g.buf[:i])
		case g.span: // salto de línea sin cerrar el fragmento: se envía tal cual
			g.out("`" + g.spanText + g.buf[:i])
			g.span = false
		default:
			g.out(g.buf[:i])
			if c == '`' {
				g.span, g.spanText = true, ""
			}
		}
		g.buf = g.buf[i+1:]
		if c == '\n' {
			g.out("\n")
			g.prose = false
			return
		}
	}
}

func (g *SolutionGuard) closeSpan(text string) {
	if g.ref.rewrites(text) {
		g.omitted++
		g.out(omittedInline)
	} else {
		g.out("`" + text + "`")
	}
	g.span = false
}


func (g *SolutionGuard) release() {
	if len(g.held) == 0 {
		return
	}
	if g.ref.blockLeaks(g.held) {
		g.omitted++
		end := len(g.held)
		for end > 0 && strings.TrimSpace(g.held[end-1]) == "" {
			end-- 
		}
		g.out(omittedBlock + strings.Join(g.held[end:], ""))
	} else {
		g.out(strings.Join(g.held, ""))
	}
	g.held = nil
}


func FilterSolution(text, code string, texts ...string) (string, int) {
	var b strings.Builder
	g := NewSolutionGuard(func(s string) { b.WriteString(s) }, code, texts...)
	g.Write(text)
	g.Close()
	return b.String(), g.Omitted()
}
