package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Documentos del curso que se dan al modelo como contexto ("CRITERIOS DEL CURSO").
//
//	syllabus/
//	  comun/                 se usa en todas las materias (opcional)
//	  <materia>/
//	    materia.json         {"nombre": ..., "alias": [...], "por_defecto": bool}
//	    *.md                 documentos de la materia
//
// La materia de una petición se elige comparando el nombre del curso (materia en la
// app) con el nombre y los alias de cada carpeta.

const (
	commonDir   = "comun"
	subjectMeta = "materia.json"
	noDocsText  = "(No hay documentos específicos para este curso. Usa buenas prácticas generales de programación y ajusta el nivel al código del estudiante.)"
)

type subject struct {
	name      string
	aliases   []string // normalizados, incluye el nombre
	isDefault bool
	docs      string
}

// Syllabus guarda los documentos cargados al arrancar, agrupados por materia.
type Syllabus struct {
	common   string
	subjects []subject
	fallback *subject // materia con "por_defecto": se usa si la petición no tiene curso
}

type subjectFile struct {
	Name      string   `json:"nombre"`
	Aliases   []string `json:"alias"`
	IsDefault bool     `json:"por_defecto"`
}

// findSyllabusDir busca syllabus/ en el directorio actual o junto al binario.
func findSyllabusDir() string {
	dir := "syllabus"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		execPath, _ := os.Executable()
		dir = filepath.Join(filepath.Dir(execPath), "syllabus")
	}
	return dir
}

// LoadSyllabus lee comun/ y cada carpeta de materia. Los .md sueltos en la raíz
// (como README.md) no se envían al modelo.
func LoadSyllabus(dir string) (*Syllabus, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading syllabus directory %q: %w", dir, err)
	}

	s := &Syllabus{}
	total := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		docs, n, err := readMarkdown(path)
		if err != nil {
			return nil, err
		}
		total += n

		if entry.Name() == commonDir {
			s.common = docs
			continue
		}

		raw, err := os.ReadFile(filepath.Join(path, subjectMeta))
		if err != nil {
			return nil, fmt.Errorf("materia %q sin %s: %w", entry.Name(), subjectMeta, err)
		}
		var meta subjectFile
		if err := json.Unmarshal(raw, &meta); err != nil {
			return nil, fmt.Errorf("leyendo %s/%s: %w", entry.Name(), subjectMeta, err)
		}
		if meta.Name == "" {
			return nil, fmt.Errorf("%s/%s: falta \"nombre\"", entry.Name(), subjectMeta)
		}

		sub := subject{name: meta.Name, isDefault: meta.IsDefault, docs: docs}
		for _, a := range append([]string{meta.Name}, meta.Aliases...) {
			if a = normalizeSubject(a); a != "" {
				sub.aliases = append(sub.aliases, a)
			}
		}
		s.subjects = append(s.subjects, sub)
	}

	if total == 0 {
		return nil, fmt.Errorf("no .md files found in %q", dir)
	}
	for i := range s.subjects {
		if s.subjects[i].isDefault {
			if s.fallback != nil {
				return nil, fmt.Errorf("más de una materia tiene \"por_defecto\": %q y %q", s.fallback.name, s.subjects[i].name)
			}
			s.fallback = &s.subjects[i]
		}
	}
	return s, nil
}

// readMarkdown concatena los .md de una carpeta en orden alfabético.
func readMarkdown(dir string) (string, int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", 0, fmt.Errorf("reading %q: %w", dir, err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	var parts []string
	for _, name := range names {
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return "", 0, fmt.Errorf("reading %s: %w", name, err)
		}
		parts = append(parts, fmt.Sprintf("--- %s/%s ---\n%s", filepath.Base(dir), name, string(content)))
	}
	return strings.Join(parts, "\n\n"), len(parts), nil
}

// match devuelve la materia cuyo nombre o alias aparece en el nombre del curso.
// Si varias coinciden gana el alias más largo (el más específico).
func (s *Syllabus) match(course string) *subject {
	c := normalizeSubject(course)
	if c == "" {
		return nil
	}
	var best *subject
	bestLen := 0
	for i := range s.subjects {
		for _, a := range s.subjects[i].aliases {
			if len(a) > bestLen && strings.Contains(c, a) {
				best, bestLen = &s.subjects[i], len(a)
			}
		}
	}
	return best
}

// For arma el contexto para un curso: comun/ más los documentos de su materia.
// Sin curso (estudiante practicando sin clase) se usa la materia por defecto; con
// un curso que no coincide con ninguna materia solo se usa comun/.
func (s *Syllabus) For(course string) (docs, subjectName string) {
	sub := s.fallback
	if strings.TrimSpace(course) != "" {
		sub = s.match(course)
	}

	var parts []string
	if s.common != "" {
		parts = append(parts, s.common)
	}
	if sub != nil {
		subjectName = sub.name
		if sub.docs != "" {
			parts = append(parts, sub.docs)
		}
	}
	if len(parts) == 0 {
		return noDocsText, subjectName
	}
	return strings.Join(parts, "\n\n"), subjectName
}

// Version concatena todos los documentos, para identificar con qué versión se generó
// una propuesta.
func (s *Syllabus) Version() string {
	parts := []string{s.common}
	for _, sub := range s.subjects {
		parts = append(parts, sub.name, sub.docs)
	}
	return strings.Join(parts, "\n")
}

// Subjects devuelve los nombres de las materias cargadas (para el log de arranque).
func (s *Syllabus) Subjects() []string {
	names := make([]string, len(s.subjects))
	for i, sub := range s.subjects {
		names[i] = sub.name
	}
	return names
}

// normalizeSubject pasa a minúsculas, quita tildes y colapsa espacios.
func normalizeSubject(s string) string {
	return strings.Join(strings.Fields(accents.Replace(strings.ToLower(s))), " ")
}
