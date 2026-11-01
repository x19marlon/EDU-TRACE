package auth

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

const (
	MaxQuestionsPerSubmission = 20
	AuditQuestionAnswered = "pregunta_respondida"
)

var ErrTooManyQuestions = errors.New("ya hiciste demasiadas preguntas sobre esta entrega; habla con tu profesor en clase")

type Question struct {
	ID string `json:"id"`
	SubmissionID string `json:"submission_id"`
	Body string `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	Answer string `json:"answer,omitempty"`
	AnsweredBy string `json:"answered_by,omitempty"`
	AnsweredRole string `json:"answered_role,omitempty"`
	AnsweredAt *time.Time `json:"answered_at,omitempty"`
}

const questionCols = `id, submission_id, body, created_at, answer, answered_by_name, answered_role, answered_at`

func scanQuestion(sc scanner) (Question, error) {
	var q Question
	var created string
	var answered sql.NullString
	if err := sc.Scan(&q.ID, &q.SubmissionID, &q.Body, &created, &q.Answer, &q.AnsweredBy, &q.AnsweredRole, &answered); err != nil {
		return q, err
	}
	q.CreatedAt, q.AnsweredAt = parseTime(created), timePtr(answered)
	return q, nil
}

func (s *Store) AskQuestion(studentID, submissionID, body string) (*Question, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var n, asked int
	if err := tx.QueryRow(`SELECT COUNT(*), (SELECT COUNT(*) FROM review_questions WHERE submission_id = ?)
		FROM submissions WHERE id = ? AND student_id = ? AND kind = 'official'`, submissionID, submissionID, studentID).Scan(&n, &asked); err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, ErrNotFound
	}
	if asked >= MaxQuestionsPerSubmission {
		return nil, ErrTooManyQuestions
	}
	id, err := randomHex(12)
	if err != nil {
		return nil, err
	}
	q := &Question{ID: id, SubmissionID: submissionID, Body: strings.TrimSpace(body), CreatedAt: time.Now().UTC()}
	if _, err := tx.Exec(`INSERT INTO review_questions (id, submission_id, body, created_at) VALUES (?, ?, ?, ?)`,
		q.ID, q.SubmissionID, q.Body, fmtTime(q.CreatedAt)); err != nil {
		return nil, err
	}
	_, _, title, err := submissionNotice(tx, submissionID)
	if err != nil {
		return nil, err
	}
	notice := "Un estudiante hizo una pregunta sobre su entrega."
	if title != "" {
		notice = "Un estudiante hizo una pregunta sobre " + quoted(title) + "."
	}
	if err := staffNotice(tx, submissionID, NoticeQuestionAsked, notice, "/docente"); err != nil {
		return nil, err
	}
	return q, tx.Commit()
}

func (s *Store) AnswerQuestion(u *User, questionID, answer string) (*Question, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var submissionID, body string
	err = tx.QueryRow(`SELECT q.submission_id, q.body FROM review_questions q
		JOIN submissions s ON s.id = q.submission_id
		JOIN class_groups g ON g.id = s.group_id
		JOIN courses c ON c.id = g.course_id
		WHERE q.id = ? AND (
			(? = 'teacher' AND c.id IN (SELECT course_id FROM course_staff WHERE teacher_id = ?)) OR
			(? = 'monitor' AND EXISTS (SELECT 1 FROM group_monitors gm WHERE gm.group_id = g.id AND gm.monitor_id = ?))
		)`, questionID, u.Role, u.ID, u.Role, u.ID).Scan(&submissionID, &body)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if _, err := tx.Exec(`UPDATE review_questions SET answer = ?, answered_by = ?, answered_by_name = ?, answered_role = ?, answered_at = ?
		WHERE id = ?`, strings.TrimSpace(answer), u.ID, u.Name, string(u.Role), fmtTime(now), questionID); err != nil {
		return nil, err
	}
	detail := "Respondió la pregunta «" + truncateText(body, 80) + "»"
	if err := logAudit(tx, u.ID, AuditQuestionAnswered, "", submissionID, detail); err != nil {
		return nil, err
	}
	if err := notifyStudent(tx, submissionID, NoticeQuestionAnswered, "Respondieron tu pregunta sobre"); err != nil {
		return nil, err
	}
	q, err := scanQuestion(tx.QueryRow(`SELECT `+questionCols+` FROM review_questions WHERE id = ?`, questionID))
	if err != nil {
		return nil, err
	}
	return &q, tx.Commit()
}

func truncateText(s string, max int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= max {
		return string(r)
	}
	return string(r[:max]) + "…"
}

func (s *Store) questionsFor(ids []string) (map[string][]Question, error) {
	out := map[string][]Question{}
	if len(ids) == 0 {
		return out, nil
	}
	args, ph := inArgs(ids)
	rows, err := s.db.Query(`SELECT `+questionCols+` FROM review_questions WHERE submission_id IN (`+ph+`) ORDER BY created_at`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		q, err := scanQuestion(rows)
		if err != nil {
			return nil, err
		}
		out[q.SubmissionID] = append(out[q.SubmissionID], q)
	}
	return out, rows.Err()
}

func (s *Store) QuestionsFor(submissionID string) []Question {
	m, err := s.questionsFor([]string{submissionID})
	if err != nil {
		return []Question{}
	}
	if m[submissionID] == nil {
		return []Question{}
	}
	return m[submissionID]
}

func (s *Store) OpenQuestionCounts(ids []string) map[string]int {
	out := map[string]int{}
	if len(ids) == 0 {
		return out
	}
	args, ph := inArgs(ids)
	rows, err := s.db.Query(`SELECT submission_id, COUNT(*) FROM review_questions
		WHERE answered_at IS NULL AND submission_id IN (`+ph+`) GROUP BY submission_id`, args...)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var n int
		if rows.Scan(&id, &n) == nil {
			out[id] = n
		}
	}
	return out
}

type OpenQuestion struct {
	Question
	GroupID string `json:"group_id"`
	GroupName string `json:"group_name"`
	StudentName string `json:"student_name"`
	AssignmentTitle string `json:"assignment_title,omitempty"`
}

func (s *Store) TeacherOpenQuestions(teacherID string) ([]OpenQuestion, error) {
	rows, err := s.db.Query(`SELECT q.id, q.submission_id, q.body, q.created_at, q.answer, q.answered_by_name, q.answered_role, q.answered_at,
			g.id, g.name, u.name, COALESCE(a.title, '')
		FROM review_questions q
		JOIN submissions s ON s.id = q.submission_id
		JOIN users u ON u.id = s.student_id
		JOIN class_groups g ON g.id = s.group_id
		JOIN courses c ON c.id = g.course_id
		LEFT JOIN assignments a ON a.id = s.assignment_id
		WHERE c.id IN (SELECT course_id FROM course_staff WHERE teacher_id = ?) AND q.answered_at IS NULL
		ORDER BY q.created_at LIMIT 100`, teacherID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []OpenQuestion{}
	for rows.Next() {
		var o OpenQuestion
		var created string
		var answered sql.NullString
		if err := rows.Scan(&o.ID, &o.SubmissionID, &o.Body, &created, &o.Answer, &o.AnsweredBy, &o.AnsweredRole, &answered,
			&o.GroupID, &o.GroupName, &o.StudentName, &o.AssignmentTitle); err != nil {
			return nil, err
		}
		o.CreatedAt, o.AnsweredAt = parseTime(created), timePtr(answered)
		out = append(out, o)
	}
	return out, rows.Err()
}
