package auth

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const MaxAssignmentFiles = 10

var (
	ErrMonitorNotFound = errors.New("no hay una cuenta de monitor con ese email")
	ErrTooManyFiles = fmt.Errorf("máximo %d archivos por taller", MaxAssignmentFiles)
)

type GroupMonitor struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Email string `json:"email"`
}

func affected(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func queryIDs(q querier, query string, args ...any) ([]string, error) {
	rows, err := q.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func inArgs(ids []string) ([]any, string) {
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	return args, strings.TrimSuffix(strings.Repeat("?, ", len(ids)), ", ")
}

func (s *Store) removeAssignmentDirs(ids []string) {
	for _, id := range ids {
		os.RemoveAll(s.assignmentDir(id))
	}
}

func writeUploads(dir string, files []FileUpload) ([]AssignmentFile, error) {
	out := []AssignmentFile{}
	undo := func(err error) ([]AssignmentFile, error) {
		for _, f := range out {
			os.Remove(filepath.Join(dir, f.ID))
		}
		return nil, err
	}
	for _, f := range files {
		fid, err := randomHex(12)
		if err != nil {
			return undo(err)
		}
		dst, err := os.OpenFile(filepath.Join(dir, fid), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return undo(err)
		}
		n, err := io.Copy(dst, f.Body)
		if cerr := dst.Close(); err == nil {
			err = cerr
		}
		out = append(out, AssignmentFile{ID: fid, Name: safeFileName(f.Name), Size: n, ContentType: f.ContentType})
		if err != nil {
			return undo(fmt.Errorf("guardar %q: %w", f.Name, err))
		}
	}
	return out, nil
}

func (s *Store) RenameCourse(teacherID, courseID, name string) error {
	return affected(s.db.Exec(`UPDATE courses SET name = ? WHERE id = ? AND teacher_id = ?`,
		strings.TrimSpace(name), courseID, teacherID))
}

func (s *Store) DeleteCourse(teacherID, courseID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	ids, err := queryIDs(tx, `SELECT a.id FROM assignments a
		JOIN class_groups g ON g.id = a.group_id
		JOIN courses c ON c.id = g.course_id
		WHERE c.id = ? AND c.teacher_id = ?`, courseID, teacherID)
	if err != nil {
		return err
	}
	if err := affected(tx.Exec(`DELETE FROM courses WHERE id = ? AND teacher_id = ?`, courseID, teacherID)); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.removeAssignmentDirs(ids)
	return nil
}

func (s *Store) RenameGroup(teacherID, groupID, name string) error {
	return affected(s.db.Exec(`UPDATE class_groups SET name = ?
		WHERE id = ? AND course_id IN (SELECT id FROM courses WHERE teacher_id = ?)`,
		strings.TrimSpace(name), groupID, teacherID))
}

func (s *Store) DeleteGroup(teacherID, groupID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := s.ownedGroup(tx, teacherID, groupID); err != nil {
		return err
	}
	ids, err := queryIDs(tx, `SELECT id FROM assignments WHERE group_id = ?`, groupID)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM class_groups WHERE id = ?`, groupID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.removeAssignmentDirs(ids)
	return nil
}

func (s *Store) RegenerateJoinCode(teacherID, groupID string) (string, error) {
	if _, err := s.ownedGroup(s.db, teacherID, groupID); err != nil {
		return "", err
	}
	for try := 0; try < 10; try++ {
		code, err := newJoinCode()
		if err != nil {
			return "", err
		}
		_, err = s.db.Exec(`UPDATE class_groups SET join_code = ? WHERE id = ?`, code, groupID)
		if err == nil {
			return code, nil
		}
		if !isUnique(err) {
			return "", err
		}
	}
	return "", errors.New("no se pudo generar un código nuevo")
}

func (s *Store) RemoveStudent(teacherID, groupID, studentID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := s.ownedGroup(tx, teacherID, groupID); err != nil {
		return err
	}
	if err := affected(tx.Exec(`DELETE FROM group_students WHERE group_id = ? AND student_id = ?`, groupID, studentID)); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM submissions WHERE group_id = ? AND student_id = ?`, groupID, studentID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) AddMonitor(teacherID, groupID, email string) (*GroupMonitor, error) {
	if _, err := s.ownedGroup(s.db, teacherID, groupID); err != nil {
		return nil, err
	}
	m := &GroupMonitor{}
	err := s.db.QueryRow(`SELECT id, name, email FROM users WHERE email = ? AND role = 'monitor'`,
		normalizeEmail(email)).Scan(&m.ID, &m.Name, &m.Email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrMonitorNotFound
	}
	if err != nil {
		return nil, err
	}
	if _, err := s.db.Exec(`INSERT INTO group_monitors (group_id, monitor_id, added_at) VALUES (?, ?, ?)
		ON CONFLICT DO NOTHING`, groupID, m.ID, fmtTime(time.Now())); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *Store) RemoveMonitor(teacherID, groupID, monitorID string) error {
	if _, err := s.ownedGroup(s.db, teacherID, groupID); err != nil {
		return err
	}
	return affected(s.db.Exec(`DELETE FROM group_monitors WHERE group_id = ? AND monitor_id = ?`, groupID, monitorID))
}

func (s *Store) groupMonitors(groupID string) ([]GroupMonitor, error) {
	rows, err := s.db.Query(`SELECT u.id, u.name, u.email FROM group_monitors gm
		JOIN users u ON u.id = gm.monitor_id
		WHERE gm.group_id = ? ORDER BY gm.added_at`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []GroupMonitor{}
	for rows.Next() {
		var m GroupMonitor
		if err := rows.Scan(&m.ID, &m.Name, &m.Email); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) ownedAssignment(q querier, teacherID, assignmentID string) error {
	var n int
	if err := q.QueryRow(`SELECT COUNT(*) FROM assignments a
		JOIN class_groups g ON g.id = a.group_id
		JOIN courses c ON c.id = g.course_id
		WHERE a.id = ? AND c.teacher_id = ?`, assignmentID, teacherID).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) UpdateAssignment(teacherID, assignmentID, title, statement, rubric string, dueAt *time.Time, policy LatePolicy, removeFiles []string, files []FileUpload) (*Assignment, error) {
	if err := policy.validate(dueAt); err != nil {
		return nil, err
	}
	if err := s.ownedAssignment(s.db, teacherID, assignmentID); err != nil {
		return nil, err
	}
	dir := s.assignmentDir(assignmentID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	written, err := writeUploads(dir, files)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*Assignment, error) {
		for _, f := range written {
			os.Remove(filepath.Join(dir, f.ID))
		}
		return nil, err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fail(err)
	}
	defer tx.Rollback()
	if err := s.ownedAssignment(tx, teacherID, assignmentID); err != nil {
		return fail(err)
	}
	if _, err := tx.Exec(`UPDATE assignments SET title = ?, statement = ?, rubric = ?, due_at = ?,
			allow_late = ?, late_until = ?, late_penalty = ? WHERE id = ?`,
		strings.TrimSpace(title), strings.TrimSpace(statement), strings.TrimSpace(rubric), fmtTimePtr(dueAt),
		policy.AllowLate, fmtTimePtr(policy.LateUntil), floatArg(policy.LatePenalty), assignmentID); err != nil {
		return fail(err)
	}
	var removed []string
	if len(removeFiles) > 0 {
		args, ph := inArgs(removeFiles)
		args = append([]any{assignmentID}, args...)
		if removed, err = queryIDs(tx, `SELECT id FROM assignment_files WHERE assignment_id = ? AND id IN (`+ph+`)`, args...); err != nil {
			return fail(err)
		}
		if _, err := tx.Exec(`DELETE FROM assignment_files WHERE assignment_id = ? AND id IN (`+ph+`)`, args...); err != nil {
			return fail(err)
		}
	}
	var count, last int
	if err := tx.QueryRow(`SELECT COUNT(*), COALESCE(MAX(position), -1) FROM assignment_files WHERE assignment_id = ?`,
		assignmentID).Scan(&count, &last); err != nil {
		return fail(err)
	}
	if count+len(written) > MaxAssignmentFiles {
		return fail(ErrTooManyFiles)
	}
	for i, f := range written {
		if _, err := tx.Exec(`INSERT INTO assignment_files (id, assignment_id, name, size, content_type, position)
			VALUES (?, ?, ?, ?, ?, ?)`, f.ID, assignmentID, f.Name, f.Size, f.ContentType, last+1+i); err != nil {
			return fail(err)
		}
	}
	a, err := scanAssignment(tx.QueryRow(`SELECT `+assignmentCols+` FROM assignments a WHERE a.id = ?`, assignmentID))
	if err != nil {
		return fail(err)
	}
	if err := tx.Commit(); err != nil {
		return fail(err)
	}
	for _, id := range removed {
		os.Remove(filepath.Join(dir, id))
	}
	list := []Assignment{a}
	if err := s.attachFiles(list); err != nil {
		return nil, err
	}
	return &list[0], nil
}

func (s *Store) DeleteAssignment(teacherID, assignmentID string) error {
	if err := affected(s.db.Exec(`DELETE FROM assignments WHERE id = ? AND group_id IN (
		SELECT g.id FROM class_groups g JOIN courses c ON c.id = g.course_id WHERE c.teacher_id = ?)`,
		assignmentID, teacherID)); err != nil {
		return err
	}
	s.removeAssignmentDirs([]string{assignmentID})
	return nil
}
