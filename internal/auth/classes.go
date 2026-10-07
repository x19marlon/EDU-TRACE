package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var (
	ErrNotFound    = errors.New("no encontrado")
	ErrInvalidCode = errors.New("el código de clase no existe")
	ErrNotEnrolled = errors.New("no estás inscrito en esa clase")
)

// Course es una materia creada por un docente.
type Course struct {
	ID        string    `json:"id"`
	TeacherID string    `json:"teacher_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// Group es una clase (grupo) de una materia. Los estudiantes se unen con JoinCode.
type Group struct {
	ID         string    `json:"id"`
	CourseID   string    `json:"course_id"`
	Name       string    `json:"name"`
	JoinCode   string    `json:"join_code"`
	StudentIDs []string  `json:"student_ids"`
	CreatedAt  time.Time `json:"created_at"`
}

// Tipos de registro: la entrega oficial (botón Enviar) y los intentos de compilación.
const (
	KindOfficial = "official"
	KindAttempt  = "attempt"
)

// SubmissionMeta es el resumen de un envío o intento (se guarda en el archivo principal).
type SubmissionMeta struct {
	ID            string    `json:"id"`
	Kind          string    `json:"kind"` // KindOfficial o KindAttempt; vacío en datos antiguos = oficial
	GroupID       string    `json:"group_id"`
	AssignmentID  string    `json:"assignment_id,omitempty"` // taller al que responde (opcional)
	StudentID     string    `json:"student_id"`
	CreatedAt     time.Time `json:"created_at"`
	Success       bool      `json:"success"`
	RunError      string    `json:"run_error,omitempty"`
	ExitCode      int       `json:"exit_code"`
	Lines         int       `json:"lines"`
	HasAIFeedback bool      `json:"has_ai_feedback"`
	ErrorSummary  string    `json:"error_summary,omitempty"` // primer error del compilador, para ver patrones
	CodeHash      string    `json:"code_hash,omitempty"`     // evita guardar dos veces seguidas el mismo intento
}

// IsAttempt indica si es un intento de compilación (no una entrega oficial).
func (m *SubmissionMeta) IsAttempt() bool { return m.Kind == KindAttempt }

// matchesKind filtra por "official" (por defecto), "attempt" o "all".
func (m *SubmissionMeta) matchesKind(kind string) bool {
	switch kind {
	case "all":
		return true
	case KindAttempt:
		return m.IsAttempt()
	default:
		return !m.IsAttempt()
	}
}

// Submission es un envío completo (se guarda en su propio archivo).
type Submission struct {
	SubmissionMeta
	Code           string `json:"code"`
	Stdin          string `json:"stdin"`
	CompilerOutput string `json:"compiler_output"`
	ProgramOutput  string `json:"program_output"`
	AIFeedback     string `json:"ai_feedback"`          // retroalimentación formal que vio el estudiante
	AIFeedbackInf  string `json:"ai_feedback_informal"` // retroalimentación informal que vio el estudiante
}

// Sin caracteres ambiguos (0/O, 1/I/L) para dictar el código en clase.
const joinAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

func newJoinCode() (string, error) {
	b := make([]byte, 6)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(joinAlphabet))))
		if err != nil {
			return "", err
		}
		b[i] = joinAlphabet[n.Int64()]
	}
	return string(b), nil
}

// NormalizeJoinCode acepta el código con minúsculas, espacios o guiones.
func NormalizeJoinCode(code string) string {
	code = strings.ToUpper(code)
	return strings.NewReplacer(" ", "", "-", "").Replace(code)
}

// ---------- Docente ----------

// CreateCourse crea una materia para el docente.
func (s *Store) CreateCourse(teacherID, name string) (*Course, error) {
	id, err := randomHex(12)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	c := &Course{ID: id, TeacherID: teacherID, Name: strings.TrimSpace(name), CreatedAt: time.Now().UTC()}
	s.data.Courses[id] = c
	if err := s.saveLocked(); err != nil {
		delete(s.data.Courses, id)
		return nil, err
	}
	cp := *c
	return &cp, nil
}

// CreateGroup crea una clase dentro de una materia del docente.
func (s *Store) CreateGroup(teacherID, courseID, name string) (*Group, error) {
	id, err := randomHex(12)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	c, ok := s.data.Courses[courseID]
	if !ok || c.TeacherID != teacherID {
		return nil, ErrNotFound
	}
	var code string
	for {
		if code, err = newJoinCode(); err != nil {
			return nil, err
		}
		if s.groupByCodeLocked(code) == nil {
			break
		}
	}
	g := &Group{ID: id, CourseID: courseID, Name: strings.TrimSpace(name), JoinCode: code, StudentIDs: []string{}, CreatedAt: time.Now().UTC()}
	s.data.Groups[id] = g
	if err := s.saveLocked(); err != nil {
		delete(s.data.Groups, id)
		return nil, err
	}
	cp := *g
	cp.StudentIDs = nil
	return &cp, nil
}

func (s *Store) groupByCodeLocked(code string) *Group {
	for _, g := range s.data.Groups {
		if g.JoinCode == code {
			return g
		}
	}
	return nil
}

// GroupSummary resume una clase para el listado de materias del docente.
type GroupSummary struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	JoinCode        string    `json:"join_code"`
	StudentCount    int       `json:"student_count"`
	SubmissionCount int       `json:"submission_count"`
	CreatedAt       time.Time `json:"created_at"`
}

// CourseSummary es una materia con sus clases.
type CourseSummary struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	CreatedAt time.Time      `json:"created_at"`
	Groups    []GroupSummary `json:"groups"`
}

func (s *Store) submissionCountsLocked() map[string]int {
	counts := map[string]int{}
	for _, m := range s.data.Submissions {
		if !m.IsAttempt() {
			counts[m.GroupID]++
		}
	}
	return counts
}

// TeacherCourses lista las materias del docente con sus clases.
func (s *Store) TeacherCourses(teacherID string) []CourseSummary {
	s.mu.Lock()
	defer s.mu.Unlock()

	subs := s.submissionCountsLocked()
	byCourse := map[string][]GroupSummary{}
	for _, g := range s.data.Groups {
		byCourse[g.CourseID] = append(byCourse[g.CourseID], GroupSummary{
			ID: g.ID, Name: g.Name, JoinCode: g.JoinCode,
			StudentCount: len(g.StudentIDs), SubmissionCount: subs[g.ID], CreatedAt: g.CreatedAt,
		})
	}
	out := []CourseSummary{}
	for _, c := range s.data.Courses {
		if c.TeacherID != teacherID {
			continue
		}
		groups := byCourse[c.ID]
		if groups == nil {
			groups = []GroupSummary{}
		}
		sort.Slice(groups, func(i, j int) bool { return groups[i].CreatedAt.Before(groups[j].CreatedAt) })
		out = append(out, CourseSummary{ID: c.ID, Name: c.Name, CreatedAt: c.CreatedAt, Groups: groups})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

// GroupStudent es un estudiante dentro de una clase.
type GroupStudent struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Email            string     `json:"email"`
	Activity         Activity   `json:"activity"`
	SubmissionCount  int        `json:"submission_count"` // entregas oficiales
	LastSubmissionAt *time.Time `json:"last_submission_at,omitempty"`
	AttemptCount     int        `json:"attempt_count"` // intentos de compilación
	LastAttemptAt    *time.Time `json:"last_attempt_at,omitempty"`
}

// GroupDetail es una clase con sus estudiantes, para el docente.
type GroupDetail struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	JoinCode   string         `json:"join_code"`
	CourseID   string         `json:"course_id"`
	CourseName string         `json:"course_name"`
	Students   []GroupStudent `json:"students"`
}

// ownedGroupLocked devuelve la clase si pertenece a una materia del docente.
func (s *Store) ownedGroupLocked(teacherID, groupID string) (*Group, *Course, error) {
	g, ok := s.data.Groups[groupID]
	if !ok {
		return nil, nil, ErrNotFound
	}
	c, ok := s.data.Courses[g.CourseID]
	if !ok || c.TeacherID != teacherID {
		return nil, nil, ErrNotFound // no se revela que la clase existe
	}
	return g, c, nil
}

// GroupForTeacher devuelve el detalle de una clase del docente.
func (s *Store) GroupForTeacher(teacherID, groupID string) (*GroupDetail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	g, c, err := s.ownedGroupLocked(teacherID, groupID)
	if err != nil {
		return nil, err
	}

	type counter struct {
		n    int
		last time.Time
	}
	type agg struct{ official, attempts counter }
	perStudent := map[string]*agg{}
	for _, m := range s.data.Submissions {
		if m.GroupID != groupID {
			continue
		}
		a := perStudent[m.StudentID]
		if a == nil {
			a = &agg{}
			perStudent[m.StudentID] = a
		}
		c := &a.official
		if m.IsAttempt() {
			c = &a.attempts
		}
		c.n++
		if m.CreatedAt.After(c.last) {
			c.last = m.CreatedAt
		}
	}

	students := make([]GroupStudent, 0, len(g.StudentIDs))
	for _, sid := range g.StudentIDs {
		u, ok := s.data.Users[sid]
		if !ok {
			continue
		}
		gs := GroupStudent{ID: u.ID, Name: u.Name, Email: u.Email, Activity: u.Activity}
		if a := perStudent[sid]; a != nil {
			if a.official.n > 0 {
				last := a.official.last
				gs.SubmissionCount, gs.LastSubmissionAt = a.official.n, &last
			}
			if a.attempts.n > 0 {
				last := a.attempts.last
				gs.AttemptCount, gs.LastAttemptAt = a.attempts.n, &last
			}
		}
		students = append(students, gs)
	}
	sort.Slice(students, func(i, j int) bool {
		return strings.ToLower(students[i].Name) < strings.ToLower(students[j].Name)
	})

	return &GroupDetail{
		ID: g.ID, Name: g.Name, JoinCode: g.JoinCode,
		CourseID: c.ID, CourseName: c.Name, Students: students,
	}, nil
}

// GroupSubmissions lista los registros de una clase del docente (opcionalmente de un
// estudiante). kind: "official" (por defecto), "attempt" o "all".
func (s *Store) GroupSubmissions(teacherID, groupID, studentID, kind string) ([]SubmissionMeta, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, _, err := s.ownedGroupLocked(teacherID, groupID); err != nil {
		return nil, err
	}
	out := []SubmissionMeta{}
	for _, m := range s.data.Submissions {
		if m.GroupID == groupID && (studentID == "" || m.StudentID == studentID) && m.matchesKind(kind) {
			out = append(out, *m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

// SubmissionForTeacher devuelve un envío completo si es de una clase del docente.
func (s *Store) SubmissionForTeacher(teacherID, submissionID string) (*Submission, error) {
	s.mu.Lock()
	m, ok := s.data.Submissions[submissionID]
	var groupID string
	if ok {
		groupID = m.GroupID
	}
	var err error
	if ok {
		_, _, err = s.ownedGroupLocked(teacherID, groupID)
	}
	s.mu.Unlock()

	if !ok || err != nil {
		return nil, ErrNotFound
	}
	return s.readSubmission(submissionID)
}

// ---------- Estudiante ----------

// StudentGroup es una clase vista por el estudiante.
type StudentGroup struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	CourseName  string `json:"course_name"`
	TeacherName string `json:"teacher_name"`
}

func (s *Store) studentGroupLocked(g *Group) StudentGroup {
	sg := StudentGroup{ID: g.ID, Name: g.Name}
	if c, ok := s.data.Courses[g.CourseID]; ok {
		sg.CourseName = c.Name
		if t, ok := s.data.Users[c.TeacherID]; ok {
			sg.TeacherName = t.Name
		}
	}
	return sg
}

// JoinGroup inscribe al estudiante en la clase con ese código (idempotente).
func (s *Store) JoinGroup(studentID, code string) (*StudentGroup, error) {
	code = NormalizeJoinCode(code)
	s.mu.Lock()
	defer s.mu.Unlock()

	g := s.groupByCodeLocked(code)
	if g == nil {
		return nil, ErrInvalidCode
	}
	for _, id := range g.StudentIDs {
		if id == studentID {
			sg := s.studentGroupLocked(g)
			return &sg, nil
		}
	}
	g.StudentIDs = append(g.StudentIDs, studentID)
	if err := s.saveLocked(); err != nil {
		g.StudentIDs = g.StudentIDs[:len(g.StudentIDs)-1]
		return nil, err
	}
	sg := s.studentGroupLocked(g)
	return &sg, nil
}

// StudentGroups lista las clases del estudiante.
func (s *Store) StudentGroups(studentID string) []StudentGroup {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []StudentGroup{}
	for _, g := range s.data.Groups {
		for _, id := range g.StudentIDs {
			if id == studentID {
				out = append(out, s.studentGroupLocked(g))
				break
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CourseName+out[i].Name < out[j].CourseName+out[j].Name })
	return out
}

// IsEnrolled indica si el estudiante pertenece a la clase.
func (s *Store) IsEnrolled(studentID, groupID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.data.Groups[groupID]
	if !ok {
		return false
	}
	for _, id := range g.StudentIDs {
		if id == studentID {
			return true
		}
	}
	return false
}

func (s *Store) submissionPath(id string) string {
	return filepath.Join(s.dir, "submissions", id+".json")
}

func (s *Store) readSubmission(id string) (*Submission, error) {
	raw, err := os.ReadFile(s.submissionPath(id))
	if err != nil {
		return nil, fmt.Errorf("leer envío: %w", err)
	}
	var sub Submission
	if err := json.Unmarshal(raw, &sub); err != nil {
		return nil, fmt.Errorf("leer envío: %w", err)
	}
	return &sub, nil
}

// ErrDuplicateAttempt indica que el intento es idéntico al anterior del estudiante (no se guarda).
var ErrDuplicateAttempt = errors.New("intento repetido")

func codeHash(code, stdin string) string {
	sum := sha256.Sum256([]byte(code + "\x00" + stdin))
	return hex.EncodeToString(sum[:])
}

// lastAttemptHashLocked devuelve la huella del último intento del estudiante en la clase.
func (s *Store) lastAttemptHashLocked(studentID, groupID string) string {
	var last *SubmissionMeta
	for _, m := range s.data.Submissions {
		if m.IsAttempt() && m.StudentID == studentID && m.GroupID == groupID &&
			(last == nil || m.CreatedAt.After(last.CreatedAt)) {
			last = m
		}
	}
	if last == nil {
		return ""
	}
	return last.CodeHash
}

// AddSubmission guarda una entrega oficial o un intento (sub.Kind). Rellena ID y CreatedAt.
// Un intento idéntico al anterior del mismo estudiante en la clase devuelve ErrDuplicateAttempt.
func (s *Store) AddSubmission(sub *Submission) error {
	if sub.Kind != KindAttempt {
		sub.Kind = KindOfficial
	}
	sub.CodeHash = codeHash(sub.Code, sub.Stdin)
	if sub.IsAttempt() {
		s.mu.Lock()
		dup := s.lastAttemptHashLocked(sub.StudentID, sub.GroupID) == sub.CodeHash
		s.mu.Unlock()
		if dup {
			return ErrDuplicateAttempt
		}
	}

	id, err := randomHex(12)
	if err != nil {
		return err
	}
	sub.ID = id
	sub.CreatedAt = time.Now().UTC()
	sub.Lines = strings.Count(strings.TrimRight(sub.Code, "\n"), "\n") + 1
	sub.HasAIFeedback = strings.TrimSpace(sub.AIFeedback) != "" || strings.TrimSpace(sub.AIFeedbackInf) != ""

	if !s.IsEnrolled(sub.StudentID, sub.GroupID) {
		return ErrNotEnrolled
	}

	// Primero el archivo con el código; después el índice, para no dejar
	// entradas en el índice que apunten a archivos inexistentes.
	raw, err := json.Marshal(sub)
	if err != nil {
		return err
	}
	path := s.submissionPath(id)
	if err := os.WriteFile(path+".tmp", raw, 0o600); err != nil {
		return err
	}
	if err := os.Rename(path+".tmp", path); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	meta := sub.SubmissionMeta
	s.data.Submissions[id] = &meta
	if err := s.saveLocked(); err != nil {
		delete(s.data.Submissions, id)
		_ = os.Remove(path)
		return err
	}
	return nil
}

// StudentSubmission es un envío propio visto por el estudiante.
type StudentSubmission struct {
	SubmissionMeta
	GroupName  string `json:"group_name"`
	CourseName string `json:"course_name"`
}

// StudentSubmissions lista los últimos envíos del estudiante.
func (s *Store) StudentSubmissions(studentID string, limit int) []StudentSubmission {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []StudentSubmission{}
	for _, m := range s.data.Submissions {
		if m.StudentID != studentID || m.IsAttempt() {
			continue
		}
		ss := StudentSubmission{SubmissionMeta: *m}
		if g, ok := s.data.Groups[m.GroupID]; ok {
			sg := s.studentGroupLocked(g)
			ss.GroupName, ss.CourseName = sg.Name, sg.CourseName
		}
		out = append(out, ss)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

// StudentProcess reúne el proceso de un estudiante en una clase del docente para analizarlo:
// sus intentos y entregas más recientes (completos, del más antiguo al más reciente).
func (s *Store) StudentProcess(teacherID, groupID, studentID string, limit int) (string, []Submission, error) {
	s.mu.Lock()
	g, _, err := s.ownedGroupLocked(teacherID, groupID)
	if err != nil {
		s.mu.Unlock()
		return "", nil, err
	}
	u, ok := s.data.Users[studentID]
	if !ok || !containsID(g.StudentIDs, studentID) {
		s.mu.Unlock()
		return "", nil, ErrNotFound
	}
	name := u.Name
	metas := []SubmissionMeta{}
	for _, m := range s.data.Submissions {
		if m.GroupID == groupID && m.StudentID == studentID {
			metas = append(metas, *m)
		}
	}
	s.mu.Unlock()

	sort.Slice(metas, func(i, j int) bool { return metas[i].CreatedAt.Before(metas[j].CreatedAt) })
	if len(metas) > limit {
		metas = metas[len(metas)-limit:]
	}
	out := make([]Submission, 0, len(metas))
	for _, m := range metas {
		sub, err := s.readSubmission(m.ID)
		if err != nil {
			return "", nil, err
		}
		out = append(out, *sub)
	}
	return name, out, nil
}
