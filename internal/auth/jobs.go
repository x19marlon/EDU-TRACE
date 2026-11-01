package auth

import (
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"time"
)

const (
	JobPending = "pendiente"
	JobRunning = "procesando"
	JobDone = "lista"
	JobFailed = "error"
	JobSkipped = "omitida"
	MaxJobAttempts = 3
)

type AIJob struct {
	SubmissionID string
	TeacherID string
	Attempts int
}

func enqueueJob(q querier, submissionID string, at time.Time) error {
	_, err := q.Exec(`INSERT INTO ai_jobs (submission_id, status, enqueued_at) VALUES (?, 'pendiente', ?)
		ON CONFLICT(submission_id) DO NOTHING`, submissionID, fmtTime(at))
	return err
}

func (s *Store) ResetRunningJobs() error {
	_, err := s.db.Exec(`UPDATE ai_jobs SET status = 'pendiente' WHERE status = 'procesando'`)
	return err
}

func (s *Store) ClaimJob(retryAfter time.Duration) (*AIJob, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	var j AIJob
	err = tx.QueryRow(`SELECT j.submission_id, c.teacher_id, j.attempts
		FROM ai_jobs j
		JOIN submissions s ON s.id = j.submission_id
		JOIN class_groups g ON g.id = s.group_id
		JOIN courses c ON c.id = g.course_id
		WHERE j.status = 'pendiente' OR (j.status = 'error' AND j.attempts < ? AND j.finished_at < ?)
		ORDER BY j.enqueued_at LIMIT 1`, MaxJobAttempts, fmtTime(now.Add(-retryAfter))).Scan(&j.SubmissionID, &j.TeacherID, &j.Attempts)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`UPDATE ai_jobs SET status = 'procesando', attempts = attempts + 1, started_at = ?, error = ''
		WHERE submission_id = ?`, fmtTime(now), j.SubmissionID); err != nil {
		return nil, err
	}
	j.Attempts++
	return &j, tx.Commit()
}

func (s *Store) FinishJob(submissionID, status, errMsg string, final bool) {
	attempts := "attempts"
	if final {
		attempts = "MAX(attempts, ?)"
	}
	args := []any{status, errMsg, fmtTime(time.Now())}
	if final {
		args = append(args, MaxJobAttempts)
	}
	args = append(args, submissionID)
	if _, err := s.db.Exec(`UPDATE ai_jobs SET status = ?, error = ?, finished_at = ?, attempts = `+attempts+`
		WHERE submission_id = ?`, args...); err != nil {
		slog.Error("cerrar trabajo de IA", "submission_id", submissionID, "error", err)
	}
}

func (s *Store) NeedsAIDraft(submissionID string) (bool, error) {
	var status string
	var aiAt, discarded sql.NullString
	err := s.db.QueryRow(`SELECT status, ai_at, ai_discarded_at FROM reviews WHERE submission_id = ?`, submissionID).
		Scan(&status, &aiAt, &discarded)
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return status != ReviewReturned && !aiAt.Valid && !discarded.Valid, nil
}

func (s *Store) JobStates(submissionIDs []string) map[string]string {
	out := map[string]string{}
	if len(submissionIDs) == 0 {
		return out
	}
	args, ph := inArgs(submissionIDs)
	rows, err := s.db.Query(`SELECT submission_id, status FROM ai_jobs WHERE submission_id IN (`+ph+`)`, args...)
	if err != nil {
		slog.Error("estado de la cola de IA", "error", err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, st string
		if err := rows.Scan(&id, &st); err != nil {
			return out
		}
		out[id] = st
	}
	return out
}

type JobState struct {
	Status string `json:"status"`
	Attempts int `json:"attempts"`
	Error string `json:"error,omitempty"`
	EnqueuedAt time.Time `json:"enqueued_at"`
	Ahead int `json:"ahead"`
}

func (s *Store) JobFor(teacherID, submissionID string) (*JobState, error) {
	if err := ownedOfficial(s.db, teacherID, submissionID); err != nil {
		return nil, err
	}
	var j JobState
	var enq string
	err := s.db.QueryRow(`SELECT status, attempts, error, enqueued_at,
			(SELECT COUNT(*) FROM ai_jobs o WHERE o.status IN ('pendiente', 'procesando') AND o.enqueued_at < j.enqueued_at)
		FROM ai_jobs j WHERE submission_id = ?`, submissionID).Scan(&j.Status, &j.Attempts, &j.Error, &enq, &j.Ahead)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	j.EnqueuedAt = parseTime(enq)
	return &j, nil
}

type FailedJobs struct {
	GroupID string `json:"group_id"`
	GroupName string `json:"group_name"`
	CourseName string `json:"course_name"`
	Failed int `json:"failed"`
	LastError string `json:"last_error"`
	LastAt *time.Time `json:"last_at,omitempty"`
}

type QueueOverview struct {
	Pending int `json:"pending"`
	Running int `json:"running"`
	Failed []FailedJobs `json:"failed"`
}

func (s *Store) TeacherQueue(teacherID string) (*QueueOverview, error) {
	out := &QueueOverview{Failed: []FailedJobs{}}
	err := s.db.QueryRow(`SELECT COALESCE(SUM(j.status = 'pendiente'), 0), COALESCE(SUM(j.status = 'procesando'), 0)
		FROM ai_jobs j
		JOIN submissions s ON s.id = j.submission_id
		JOIN class_groups g ON g.id = s.group_id
		JOIN courses c ON c.id = g.course_id
		WHERE c.id IN (SELECT course_id FROM course_staff WHERE teacher_id = ?)`, teacherID).Scan(&out.Pending, &out.Running)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT g.id, g.name, c.name, COUNT(*), MAX(j.finished_at),
			(SELECT j2.error FROM ai_jobs j2 JOIN submissions s2 ON s2.id = j2.submission_id
				WHERE s2.group_id = g.id AND j2.status = 'error' ORDER BY j2.finished_at DESC LIMIT 1)
		FROM ai_jobs j
		JOIN submissions s ON s.id = j.submission_id
		JOIN class_groups g ON g.id = s.group_id
		JOIN courses c ON c.id = g.course_id
		WHERE c.id IN (SELECT course_id FROM course_staff WHERE teacher_id = ?) AND j.status = 'error' AND j.attempts >= ?
		GROUP BY g.id ORDER BY MAX(j.finished_at) DESC`, teacherID, MaxJobAttempts)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var f FailedJobs
		var last sql.NullString
		if err := rows.Scan(&f.GroupID, &f.GroupName, &f.CourseName, &f.Failed, &last, &f.LastError); err != nil {
			return nil, err
		}
		f.LastAt = timePtr(last)
		out.Failed = append(out.Failed, f)
	}
	return out, rows.Err()
}

func (s *Store) RetryFailedJobs(teacherID, groupID string) (int64, error) {
	if _, err := s.ownedGroup(s.db, teacherID, groupID); err != nil {
		return 0, err
	}
	res, err := s.db.Exec(`UPDATE ai_jobs SET status = 'pendiente', attempts = 0, error = ''
		WHERE status = 'error' AND submission_id IN (SELECT id FROM submissions WHERE group_id = ?)`, groupID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func JobError(err error) string {
	msg := strings.TrimSpace(err.Error())
	if len(msg) > 300 {
		msg = strings.ToValidUTF8(msg[:300], "")
	}
	return msg
}

func (s *Store) QueueStalled(now time.Time, after time.Duration) (bool, error) {
	var oldest, last sql.NullString
	if err := s.db.QueryRow(`SELECT (SELECT MIN(enqueued_at) FROM ai_jobs WHERE status IN ('pendiente', 'procesando')),
		(SELECT MAX(finished_at) FROM ai_jobs)`).Scan(&oldest, &last); err != nil {
		return false, err
	}
	cutoff := now.Add(-after)
	if !oldest.Valid || parseTime(oldest.String).After(cutoff) {
		return false, nil
	}
	return !last.Valid || parseTime(last.String).Before(cutoff), nil
}
