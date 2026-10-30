package auth

import (
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AssignmentFile es un archivo adjunto a un taller.
type AssignmentFile struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Size int64 `json:"size"`
	ContentType string `json:"content_type"`
}

// Assignment es un taller que el profesor deja en una clase.
type Assignment struct {
	ID string `json:"id"`
	GroupID string `json:"group_id"`
	Title string `json:"title"`
	Statement string `json:"statement"`
	Rubric string `json:"rubric,omitempty"`
	DueAt *time.Time `json:"due_at,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	Files []AssignmentFile `json:"files"`
	AllowLate bool `json:"allow_late"`
	LateUntil *time.Time `json:"late_until,omitempty"`
	LatePenalty *float64 `json:"late_penalty,omitempty"`
}

// FileUpload es un archivo recibido para guardar en un taller.
type FileUpload struct {
	Name string
	ContentType string
	Body io.Reader
}

func (s *Store) assignmentDir(id string) string {
	return filepath.Join(s.dir, "assignments", id)
}

// safeFileName conserva solo el nombre base y quita caracteres de control,
// para mostrarlo y usarlo en Content-Disposition sin riesgos.
func safeFileName(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || r == '"' || r == '/' {
			return -1
		}
		return r
	}, name)
	if name == "" || name == "." || name == ".." {
		name = "archivo"
	}
	if len(name) > 120 {
		name = name[:120]
	}
	return strings.ToValidUTF8(name, "")
}

// CreateAssignment crea un taller en una clase del profesor y guarda sus archivos.
func (s *Store) CreateAssignment(teacherID, groupID, title, statement, rubric string, dueAt *time.Time, policy LatePolicy, files []FileUpload) (*Assignment, error) {
	if err := policy.validate(dueAt); err != nil {
		return nil, err
	}
	if _, err := s.ownedGroup(s.db, teacherID, groupID); err != nil {
		return nil, err
	}
	id, err := randomHex(12)
	if err != nil {
		return nil, err
	}
	a := &Assignment{
		ID: id, GroupID: groupID,
		Title: strings.TrimSpace(title), Statement: strings.TrimSpace(statement), Rubric: strings.TrimSpace(rubric),
		DueAt: dueAt, CreatedAt: time.Now().UTC(), Files: []AssignmentFile{},
		AllowLate: policy.AllowLate, LateUntil: policy.LateUntil, LatePenalty: policy.LatePenalty,
	}

	// Primero los archivos en disco; si algo falla después, se borra la carpeta.
	dir := s.assignmentDir(id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	fail := func(err error) (*Assignment, error) {
		os.RemoveAll(dir)
		return nil, err
	}
	written, err := writeUploads(dir, files)
	if err != nil {
		return fail(err)
	}
	a.Files = written

	// El taller y sus archivos se registran juntos o no se registra nada.
	tx, err := s.db.Begin()
	if err != nil {
		return fail(err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT INTO assignments (id, group_id, title, statement, rubric, due_at, created_at, allow_late, late_until, late_penalty)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, a.ID, a.GroupID, a.Title, a.Statement, a.Rubric, fmtTimePtr(a.DueAt), fmtTime(a.CreatedAt),
		a.AllowLate, fmtTimePtr(a.LateUntil), floatArg(a.LatePenalty)); err != nil {
		return fail(err)
	}
	for i, f := range a.Files {
		if _, err := tx.Exec(`INSERT INTO assignment_files (id, assignment_id, name, size, content_type, position)
			VALUES (?, ?, ?, ?, ?, ?)`, f.ID, a.ID, f.Name, f.Size, f.ContentType, i); err != nil {
			return fail(err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fail(err)
	}
	return a, nil
}

// Columnas de assignments (alias "a"), en el orden que espera scanAssignment.
const assignmentCols = `a.id, a.group_id, a.title, a.statement, a.rubric, a.due_at, a.created_at, a.allow_late, a.late_until, a.late_penalty`

func scanAssignment(sc scanner, extra ...any) (Assignment, error) {
	var a Assignment
	var due, lateUntil sql.NullString
	var created string
	var penalty sql.NullFloat64
	dest := append([]any{&a.ID, &a.GroupID, &a.Title, &a.Statement, &a.Rubric, &due, &created, &a.AllowLate, &lateUntil, &penalty}, extra...)
	if err := sc.Scan(dest...); err != nil {
		return a, err
	}
	a.DueAt, a.CreatedAt, a.Files = timePtr(due), parseTime(created), []AssignmentFile{}
	a.LateUntil, a.LatePenalty = timePtr(lateUntil), floatPtr(penalty)
	return a, nil
}

// attachFiles completa la lista de archivos de cada taller (después de cerrar la
// consulta de talleres, para no tener dos consultas abiertas a la vez).
func (s *Store) attachFiles(list []Assignment) error {
	for i := range list {
		rows, err := s.db.Query(`SELECT id, name, size, content_type FROM assignment_files
			WHERE assignment_id = ? ORDER BY position`, list[i].ID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var f AssignmentFile
			if err := rows.Scan(&f.ID, &f.Name, &f.Size, &f.ContentType); err != nil {
				rows.Close()
				return err
			}
			list[i].Files = append(list[i].Files, f)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
	}
	return nil
}

// GroupAssignments lista los talleres de una clase del profesor.
func (s *Store) GroupAssignments(teacherID, groupID string) ([]Assignment, error) {
	if _, err := s.ownedGroup(s.db, teacherID, groupID); err != nil {
		return nil, err
	}
	return s.groupAssignments(groupID)
}

func (s *Store) groupAssignments(groupID string) ([]Assignment, error) {
	rows, err := s.db.Query(`SELECT `+assignmentCols+` FROM assignments a
		WHERE a.group_id = ? ORDER BY a.created_at DESC`, groupID)
	if err != nil {
		return nil, err
	}
	out := []Assignment{}
	for rows.Next() {
		a, err := scanAssignment(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, a)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, s.attachFiles(out)
}

// StudentAssignment es un taller visto por el estudiante.
type StudentAssignment struct {
	Assignment
	GroupName string `json:"group_name"`
	CourseName string `json:"course_name"`
}

// StudentAssignments lista los talleres de todas las clases del estudiante.
func (s *Store) StudentAssignments(studentID string) []StudentAssignment {
	out, err := s.studentAssignments(studentID)
	if err != nil {
		slog.Error("listar talleres del estudiante", "student_id", studentID, "error", err)
		return []StudentAssignment{}
	}
	return out
}

func (s *Store) studentAssignments(studentID string) ([]StudentAssignment, error) {
	rows, err := s.db.Query(`SELECT `+assignmentCols+`, g.name, c.name
		FROM assignments a
		JOIN group_students gs ON gs.group_id = a.group_id
		JOIN class_groups g ON g.id = a.group_id
		JOIN courses c ON c.id = g.course_id
		WHERE gs.student_id = ? ORDER BY a.created_at DESC`, studentID)
	if err != nil {
		return nil, err
	}
	var list []Assignment
	var names [][2]string
	for rows.Next() {
		var groupName, courseName string
		a, err := scanAssignment(rows, &groupName, &courseName)
		if err != nil {
			rows.Close()
			return nil, err
		}
		list = append(list, a)
		names = append(names, [2]string{groupName, courseName})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := s.attachFiles(list); err != nil {
		return nil, err
	}
	out := make([]StudentAssignment, len(list))
	for i, a := range list {
		out[i] = StudentAssignment{Assignment: a, GroupName: names[i][0], CourseName: names[i][1]}
	}
	return out, nil
}

// visibleAssignment devuelve título y enunciado si el usuario puede ver el taller:
// el profesor dueño de la materia o un estudiante inscrito en la clase.
func (s *Store) visibleAssignment(u *User, assignmentID string) (title, statement string, err error) {
	err = s.db.QueryRow(`SELECT a.title, a.statement FROM assignments a
		JOIN class_groups g ON g.id = a.group_id
		JOIN courses c ON c.id = g.course_id
		WHERE a.id = ? AND (
			(? = 'teacher' AND c.teacher_id = ?) OR
			(? = 'student' AND EXISTS (SELECT 1 FROM group_students gs
				WHERE gs.group_id = a.group_id AND gs.student_id = ?)) OR
			(? = 'monitor' AND EXISTS (SELECT 1 FROM group_monitors gm
				WHERE gm.group_id = a.group_id AND gm.monitor_id = ?))
		)`, assignmentID, u.Role, u.ID, u.Role, u.ID, u.Role, u.ID).Scan(&title, &statement)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	return
}

// AssignmentStatement devuelve el enunciado si el usuario puede ver el taller.
func (s *Store) AssignmentStatement(u *User, assignmentID string) (string, error) {
	title, statement, err := s.visibleAssignment(u, assignmentID)
	if err != nil {
		return "", err
	}
	return title + "\n" + statement, nil
}

// AssignmentForGrading devuelve el enunciado y los criterios de un taller de una clase del profesor.
func (s *Store) AssignmentForGrading(teacherID, assignmentID string) (statement, rubric string, err error) {
	var title, body string
	err = s.db.QueryRow(`SELECT a.title, a.statement, a.rubric FROM assignments a
		JOIN class_groups g ON g.id = a.group_id
		JOIN courses c ON c.id = g.course_id
		WHERE a.id = ? AND c.teacher_id = ?`, assignmentID, teacherID).Scan(&title, &body, &rubric)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", ErrNotFound
	}
	if err != nil {
		return "", "", err
	}
	return title + "\n" + body, rubric, nil
}

// AssignmentInGroup indica si el taller pertenece a la clase.
func (s *Store) AssignmentInGroup(assignmentID, groupID string) bool {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM assignments WHERE id = ? AND group_id = ?`, assignmentID, groupID).Scan(&n)
	return err == nil && n > 0
}

// AssignmentFilePath devuelve la ruta en disco y los datos de un adjunto si el usuario puede verlo.
func (s *Store) AssignmentFilePath(u *User, assignmentID, fileID string) (string, AssignmentFile, error) {
	if _, _, err := s.visibleAssignment(u, assignmentID); err != nil {
		return "", AssignmentFile{}, err
	}
	f := AssignmentFile{ID: fileID}
	err := s.db.QueryRow(`SELECT name, size, content_type FROM assignment_files WHERE id = ? AND assignment_id = ?`,
		fileID, assignmentID).Scan(&f.Name, &f.Size, &f.ContentType)
	if errors.Is(err, sql.ErrNoRows) {
		return "", AssignmentFile{}, ErrNotFound
	}
	if err != nil {
		return "", AssignmentFile{}, err
	}
	return filepath.Join(s.assignmentDir(assignmentID), fileID), f, nil
}
