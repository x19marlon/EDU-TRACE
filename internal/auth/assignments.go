package auth

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// AssignmentFile es un archivo adjunto a un taller.
type AssignmentFile struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
}

// Assignment es un taller que el profesor deja en una clase.
type Assignment struct {
	ID        string           `json:"id"`
	GroupID   string           `json:"group_id"`
	Title     string           `json:"title"`
	Statement string           `json:"statement"`
	DueAt     *time.Time       `json:"due_at,omitempty"`
	CreatedAt time.Time        `json:"created_at"`
	Files     []AssignmentFile `json:"files"`
}

// FileUpload es un archivo recibido para guardar en un taller.
type FileUpload struct {
	Name        string
	ContentType string
	Body        io.Reader
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
func (s *Store) CreateAssignment(teacherID, groupID, title, statement string, dueAt *time.Time, files []FileUpload) (*Assignment, error) {
	s.mu.Lock()
	_, _, err := s.ownedGroupLocked(teacherID, groupID)
	s.mu.Unlock()
	if err != nil {
		return nil, err
	}

	id, err := randomHex(12)
	if err != nil {
		return nil, err
	}
	a := &Assignment{
		ID: id, GroupID: groupID,
		Title: strings.TrimSpace(title), Statement: strings.TrimSpace(statement),
		DueAt: dueAt, CreatedAt: time.Now().UTC(), Files: []AssignmentFile{},
	}

	// Los archivos se escriben fuera del lock; si algo falla se borra la carpeta.
	dir := s.assignmentDir(id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	for _, f := range files {
		fid, err := randomHex(12)
		if err != nil {
			os.RemoveAll(dir)
			return nil, err
		}
		out, err := os.OpenFile(filepath.Join(dir, fid), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			os.RemoveAll(dir)
			return nil, err
		}
		n, err := io.Copy(out, f.Body)
		if cerr := out.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			os.RemoveAll(dir)
			return nil, fmt.Errorf("guardar %q: %w", f.Name, err)
		}
		a.Files = append(a.Files, AssignmentFile{ID: fid, Name: safeFileName(f.Name), Size: n, ContentType: f.ContentType})
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Assignments[id] = a
	if err := s.saveLocked(); err != nil {
		delete(s.data.Assignments, id)
		os.RemoveAll(dir)
		return nil, err
	}
	cp := *a
	return &cp, nil
}

func sortAssignments(list []Assignment) {
	sort.Slice(list, func(i, j int) bool { return list[i].CreatedAt.After(list[j].CreatedAt) })
}

// GroupAssignments lista los talleres de una clase del profesor.
func (s *Store) GroupAssignments(teacherID, groupID string) ([]Assignment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, _, err := s.ownedGroupLocked(teacherID, groupID); err != nil {
		return nil, err
	}
	out := []Assignment{}
	for _, a := range s.data.Assignments {
		if a.GroupID == groupID {
			out = append(out, *a)
		}
	}
	sortAssignments(out)
	return out, nil
}

// StudentAssignment es un taller visto por el estudiante.
type StudentAssignment struct {
	Assignment
	GroupName  string `json:"group_name"`
	CourseName string `json:"course_name"`
}

// StudentAssignments lista los talleres de todas las clases del estudiante.
func (s *Store) StudentAssignments(studentID string) []StudentAssignment {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []StudentAssignment{}
	for _, a := range s.data.Assignments {
		g, ok := s.data.Groups[a.GroupID]
		if !ok || !containsID(g.StudentIDs, studentID) {
			continue
		}
		sg := s.studentGroupLocked(g)
		out = append(out, StudentAssignment{Assignment: *a, GroupName: sg.Name, CourseName: sg.CourseName})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func containsID(ids []string, id string) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

// canSeeAssignmentLocked: el profesor dueño de la materia o un estudiante inscrito en la clase.
func (s *Store) canSeeAssignmentLocked(u *User, a *Assignment) bool {
	g, ok := s.data.Groups[a.GroupID]
	if !ok {
		return false
	}
	switch u.Role {
	case RoleTeacher:
		c, ok := s.data.Courses[g.CourseID]
		return ok && c.TeacherID == u.ID
	case RoleStudent:
		return containsID(g.StudentIDs, u.ID)
	}
	return false
}

// AssignmentStatement devuelve el enunciado si el usuario puede ver el taller.
func (s *Store) AssignmentStatement(u *User, assignmentID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.data.Assignments[assignmentID]
	if !ok || !s.canSeeAssignmentLocked(u, a) {
		return "", ErrNotFound
	}
	return a.Title + "\n\n" + a.Statement, nil
}

// AssignmentInGroup indica si el taller pertenece a la clase.
func (s *Store) AssignmentInGroup(assignmentID, groupID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.data.Assignments[assignmentID]
	return ok && a.GroupID == groupID
}

// AssignmentFilePath devuelve la ruta en disco y los datos de un adjunto si el usuario puede verlo.
func (s *Store) AssignmentFilePath(u *User, assignmentID, fileID string) (string, AssignmentFile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.data.Assignments[assignmentID]
	if !ok || !s.canSeeAssignmentLocked(u, a) {
		return "", AssignmentFile{}, ErrNotFound
	}
	for _, f := range a.Files {
		if f.ID == fileID {
			return filepath.Join(s.assignmentDir(a.ID), f.ID), f, nil
		}
	}
	return "", AssignmentFile{}, ErrNotFound
}
