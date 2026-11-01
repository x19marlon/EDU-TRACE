package auth

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

const (
	NoticeReviewReturned = "revision_devuelta"
	NoticeReviewUpdated = "revision_actualizada"
	NoticeQuestionAnswered = "pregunta_respondida"
	NoticeQuestionAsked = "pregunta_nueva"
	NoticeMonitorFeedback = "comentario_monitor"
	NoticeAnnotations = "marcas_monitor"
	NoticeResetCode = "codigo_restablecimiento"
	NoticeCoTeacher = "profesor_agregado"
	notificationsShown = 30
	maxNoticeTitle = 200
)

type Notification struct {
	ID int64 `json:"id"`
	Kind string `json:"kind"`
	Title string `json:"title"`
	Link string `json:"link,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	ReadAt *time.Time `json:"read_at,omitempty"`
}

type NotificationList struct {
	Items []Notification `json:"items"`
	Unread int `json:"unread"`
	Email bool `json:"email"`
}

type PendingEmail struct {
	ID int64
	UserID string
	Name string
	Email string
	Title string
	Link string
}

func notify(q querier, userID, kind, title, link string) error {
	if userID == "" {
		return nil
	}
	_, err := q.Exec(`INSERT INTO notifications (user_id, kind, title, link, created_at) VALUES (?, ?, ?, ?, ?)`,
		userID, kind, truncateText(title, maxNoticeTitle), link, fmtTime(time.Now()))
	return err
}

func submissionNotice(q querier, submissionID string) (studentID, teacherID, title string, err error) {
	err = q.QueryRow(`SELECT s.student_id, c.teacher_id, COALESCE(a.title, '')
		FROM submissions s
		JOIN class_groups g ON g.id = s.group_id
		JOIN courses c ON c.id = g.course_id
		LEFT JOIN assignments a ON a.id = s.assignment_id
		WHERE s.id = ?`, submissionID).Scan(&studentID, &teacherID, &title)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	return
}

func notifyStudent(q querier, submissionID, kind, prefix string) error {
	studentID, _, title, err := submissionNotice(q, submissionID)
	if err != nil {
		return err
	}
	if title == "" {
		return notify(q, studentID, kind, prefix+" tu entrega.", "/historial")
	}
	return notify(q, studentID, kind, prefix+" la entrega de "+quoted(title)+".", "/historial")
}

func (s *Store) Notifications(userID string) (*NotificationList, error) {
	out := &NotificationList{Items: []Notification{}}
	if err := s.db.QueryRow(`SELECT email_notifications, (SELECT COUNT(*) FROM notifications WHERE user_id = ? AND read_at IS NULL)
		FROM users WHERE id = ?`, userID, userID).Scan(&out.Email, &out.Unread); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	rows, err := s.db.Query(`SELECT id, kind, title, link, created_at, read_at FROM notifications
		WHERE user_id = ? ORDER BY id DESC LIMIT ?`, userID, notificationsShown)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var n Notification
		var created string
		var read sql.NullString
		if err := rows.Scan(&n.ID, &n.Kind, &n.Title, &n.Link, &created, &read); err != nil {
			return nil, err
		}
		n.CreatedAt, n.ReadAt = parseTime(created), timePtr(read)
		out.Items = append(out.Items, n)
	}
	return out, rows.Err()
}

func (s *Store) MarkNotificationsRead(userID string, ids []int64) error {
	now := fmtTime(time.Now())
	if len(ids) == 0 {
		_, err := s.db.Exec(`UPDATE notifications SET read_at = ? WHERE user_id = ? AND read_at IS NULL`, now, userID)
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, id := range ids {
		if _, err := tx.Exec(`UPDATE notifications SET read_at = ? WHERE id = ? AND user_id = ? AND read_at IS NULL`, now, id, userID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) SetEmailNotifications(userID string, on bool) error {
	return affected(s.db.Exec(`UPDATE users SET email_notifications = ? WHERE id = ?`, on, userID))
}

func (s *Store) PendingEmails(createdBefore, createdAfter time.Time, limit int) ([]PendingEmail, error) {
	rows, err := s.db.Query(`SELECT n.id, n.user_id, u.name, u.email, n.title, n.link
		FROM notifications n JOIN users u ON u.id = n.user_id
		WHERE n.emailed_at IS NULL AND n.read_at IS NULL AND u.email_notifications = 1
			AND n.created_at <= ? AND n.created_at >= ?
		ORDER BY n.user_id, n.id LIMIT ?`, fmtTime(createdBefore), fmtTime(createdAfter), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PendingEmail
	for rows.Next() {
		var p PendingEmail
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.Email, &p.Title, &p.Link); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) MarkEmailed(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	args := []any{fmtTime(time.Now())}
	for _, id := range ids {
		args = append(args, id)
	}
	_, err := s.db.Exec(`UPDATE notifications SET emailed_at = ? WHERE id IN (?`+strings.Repeat(", ?", len(ids)-1)+`)`, args...)
	return err
}
