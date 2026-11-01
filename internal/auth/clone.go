package auth

import (
	"database/sql"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const AuditCourseCloned = "materia_copiada"

type cloneFile struct {
	oldID, newID, name, contentType string
	size int64
}

type cloneAssignment struct {
	oldID, newID, groupID, title, statement, rubric string
	allowLate, evaluation bool
	penalty sql.NullFloat64
	files []cloneFile
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	return err
}

func (s *Store) CloneCourse(teacherID, courseID, name string, closeOld bool) (string, error) {
	var oldName string
	var review, batch, feedback sql.NullFloat64
	err := s.db.QueryRow(`SELECT name, review_minutes_max, batch_hours_max, feedback_hours_max FROM courses
		WHERE id = ? AND id IN (SELECT course_id FROM course_staff WHERE teacher_id = ?)`, courseID, teacherID).
		Scan(&oldName, &review, &batch, &feedback)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if name = strings.TrimSpace(name); name == "" {
		name = oldName + " (copia)"
	}

	rows, err := s.db.Query(`SELECT a.id, a.group_id, a.title, a.statement, a.rubric, a.allow_late, a.late_penalty, a.evaluation
		FROM assignments a JOIN class_groups g ON g.id = a.group_id WHERE g.course_id = ? ORDER BY a.created_at`, courseID)
	if err != nil {
		return "", err
	}
	var assignments []*cloneAssignment
	for rows.Next() {
		a := &cloneAssignment{}
		if err := rows.Scan(&a.oldID, &a.groupID, &a.title, &a.statement, &a.rubric, &a.allowLate, &a.penalty, &a.evaluation); err != nil {
			rows.Close()
			return "", err
		}
		if a.newID, err = randomHex(12); err != nil {
			rows.Close()
			return "", err
		}
		assignments = append(assignments, a)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return "", err
	}

	var dirs []string
	fail := func(err error) (string, error) {
		for _, d := range dirs {
			os.RemoveAll(d)
		}
		return "", err
	}
	for _, a := range assignments {
		frows, err := s.db.Query(`SELECT id, name, size, content_type FROM assignment_files WHERE assignment_id = ? ORDER BY position`, a.oldID)
		if err != nil {
			return fail(err)
		}
		for frows.Next() {
			var f cloneFile
			if err := frows.Scan(&f.oldID, &f.name, &f.size, &f.contentType); err != nil {
				frows.Close()
				return fail(err)
			}
			if f.newID, err = randomHex(12); err != nil {
				frows.Close()
				return fail(err)
			}
			a.files = append(a.files, f)
		}
		frows.Close()
		if len(a.files) == 0 {
			continue
		}
		dir := s.assignmentDir(a.newID)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fail(err)
		}
		dirs = append(dirs, dir)
		for _, f := range a.files {
			if err := copyFile(filepath.Join(s.assignmentDir(a.oldID), f.oldID), filepath.Join(dir, f.newID)); err != nil {
				return fail(err)
			}
		}
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fail(err)
	}
	defer tx.Rollback()
	newID, err := randomHex(12)
	if err != nil {
		return fail(err)
	}
	now := fmtTime(time.Now())
	if _, err := tx.Exec(`INSERT INTO courses (id, teacher_id, name, created_at, review_minutes_max, batch_hours_max, feedback_hours_max)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, newID, teacherID, name, now, review, batch, feedback); err != nil {
		return fail(err)
	}
	if _, err := tx.Exec(`INSERT INTO course_teachers (course_id, teacher_id, added_at)
		SELECT ?, teacher_id, ? FROM course_staff WHERE course_id = ? AND teacher_id != ?`, newID, now, courseID, teacherID); err != nil {
		return fail(err)
	}
	if _, err := tx.Exec(`INSERT INTO course_raes (id, course_id, code, description, position, created_at)
		SELECT lower(hex(randomblob(12))), ?, code, description, position, ? FROM course_raes WHERE course_id = ?`, newID, now, courseID); err != nil {
		return fail(err)
	}

	groups := map[string]string{}
	grows, err := tx.Query(`SELECT id, name, ai_mode FROM class_groups WHERE course_id = ? ORDER BY created_at`, courseID)
	if err != nil {
		return fail(err)
	}
	type group struct{ oldID, name, mode string }
	var oldGroups []group
	for grows.Next() {
		var g group
		if err := grows.Scan(&g.oldID, &g.name, &g.mode); err != nil {
			grows.Close()
			return fail(err)
		}
		oldGroups = append(oldGroups, g)
	}
	grows.Close()
	for _, g := range oldGroups {
		id, err := randomHex(12)
		if err != nil {
			return fail(err)
		}
		for try := 0; try < 10; try++ {
			code, cerr := newJoinCode()
			if cerr != nil {
				return fail(cerr)
			}
			_, err = tx.Exec(`INSERT INTO class_groups (id, course_id, name, join_code, created_at, ai_mode) VALUES (?, ?, ?, ?, ?, ?)`,
				id, newID, g.name, code, now, g.mode)
			if !isUnique(err) {
				break
			}
		}
		if err != nil {
			return fail(err)
		}
		groups[g.oldID] = id
		if err := logAudit(tx, teacherID, AuditCourseCloned, id, "", "Copiada de "+quoted(oldName)); err != nil {
			return fail(err)
		}
	}

	for _, a := range assignments {
		if _, err := tx.Exec(`INSERT INTO assignments (id, group_id, title, statement, rubric, created_at, allow_late, late_penalty, evaluation)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, a.newID, groups[a.groupID], a.title, a.statement, a.rubric, now, a.allowLate, a.penalty, a.evaluation); err != nil {
			return fail(err)
		}
		for i, f := range a.files {
			if _, err := tx.Exec(`INSERT INTO assignment_files (id, assignment_id, name, size, content_type, position) VALUES (?, ?, ?, ?, ?, ?)`,
				f.newID, a.newID, f.name, f.size, f.contentType, i); err != nil {
				return fail(err)
			}
		}
		for _, stmt := range []string{
			`INSERT INTO assignment_tests (id, assignment_id, position, name, stdin, expected, match, hidden)
				SELECT lower(hex(randomblob(12))), ?, position, name, stdin, expected, match, hidden FROM assignment_tests WHERE assignment_id = ?`,
			`INSERT INTO assignment_criteria (id, assignment_id, position, name, points)
				SELECT lower(hex(randomblob(12))), ?, position, name, points FROM assignment_criteria WHERE assignment_id = ?`,
		} {
			if _, err := tx.Exec(stmt, a.newID, a.oldID); err != nil {
				return fail(err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return fail(err)
	}
	if closeOld {
		if _, err := s.CloseCourse(teacherID, courseID); err != nil {
			return newID, err
		}
	}
	return newID, nil
}
