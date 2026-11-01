package auth

import (
	"database/sql"
	"errors"
	"time"
)

const (
	DefaultRetention = 180 * 24 * time.Hour
	noticeRetention = 90 * 24 * time.Hour
	aiRequestRetention = 365 * 24 * time.Hour
	AuditCourseClosed = "periodo_cerrado"
	AuditCourseReopened = "periodo_reabierto"
	NoticeCourseClosed = "materia_cerrada"
)

var ErrCourseClosed = errors.New("esta materia ya cerró su periodo: no recibe entregas ni inscripciones")

type PurgeReport struct {
	Courses int
	Accounts int
	Notifications int64
	AIRequests int64
	Sessions int64
}

func (r PurgeReport) Empty() bool {
	return r.Courses == 0 && r.Accounts == 0 && r.Notifications == 0 && r.AIRequests == 0 && r.Sessions == 0
}

func (s *Store) SetRetention(d time.Duration) { s.retention = max(d, 0) }

func (s *Store) purgeDate(closed *time.Time) *time.Time {
	if closed == nil || s.retention == 0 {
		return nil
	}
	t := closed.Add(s.retention)
	return &t
}

func courseClosed(q querier, groupID string) (bool, error) {
	var closed sql.NullString
	err := q.QueryRow(`SELECT c.closed_at FROM class_groups g JOIN courses c ON c.id = g.course_id WHERE g.id = ?`, groupID).Scan(&closed)
	if errors.Is(err, sql.ErrNoRows) {
		return false, ErrNotFound
	}
	return closed.Valid, err
}

func dateText(t time.Time) string { return t.In(time.Local).Format("02/01/2006") }

func (s *Store) CloseCourse(teacherID, courseID string) (*time.Time, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var name string
	var closed sql.NullString
	err = tx.QueryRow(`SELECT name, closed_at FROM courses WHERE id = ? AND id IN (SELECT course_id FROM course_staff WHERE teacher_id = ?)`, courseID, teacherID).Scan(&name, &closed)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if closed.Valid {
		return s.purgeDate(timePtr(closed)), nil
	}
	now := time.Now().UTC()
	purge := s.purgeDate(&now)
	if _, err := tx.Exec(`UPDATE courses SET closed_at = ? WHERE id = ?`, fmtTime(now), courseID); err != nil {
		return nil, err
	}
	detail, notice := "Cerró el periodo", "La materia "+quoted(name)+" cerró su periodo."
	if purge != nil {
		detail += "; los datos se borran el " + dateText(*purge)
		notice += " Tus entregas se borrarán el " + dateText(*purge) + "; si quieres conservarlas, descarga tus datos en «Mi cuenta»."
	}
	groups, err := queryIDs(tx, `SELECT id FROM class_groups WHERE course_id = ?`, courseID)
	if err != nil {
		return nil, err
	}
	for _, g := range groups {
		if err := logAudit(tx, teacherID, AuditCourseClosed, g, "", detail); err != nil {
			return nil, err
		}
	}
	students, err := queryIDs(tx, `SELECT DISTINCT gs.student_id FROM group_students gs
		JOIN class_groups g ON g.id = gs.group_id WHERE g.course_id = ?`, courseID)
	if err != nil {
		return nil, err
	}
	for _, id := range students {
		if err := notify(tx, id, NoticeCourseClosed, notice, "/cuenta"); err != nil {
			return nil, err
		}
	}
	return purge, tx.Commit()
}

func (s *Store) ReopenCourse(teacherID, courseID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := affected(tx.Exec(`UPDATE courses SET closed_at = NULL WHERE id = ? AND id IN (SELECT course_id FROM course_staff WHERE teacher_id = ?) AND closed_at IS NOT NULL`, courseID, teacherID)); err != nil {
		return err
	}
	groups, err := queryIDs(tx, `SELECT id FROM class_groups WHERE course_id = ?`, courseID)
	if err != nil {
		return err
	}
	for _, g := range groups {
		if err := logAudit(tx, teacherID, AuditCourseReopened, g, "", "Reabrió el periodo; ya no se borrarán los datos"); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) purgeCourse(courseID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	ids, err := queryIDs(tx, `SELECT a.id FROM assignments a JOIN class_groups g ON g.id = a.group_id WHERE g.course_id = ?`, courseID)
	if err != nil {
		return err
	}
	if err := affected(tx.Exec(`DELETE FROM courses WHERE id = ?`, courseID)); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.removeAssignmentDirs(ids)
	return nil
}

func (s *Store) purgeAccount(userID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := deleteUser(tx, userID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Purge(now time.Time) (PurgeReport, error) {
	var r PurgeReport
	if s.retention == 0 {
		return r, nil
	}
	cutoff := fmtTime(now.Add(-s.retention))
	courses, err := queryIDs(s.db, `SELECT id FROM courses WHERE closed_at IS NOT NULL AND closed_at <= ?`, cutoff)
	if err != nil {
		return r, err
	}
	for _, id := range courses {
		if err := s.purgeCourse(id); err != nil {
			return r, err
		}
		r.Courses++
	}
	accounts, err := queryIDs(s.db, `SELECT u.id FROM users u
		WHERE u.role IN ('student', 'monitor') AND COALESCE(u.last_active_at, u.created_at) <= ?
			AND NOT EXISTS (SELECT 1 FROM group_students gs WHERE gs.student_id = u.id)
			AND NOT EXISTS (SELECT 1 FROM group_monitors gm WHERE gm.monitor_id = u.id)`, cutoff)
	if err != nil {
		return r, err
	}
	for _, id := range accounts {
		if err := s.purgeAccount(id); err != nil {
			return r, err
		}
		r.Accounts++
	}
	for _, del := range []struct {
		count *int64
		query string
		before time.Time
	}{
		{&r.Notifications, `DELETE FROM notifications WHERE created_at <= ?`, now.Add(-noticeRetention)},
		{&r.AIRequests, `DELETE FROM ai_requests WHERE at <= ?`, now.Add(-aiRequestRetention)},
		{&r.Sessions, `DELETE FROM sessions WHERE expires_at <= ?`, now},
	} {
		res, err := s.db.Exec(del.query, fmtTime(del.before))
		if err != nil {
			return r, err
		}
		if *del.count, err = res.RowsAffected(); err != nil {
			return r, err
		}
	}
	return r, nil
}
