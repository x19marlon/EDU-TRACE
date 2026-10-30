package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"log/slog"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/marlon/edu-trace/internal/model"
)

var (
	ErrNotFound = errors.New("no encontrado")
	ErrInvalidCode = errors.New("el código de clase no existe")
	ErrNotEnrolled = errors.New("no estás inscrito en esa clase")
)

type Course struct {
	ID string `json:"id"`
	TeacherID string `json:"teacher_id"`
	Name string `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Group struct {
	ID string `json:"id"`
	CourseID string `json:"course_id"`
	Name string `json:"name"`
	JoinCode string `json:"join_code"`
	StudentIDs []string `json:"student_ids"`
	CreatedAt time.Time `json:"created_at"`
	AIMode string `json:"ai_mode"`
}

const (
	KindOfficial = "official"
	KindAttempt = "attempt"
)

type SubmissionMeta struct {
	ID string `json:"id"`
	Kind string `json:"kind"`
	GroupID string `json:"group_id"`
	AssignmentID string `json:"assignment_id,omitempty"` // taller al que responde
	StudentID string `json:"student_id"`
	CreatedAt time.Time `json:"created_at"`
	Success bool `json:"success"`
	RunError string `json:"run_error,omitempty"`
	ExitCode int `json:"exit_code"`
	Lines int `json:"lines"`
	HasAIFeedback bool `json:"has_ai_feedback"`
	ErrorSummary string `json:"error_summary,omitempty"`
	CodeHash string `json:"code_hash,omitempty"`
	DueAt *time.Time `json:"due_at,omitempty"`
	LateSeconds int64 `json:"late_seconds,omitempty"`
	LatePenalty *float64 `json:"late_penalty,omitempty"`
	TestsTotal int `json:"tests_total,omitempty"`
	TestsPassed int `json:"tests_passed,omitempty"`
}

func (m *SubmissionMeta) IsAttempt() bool { return m.Kind == KindAttempt }

type Submission struct {
	SubmissionMeta
	Code string `json:"code"`
	Stdin string `json:"stdin"`
	CompilerOutput string `json:"compiler_output"`
	ProgramOutput string `json:"program_output"`
	AIFeedback string `json:"ai_feedback"` // retroalimentación formal que vio el estudiante
	AIFeedbackInf string `json:"ai_feedback_informal"` // retroalimentación informal que vio el estudiante
	Tests []model.TestResult `json:"tests,omitempty"`
}

const (
	metaCols = `s.id, s.kind, s.group_id, COALESCE(s.assignment_id, ''), s.student_id, s.created_at,
		s.success, s.run_error, s.exit_code, s.lines, s.has_ai_feedback, s.error_summary, s.code_hash,
		(SELECT da.due_at FROM assignments da WHERE da.id = s.assignment_id),
		(SELECT da.late_penalty FROM assignments da WHERE da.id = s.assignment_id),
		(SELECT COUNT(*) FROM submission_tests st WHERE st.submission_id = s.id),
		(SELECT COALESCE(SUM(st.passed), 0) FROM submission_tests st WHERE st.submission_id = s.id)`
	contentCols = `s.code, s.stdin, s.compiler_output, s.program_output, s.ai_feedback, s.ai_feedback_informal`
)

func scanMeta(sc scanner, extra ...any) (SubmissionMeta, error) {
	var m SubmissionMeta
	var created string
	var due sql.NullString
	var penalty sql.NullFloat64
	dest := append([]any{&m.ID, &m.Kind, &m.GroupID, &m.AssignmentID, &m.StudentID, &created,
		&m.Success, &m.RunError, &m.ExitCode, &m.Lines, &m.HasAIFeedback, &m.ErrorSummary, &m.CodeHash, &due, &penalty, &m.TestsTotal, &m.TestsPassed}, extra...)
	if err := sc.Scan(dest...); err != nil {
		return m, err
	}
	m.CreatedAt = parseTime(created)
	if m.DueAt = timePtr(due); m.DueAt != nil && m.CreatedAt.After(*m.DueAt) {
		m.LateSeconds = int64(m.CreatedAt.Sub(*m.DueAt) / time.Second)
		m.LatePenalty = floatPtr(penalty)
	}
	return m, nil
}

func scanSubmission(sc scanner) (*Submission, error) {
	var sub Submission
	m, err := scanMeta(sc, &sub.Code, &sub.Stdin, &sub.CompilerOutput, &sub.ProgramOutput, &sub.AIFeedback, &sub.AIFeedbackInf)
	if err != nil {
		return nil, err
	}
	sub.SubmissionMeta = m
	return &sub, nil
}

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

func NormalizeJoinCode(code string) string {
	code = strings.ToUpper(code)
	return strings.NewReplacer(" ", "", "-", "").Replace(code)
}

func (s *Store) CreateCourse(teacherID, name string) (*Course, error) {
	id, err := randomHex(12)
	if err != nil {
		return nil, err
	}
	c := &Course{ID: id, TeacherID: teacherID, Name: strings.TrimSpace(name), CreatedAt: time.Now().UTC()}
	if _, err := s.db.Exec(`INSERT INTO courses (id, teacher_id, name, created_at) VALUES (?, ?, ?, ?)`,
		c.ID, c.TeacherID, c.Name, fmtTime(c.CreatedAt)); err != nil {
		return nil, err
	}
	return c, nil
}

// CreateGroup crea una clase dentro de una materia del docente.
func (s *Store) CreateGroup(teacherID, courseID, name string) (*Group, error) {
	var owned int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM courses WHERE id = ? AND teacher_id = ?`,
		courseID, teacherID).Scan(&owned); err != nil {
		return nil, err
	}
	if owned == 0 {
		return nil, ErrNotFound
	}
	id, err := randomHex(12)
	if err != nil {
		return nil, err
	}
	g := &Group{ID: id, CourseID: courseID, Name: strings.TrimSpace(name), CreatedAt: time.Now().UTC(), AIMode: AIModeSupervised}
	// El código es UNIQUE: si por azar ya existe, se genera otro.
	for try := 0; try < 10; try++ {
		if g.JoinCode, err = newJoinCode(); err != nil {
			return nil, err
		}
		_, err = s.db.Exec(`INSERT INTO class_groups (id, course_id, name, join_code, created_at) VALUES (?, ?, ?, ?, ?)`,
			g.ID, g.CourseID, g.Name, g.JoinCode, fmtTime(g.CreatedAt))
		if !isUnique(err) {
			break
		}
	}
	if err != nil {
		return nil, err
	}
	return g, nil
}

// GroupSummary resume una clase para el listado de materias del docente.
type GroupSummary struct {
	ID string `json:"id"`
	Name string `json:"name"`
	JoinCode string `json:"join_code"`
	StudentCount int `json:"student_count"`
	SubmissionCount int `json:"submission_count"`
	CreatedAt time.Time `json:"created_at"`
	AIMode string `json:"ai_mode"`
}

// CourseSummary es una materia con sus clases.
type CourseSummary struct {
	ID string `json:"id"`
	Name string `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	Groups []GroupSummary `json:"groups"`
	RAEs []RAE `json:"raes"`
}

// TeacherCourses lista las materias del docente con sus clases.
func (s *Store) TeacherCourses(teacherID string) []CourseSummary {
	out, err := s.teacherCourses(teacherID)
	if err != nil {
		slog.Error("listar materias", "teacher_id", teacherID, "error", err)
		return []CourseSummary{}
	}
	return out
}

func (s *Store) teacherCourses(teacherID string) ([]CourseSummary, error) {
	out := []CourseSummary{}
	index := map[string]int{}

	rows, err := s.db.Query(`SELECT id, name, created_at FROM courses WHERE teacher_id = ? ORDER BY created_at`, teacherID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var c CourseSummary
		var created string
		if err := rows.Scan(&c.ID, &c.Name, &created); err != nil {
			rows.Close()
			return nil, err
		}
		c.CreatedAt, c.Groups, c.RAEs = parseTime(created), []GroupSummary{}, []RAE{}
		index[c.ID] = len(out)
		out = append(out, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	raes, err := s.teacherRAEs(teacherID)
	if err != nil {
		return nil, err
	}
	for i := range out {
		if list, ok := raes[out[i].ID]; ok {
			out[i].RAEs = list
		}
	}

	rows, err = s.db.Query(`SELECT g.id, g.course_id, g.name, g.join_code, g.created_at, g.ai_mode,
			(SELECT COUNT(*) FROM group_students gs WHERE gs.group_id = g.id),
			(SELECT COUNT(*) FROM submissions s WHERE s.group_id = g.id AND s.kind = 'official')
		FROM class_groups g JOIN courses c ON c.id = g.course_id
		WHERE c.teacher_id = ? ORDER BY g.created_at`, teacherID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var g GroupSummary
		var courseID, created string
		if err := rows.Scan(&g.ID, &courseID, &g.Name, &g.JoinCode, &created, &g.AIMode, &g.StudentCount, &g.SubmissionCount); err != nil {
			return nil, err
		}
		g.CreatedAt = parseTime(created)
		if i, ok := index[courseID]; ok {
			out[i].Groups = append(out[i].Groups, g)
		}
	}
	return out, rows.Err()
}

// GroupStudent es un estudiante dentro de una clase.
type GroupStudent struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Email string `json:"email"`
	Activity Activity `json:"activity"`
	SubmissionCount int `json:"submission_count"` // entregas oficiales
	LastSubmissionAt *time.Time `json:"last_submission_at,omitempty"`
	AttemptCount int `json:"attempt_count"` // intentos de compilación
	LastAttemptAt *time.Time `json:"last_attempt_at,omitempty"`
}

// GroupDetail es una clase con sus estudiantes, para el docente.
type GroupDetail struct {
	ID string `json:"id"`
	Name string `json:"name"`
	JoinCode string `json:"join_code"`
	CourseID string `json:"course_id"`
	CourseName string `json:"course_name"`
	Students []GroupStudent `json:"students"`
	Monitors []GroupMonitor `json:"monitors"`
	AIMode string `json:"ai_mode"`
}

// ownedGroup devuelve la clase si pertenece a una materia del docente.
// Si no, ErrNotFound: no se revela que la clase existe.
func (s *Store) ownedGroup(q querier, teacherID, groupID string) (*GroupDetail, error) {
	g := &GroupDetail{ID: groupID, Students: []GroupStudent{}, Monitors: []GroupMonitor{}}
	err := q.QueryRow(`SELECT g.name, g.join_code, c.id, c.name, g.ai_mode
		FROM class_groups g JOIN courses c ON c.id = g.course_id
		WHERE g.id = ? AND c.teacher_id = ?`, groupID, teacherID).Scan(&g.Name, &g.JoinCode, &g.CourseID, &g.CourseName, &g.AIMode)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return g, nil
}

// GroupForTeacher devuelve el detalle de una clase del docente.
func (s *Store) GroupForTeacher(teacherID, groupID string) (*GroupDetail, error) {
	g, err := s.ownedGroup(s.db, teacherID, groupID)
	if err != nil {
		return nil, err
	}
	return g, s.fillGroup(g)
}

func (s *Store) fillGroup(g *GroupDetail) error {
	monitors, err := s.groupMonitors(g.ID)
	if err != nil {
		return err
	}
	g.Monitors = monitors
	rows, err := s.db.Query(`SELECT u.id, u.name, u.email, u.compiles, u.feedbacks, u.last_active_at,
			COALESCE(SUM(s.kind = 'official'), 0), MAX(CASE WHEN s.kind = 'official' THEN s.created_at END),
			COALESCE(SUM(s.kind = 'attempt'), 0),  MAX(CASE WHEN s.kind = 'attempt'  THEN s.created_at END)
		FROM group_students gs
		JOIN users u ON u.id = gs.student_id
		LEFT JOIN submissions s ON s.group_id = gs.group_id AND s.student_id = gs.student_id
		WHERE gs.group_id = ?
		GROUP BY u.id`, g.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var st GroupStudent
		var lastActive, lastSub, lastAttempt sql.NullString
		if err := rows.Scan(&st.ID, &st.Name, &st.Email, &st.Activity.Compiles, &st.Activity.Feedbacks, &lastActive,
			&st.SubmissionCount, &lastSub, &st.AttemptCount, &lastAttempt); err != nil {
			return err
		}
		if t := timePtr(lastActive); t != nil {
			st.Activity.LastActiveAt = *t
		}
		st.LastSubmissionAt, st.LastAttemptAt = timePtr(lastSub), timePtr(lastAttempt)
		g.Students = append(g.Students, st)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	// Se ordena en Go: lower() de SQLite no entiende tildes ("Ángela").
	sort.Slice(g.Students, func(i, j int) bool {
		return strings.ToLower(g.Students[i].Name) < strings.ToLower(g.Students[j].Name)
	})
	return nil
}

// kindFilter traduce "official" (por defecto), "attempt" o "all" a una condición SQL.
func kindFilter(kind string) string {
	switch kind {
	case "all":
		return ""
	case KindAttempt:
		return ` AND s.kind = 'attempt'`
	default:
		return ` AND s.kind = 'official'`
	}
}

// GroupSubmissions lista los registros de una clase del docente (opcionalmente de un
// estudiante). kind: "official" (por defecto), "attempt" o "all".
func (s *Store) GroupSubmissions(teacherID, groupID, studentID, kind string) ([]SubmissionMeta, error) {
	if _, err := s.ownedGroup(s.db, teacherID, groupID); err != nil {
		return nil, err
	}
	return s.groupSubmissions(groupID, studentID, kind)
}

func (s *Store) groupSubmissions(groupID, studentID, kind string) ([]SubmissionMeta, error) {
	query := `SELECT ` + metaCols + ` FROM submissions s WHERE s.group_id = ?`
	args := []any{groupID}
	if studentID != "" {
		query += ` AND s.student_id = ?`
		args = append(args, studentID)
	}
	query += kindFilter(kind) + ` ORDER BY s.created_at DESC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SubmissionMeta{}
	for rows.Next() {
		m, err := scanMeta(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// SubmissionForTeacher devuelve un envío completo si es de una clase del docente.
func (s *Store) SubmissionForTeacher(teacherID, submissionID string) (*Submission, error) {
	sub, err := scanSubmission(s.db.QueryRow(`SELECT `+metaCols+`, `+contentCols+`
		FROM submissions s
		JOIN class_groups g ON g.id = s.group_id
		JOIN courses c ON c.id = g.course_id
		WHERE s.id = ? AND c.teacher_id = ?`, submissionID, teacherID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return sub, s.withTests(sub, true)
}

// ---------- Estudiante ----------

// StudentGroup es una clase vista por el estudiante.
type StudentGroup struct {
	ID string `json:"id"`
	Name string `json:"name"`
	CourseName string `json:"course_name"`
	TeacherName string `json:"teacher_name"`
	AIMode string `json:"ai_mode"`
}

const studentGroupSelect = `SELECT g.id, g.name, c.name, t.name, g.ai_mode
	FROM class_groups g
	JOIN courses c ON c.id = g.course_id
	JOIN users t ON t.id = c.teacher_id`

// JoinGroup inscribe al estudiante en la clase con ese código (idempotente).
func (s *Store) JoinGroup(studentID, code string) (*StudentGroup, error) {
	var groupID string
	err := s.db.QueryRow(`SELECT id FROM class_groups WHERE join_code = ?`, NormalizeJoinCode(code)).Scan(&groupID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrInvalidCode
	}
	if err != nil {
		return nil, err
	}
	if _, err := s.db.Exec(`INSERT INTO group_students (group_id, student_id, joined_at) VALUES (?, ?, ?)
		ON CONFLICT DO NOTHING`, groupID, studentID, fmtTime(time.Now())); err != nil {
		return nil, err
	}
	var sg StudentGroup
	if err := s.db.QueryRow(studentGroupSelect+` WHERE g.id = ?`, groupID).
		Scan(&sg.ID, &sg.Name, &sg.CourseName, &sg.TeacherName, &sg.AIMode); err != nil {
		return nil, err
	}
	return &sg, nil
}

// StudentGroups lista las clases del estudiante.
func (s *Store) StudentGroups(studentID string) []StudentGroup {
	out, err := s.studentGroups(studentID)
	if err != nil {
		slog.Error("listar clases del estudiante", "student_id", studentID, "error", err)
		return []StudentGroup{}
	}
	return out
}

func (s *Store) studentGroups(studentID string) ([]StudentGroup, error) {
	rows, err := s.db.Query(studentGroupSelect+`
		JOIN group_students gs ON gs.group_id = g.id
		WHERE gs.student_id = ?`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StudentGroup{}
	for rows.Next() {
		var sg StudentGroup
		if err := rows.Scan(&sg.ID, &sg.Name, &sg.CourseName, &sg.TeacherName, &sg.AIMode); err != nil {
			return nil, err
		}
		out = append(out, sg)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CourseName+out[i].Name < out[j].CourseName+out[j].Name })
	return out, rows.Err()
}

func isEnrolled(q querier, studentID, groupID string) (bool, error) {
	var n int
	err := q.QueryRow(`SELECT COUNT(*) FROM group_students WHERE group_id = ? AND student_id = ?`, groupID, studentID).Scan(&n)
	return n > 0, err
}

// IsEnrolled indica si el estudiante pertenece a la clase.
func (s *Store) IsEnrolled(studentID, groupID string) bool {
	ok, err := isEnrolled(s.db, studentID, groupID)
	return err == nil && ok
}

// ErrDuplicateAttempt indica que el intento es idéntico al anterior del estudiante (no se guarda).
var ErrDuplicateAttempt = errors.New("intento repetido")

func codeHash(code, stdin string) string {
	sum := sha256.Sum256([]byte(code + "\x00" + stdin))
	return hex.EncodeToString(sum[:])
}

// AddSubmission guarda una entrega oficial o un intento (sub.Kind). Rellena ID y CreatedAt.
// Un intento idéntico al anterior del mismo estudiante en la clase devuelve ErrDuplicateAttempt.
func (s *Store) AddSubmission(sub *Submission) error {
	if sub.Kind != KindAttempt {
		sub.Kind = KindOfficial
	}
	id, err := randomHex(12)
	if err != nil {
		return err
	}
	sub.ID = id
	sub.CreatedAt = time.Now().UTC()
	sub.CodeHash = codeHash(sub.Code, sub.Stdin)
	sub.Lines = strings.Count(strings.TrimRight(sub.Code, "\n"), "\n") + 1
	sub.HasAIFeedback = strings.TrimSpace(sub.AIFeedback) != "" || strings.TrimSpace(sub.AIFeedbackInf) != ""

	// Inscripción, deduplicación e inserción en una sola transacción.
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // no hace nada si ya hubo Commit

	ok, err := isEnrolled(tx, sub.StudentID, sub.GroupID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotEnrolled
	}
	if !sub.IsAttempt() && sub.AssignmentID != "" {
		if err := checkDeadline(tx, sub.AssignmentID, sub.GroupID, sub.CreatedAt); err != nil {
			return err
		}
	}
	if sub.IsAttempt() {
		var last string
		err := tx.QueryRow(`SELECT code_hash FROM submissions
			WHERE kind = 'attempt' AND student_id = ? AND group_id = ?
			ORDER BY created_at DESC LIMIT 1`, sub.StudentID, sub.GroupID).Scan(&last)
		if err == nil && last == sub.CodeHash {
			return ErrDuplicateAttempt
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	}
	if _, err := tx.Exec(`INSERT INTO submissions (id, kind, group_id, assignment_id, student_id, created_at,
			success, run_error, exit_code, lines, has_ai_feedback, error_summary, code_hash,
			code, stdin, compiler_output, program_output, ai_feedback, ai_feedback_informal)
		VALUES (?, ?, ?, NULLIF(?, ''), ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sub.ID, sub.Kind, sub.GroupID, sub.AssignmentID, sub.StudentID, fmtTime(sub.CreatedAt),
		sub.Success, sub.RunError, sub.ExitCode, sub.Lines, sub.HasAIFeedback, sub.ErrorSummary, sub.CodeHash,
		sub.Code, sub.Stdin, sub.CompilerOutput, sub.ProgramOutput, sub.AIFeedback, sub.AIFeedbackInf); err != nil {
		return err
	}
	if sub.AssignmentID != "" && len(sub.Tests) > 0 {
		if err := saveTestResults(tx, sub.ID, sub.Tests); err != nil {
			return err
		}
		for _, r := range sub.Tests {
			sub.TestsTotal++
			if r.Passed {
				sub.TestsPassed++
			}
		}
	}
	if !sub.IsAttempt() {
		if err := enqueueJob(tx, sub.ID, sub.CreatedAt); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// StudentSubmission es un envío propio visto por el estudiante.
type StudentSubmission struct {
	SubmissionMeta
	GroupName string `json:"group_name"`
	CourseName string `json:"course_name"`
	AssignmentTitle string `json:"assignment_title,omitempty"`
	MonitorAnnotations int `json:"monitor_annotations,omitempty"`
	Review *StudentReview `json:"review,omitempty"` // solo si el profesor ya la devolvió
	MonitorFeedback []MonitorFeedback `json:"monitor_feedback,omitempty"`
}

// StudentSubmissions lista las últimas entregas oficiales del estudiante, con la
// revisión que le devolvió su profesor (nunca la propuesta de la IA).
func (s *Store) StudentSubmissions(studentID string, limit int) []StudentSubmission {
	out, err := s.studentSubmissions(studentID, KindOfficial, limit)
	if err != nil {
		slog.Error("listar entregas del estudiante", "student_id", studentID, "error", err)
		return []StudentSubmission{}
	}
	return out
}

func (s *Store) studentSubmissions(studentID, kind string, limit int) ([]StudentSubmission, error) {
	rows, err := s.db.Query(`SELECT `+metaCols+`, g.name, c.name, COALESCE(a.title, ''),
			(SELECT COUNT(*) FROM code_annotations n JOIN users nu ON nu.id = n.author_id
				WHERE n.submission_id = s.id AND nu.role = 'monitor' AND n.sent_at IS NOT NULL),
			r.feedback, r.grade, r.returned_at, r.ai_at, COALESCE(r.feedback_origin, ''), COALESCE(rt.name, '')
		FROM submissions s
		JOIN class_groups g ON g.id = s.group_id
		JOIN courses c ON c.id = g.course_id
		LEFT JOIN assignments a ON a.id = s.assignment_id
		LEFT JOIN reviews r ON r.submission_id = s.id AND r.status = 'returned'
		LEFT JOIN users rt ON rt.id = r.teacher_id
		WHERE s.student_id = ? AND s.kind = ?
		ORDER BY s.created_at DESC LIMIT ?`, studentID, kind, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StudentSubmission{}
	for rows.Next() {
		var ss StudentSubmission
		var feedback, returnedAt, aiAt sql.NullString
		var grade sql.NullFloat64
		var origin, teacherName string
		m, err := scanMeta(rows, &ss.GroupName, &ss.CourseName, &ss.AssignmentTitle, &ss.MonitorAnnotations, &feedback, &grade, &returnedAt, &aiAt, &origin, &teacherName)
		if err != nil {
			return nil, err
		}
		ss.SubmissionMeta = m
		if feedback.Valid {
			ss.Review = &StudentReview{Feedback: feedback.String, Grade: floatPtr(grade),
				ReturnedAt: timePtr(returnedAt), AIAssisted: aiAt.Valid, Origin: origin, TeacherName: teacherName}
		}
		out = append(out, ss)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	ids := make([]string, len(out))
	for i := range out {
		ids[i] = out[i].ID
	}
	notes, err := s.monitorFeedback(ids)
	if err != nil {
		return nil, err
	}
	levels, err := s.returnedRAELevels(ids)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].MonitorFeedback = notes[out[i].ID]
		if out[i].Review != nil {
			out[i].Review.RAEs = levels[out[i].ID]
		}
	}
	return out, nil
}

// StudentProcess reúne el proceso de un estudiante en una clase del docente para analizarlo:
// sus intentos y entregas más recientes (completos, del más antiguo al más reciente).
func (s *Store) StudentProcess(teacherID, groupID, studentID string, limit int) (string, []Submission, error) {
	if _, err := s.ownedGroup(s.db, teacherID, groupID); err != nil {
		return "", nil, err
	}
	var name string
	err := s.db.QueryRow(`SELECT u.name FROM group_students gs JOIN users u ON u.id = gs.student_id
		WHERE gs.group_id = ? AND gs.student_id = ?`, groupID, studentID).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil, ErrNotFound
	}
	if err != nil {
		return "", nil, err
	}

	rows, err := s.db.Query(`SELECT `+metaCols+`, `+contentCols+` FROM submissions s
		WHERE s.group_id = ? AND s.student_id = ?
		ORDER BY s.created_at DESC LIMIT ?`, groupID, studentID, limit)
	if err != nil {
		return "", nil, err
	}
	defer rows.Close()
	out := []Submission{}
	for rows.Next() {
		sub, err := scanSubmission(rows)
		if err != nil {
			return "", nil, err
		}
		out = append(out, *sub)
	}
	if err := rows.Err(); err != nil {
		return "", nil, err
	}
	// Se pidieron los más recientes; el análisis los quiere del más antiguo al más reciente.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return name, out, nil
}
