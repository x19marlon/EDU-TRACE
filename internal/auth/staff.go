package auth

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

const (
	AuditTeacherAdded = "profesor_agregado"
	AuditTeacherRemoved = "profesor_retirado"
)

var (
	ErrTeacherNotFound = errors.New("no hay una cuenta de profesor con ese email; pídele que se registre como profesor")
	ErrAlreadyTeacher = errors.New("esa persona ya es profesora de la materia")
)

type CourseTeacher struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Email string `json:"email"`
	Owner bool `json:"owner"`
}

func courseOwner(q querier, courseID string) (string, string, error) {
	var owner, name string
	err := q.QueryRow(`SELECT teacher_id, name FROM courses WHERE id = ?`, courseID).Scan(&owner, &name)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", ErrNotFound
	}
	return owner, name, err
}

func logCourseAudit(q querier, actorID, courseID, action, detail string) error {
	groups, err := queryIDs(q, `SELECT id FROM class_groups WHERE course_id = ?`, courseID)
	if err != nil {
		return err
	}
	for _, g := range groups {
		if err := logAudit(q, actorID, action, g, "", detail); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) AddCoTeacher(ownerID, courseID, email string) (*CourseTeacher, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	owner, course, err := courseOwner(tx, courseID)
	if err != nil {
		return nil, err
	}
	if owner != ownerID {
		return nil, ErrNotFound
	}
	t := &CourseTeacher{}
	err = tx.QueryRow(`SELECT id, name, email FROM users WHERE email = ? AND role = 'teacher'`, normalizeEmail(email)).Scan(&t.ID, &t.Name, &t.Email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTeacherNotFound
	}
	if err != nil {
		return nil, err
	}
	if t.ID == owner {
		return nil, ErrAlreadyTeacher
	}
	res, err := tx.Exec(`INSERT INTO course_teachers (course_id, teacher_id, added_at) VALUES (?, ?, ?) ON CONFLICT DO NOTHING`,
		courseID, t.ID, fmtTime(time.Now()))
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ErrAlreadyTeacher
	}
	if err := logCourseAudit(tx, ownerID, courseID, AuditTeacherAdded, "Agregó a "+t.Name+" como profesor de la materia"); err != nil {
		return nil, err
	}
	if err := notify(tx, t.ID, NoticeCoTeacher, "Ahora eres profesor de la materia "+quoted(course)+".", "/docente"); err != nil {
		return nil, err
	}
	return t, tx.Commit()
}

func (s *Store) RemoveCoTeacher(actorID, courseID, teacherID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	owner, _, err := courseOwner(tx, courseID)
	if err != nil {
		return err
	}
	if actorID != owner && actorID != teacherID {
		return ErrNotFound
	}
	var name string
	if err := tx.QueryRow(`SELECT name FROM users WHERE id = ?`, teacherID).Scan(&name); err != nil {
		return ErrNotFound
	}
	if err := affected(tx.Exec(`DELETE FROM course_teachers WHERE course_id = ? AND teacher_id = ?`, courseID, teacherID)); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE reviews SET teacher_id = ? WHERE teacher_id = ? AND submission_id IN (
		SELECT s.id FROM submissions s JOIN class_groups g ON g.id = s.group_id WHERE g.course_id = ?)`, owner, teacherID, courseID); err != nil {
		return err
	}
	detail := "Retiró a " + name + " de la materia"
	if actorID == teacherID {
		detail = name + " dejó la materia"
	}
	if err := logCourseAudit(tx, actorID, courseID, AuditTeacherRemoved, detail); err != nil {
		return err
	}
	return tx.Commit()
}

func courseTeachers(q querier, courseIDs []string) (map[string][]CourseTeacher, error) {
	out := map[string][]CourseTeacher{}
	if len(courseIDs) == 0 {
		return out, nil
	}
	args, ph := inArgs(courseIDs)
	rows, err := q.Query(`SELECT st.course_id, u.id, u.name, u.email, u.id = c.teacher_id
		FROM course_staff st JOIN users u ON u.id = st.teacher_id JOIN courses c ON c.id = st.course_id
		WHERE st.course_id IN (`+ph+`) ORDER BY u.id = c.teacher_id DESC, u.name`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var courseID string
		var t CourseTeacher
		if err := rows.Scan(&courseID, &t.ID, &t.Name, &t.Email, &t.Owner); err != nil {
			return nil, err
		}
		out[courseID] = append(out[courseID], t)
	}
	return out, rows.Err()
}

func courseStaffIDs(q querier, submissionID string) ([]string, error) {
	return queryIDs(q, `SELECT st.teacher_id FROM course_staff st
		JOIN class_groups g ON g.course_id = st.course_id
		JOIN submissions s ON s.group_id = g.id
		WHERE s.id = ?`, submissionID)
}

func staffNotice(q querier, submissionID, kind, title, link string) error {
	ids, err := courseStaffIDs(q, submissionID)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := notify(q, id, kind, strings.TrimSpace(title), link); err != nil {
			return err
		}
	}
	return nil
}
