package auth

import (
	"database/sql"
	"errors"
	"log/slog"
	"sort"
	"strings"
	"time"
)

var ErrAttemptFeedback = errors.New("el monitor solo retroalimenta entregas oficiales")

type MonitorGroup struct {
	ID string `json:"id"`
	Name string `json:"name"`
	CourseName string `json:"course_name"`
	TeacherName string `json:"teacher_name"`
	StudentCount int `json:"student_count"`
	SubmissionCount int `json:"submission_count"`
}

type MonitorFeedback struct {
	ID string `json:"id"`
	SubmissionID string `json:"submission_id"`
	MonitorID string `json:"monitor_id"`
	MonitorName string `json:"monitor_name"`
	Body string `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Store) MonitorGroups(monitorID string) []MonitorGroup {
	out, err := s.monitorGroups(monitorID)
	if err != nil {
		slog.Error("listar clases del monitor", "monitor_id", monitorID, "error", err)
		return []MonitorGroup{}
	}
	return out
}

func (s *Store) monitorGroups(monitorID string) ([]MonitorGroup, error) {
	rows, err := s.db.Query(`SELECT g.id, g.name, c.name, t.name,
			(SELECT COUNT(*) FROM group_students gs WHERE gs.group_id = g.id),
			(SELECT COUNT(*) FROM submissions s WHERE s.group_id = g.id AND s.kind = 'official')
		FROM group_monitors gm
		JOIN class_groups g ON g.id = gm.group_id
		JOIN courses c ON c.id = g.course_id
		JOIN users t ON t.id = c.teacher_id
		WHERE gm.monitor_id = ?`, monitorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []MonitorGroup{}
	for rows.Next() {
		var m MonitorGroup
		if err := rows.Scan(&m.ID, &m.Name, &m.CourseName, &m.TeacherName, &m.StudentCount, &m.SubmissionCount); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CourseName+out[i].Name < out[j].CourseName+out[j].Name })
	return out, rows.Err()
}

func (s *Store) assignedGroup(q querier, monitorID, groupID string) (*GroupDetail, error) {
	g := &GroupDetail{ID: groupID, Students: []GroupStudent{}, Monitors: []GroupMonitor{}}
	err := q.QueryRow(`SELECT g.name, c.id, c.name
		FROM group_monitors gm
		JOIN class_groups g ON g.id = gm.group_id
		JOIN courses c ON c.id = g.course_id
		WHERE gm.group_id = ? AND gm.monitor_id = ?`, groupID, monitorID).Scan(&g.Name, &g.CourseID, &g.CourseName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return g, nil
}

func (s *Store) GroupForMonitor(monitorID, groupID string) (*GroupDetail, error) {
	g, err := s.assignedGroup(s.db, monitorID, groupID)
	if err != nil {
		return nil, err
	}
	if err := s.fillGroup(g); err != nil {
		return nil, err
	}
	for i := range g.Students {
		g.Students[i].Email = ""
	}
	for i := range g.Monitors {
		g.Monitors[i].Email = ""
	}
	return g, nil
}

func (s *Store) MonitorGroupSubmissions(monitorID, groupID, studentID, kind string) ([]SubmissionMeta, error) {
	if _, err := s.assignedGroup(s.db, monitorID, groupID); err != nil {
		return nil, err
	}
	return s.groupSubmissions(groupID, studentID, kind)
}

func (s *Store) MonitorGroupAssignments(monitorID, groupID string) ([]Assignment, error) {
	if _, err := s.assignedGroup(s.db, monitorID, groupID); err != nil {
		return nil, err
	}
	return s.groupAssignments(groupID, false)
}

func (s *Store) SubmissionForMonitor(monitorID, submissionID string) (*Submission, error) {
	sub, err := scanSubmission(s.db.QueryRow(`SELECT `+metaCols+`, `+contentCols+`
		FROM submissions s
		JOIN group_monitors gm ON gm.group_id = s.group_id
		WHERE s.id = ? AND gm.monitor_id = ?`, submissionID, monitorID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return sub, s.withTests(sub, false)
}

func (s *Store) AddMonitorFeedback(monitorID, submissionID, body string) (*MonitorFeedback, error) {
	f := &MonitorFeedback{SubmissionID: submissionID, MonitorID: monitorID, Body: strings.TrimSpace(body), CreatedAt: time.Now().UTC()}
	var kind string
	err := s.db.QueryRow(`SELECT s.kind, u.name FROM submissions s
		JOIN group_monitors gm ON gm.group_id = s.group_id
		JOIN users u ON u.id = gm.monitor_id
		WHERE s.id = ? AND gm.monitor_id = ?`, submissionID, monitorID).Scan(&kind, &f.MonitorName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if kind != KindOfficial {
		return nil, ErrAttemptFeedback
	}
	if f.ID, err = randomHex(12); err != nil {
		return nil, err
	}
	if _, err := s.db.Exec(`INSERT INTO monitor_feedback (id, submission_id, monitor_id, body, created_at) VALUES (?, ?, ?, ?, ?)`,
		f.ID, f.SubmissionID, f.MonitorID, f.Body, fmtTime(f.CreatedAt)); err != nil {
		return nil, err
	}
	return f, nil
}

func (s *Store) DeleteMonitorFeedback(u *User, feedbackID string) error {
	switch u.Role {
	case RoleMonitor:
		return affected(s.db.Exec(`DELETE FROM monitor_feedback WHERE id = ? AND monitor_id = ?`, feedbackID, u.ID))
	case RoleTeacher:
		return affected(s.db.Exec(`DELETE FROM monitor_feedback WHERE id = ? AND submission_id IN (
			SELECT s.id FROM submissions s
			JOIN class_groups g ON g.id = s.group_id
			JOIN courses c ON c.id = g.course_id
			WHERE c.teacher_id = ?)`, feedbackID, u.ID))
	}
	return ErrNotFound
}

func (s *Store) MonitorFeedbackFor(submissionID string) []MonitorFeedback {
	notes, err := s.monitorFeedback([]string{submissionID})
	if err != nil {
		slog.Error("cargar retroalimentación del monitor", "submission_id", submissionID, "error", err)
	}
	if notes[submissionID] == nil {
		return []MonitorFeedback{}
	}
	return notes[submissionID]
}

func (s *Store) MonitorFeedbackCounts(submissionIDs []string) map[string]int {
	out := map[string]int{}
	if len(submissionIDs) == 0 {
		return out
	}
	args, ph := inArgs(submissionIDs)
	rows, err := s.db.Query(`SELECT submission_id, COUNT(*) FROM monitor_feedback
		WHERE submission_id IN (`+ph+`) GROUP BY submission_id`, args...)
	if err != nil {
		slog.Error("contar retroalimentación del monitor", "error", err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return out
		}
		out[id] = n
	}
	return out
}

func (s *Store) monitorFeedback(submissionIDs []string) (map[string][]MonitorFeedback, error) {
	out := map[string][]MonitorFeedback{}
	if len(submissionIDs) == 0 {
		return out, nil
	}
	args, ph := inArgs(submissionIDs)
	rows, err := s.db.Query(`SELECT f.id, f.submission_id, f.monitor_id, u.name, f.body, f.created_at
		FROM monitor_feedback f JOIN users u ON u.id = f.monitor_id
		WHERE f.submission_id IN (`+ph+`) ORDER BY f.created_at`, args...)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var f MonitorFeedback
		var created string
		if err := rows.Scan(&f.ID, &f.SubmissionID, &f.MonitorID, &f.MonitorName, &f.Body, &created); err != nil {
			return out, err
		}
		f.CreatedAt = parseTime(created)
		out[f.SubmissionID] = append(out[f.SubmissionID], f)
	}
	return out, rows.Err()
}
