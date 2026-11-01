package auth

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	AnnotationHighlight = "highlight"
	AnnotationStrike = "strike"
	AnnotationGood = "good"
	AnnotationNote = "note"
	AnnotationDrawing = "drawing"

	maxAnnotationComment = 2000
	maxDrawingStrokes = 100
	maxStrokePoints = 2000
	maxDrawingColumns = 400
)

var (
	ErrInvalidAnnotation = errors.New("anotación inválida")
	penColors = map[string]bool{"#D64545": true, "#7154CC": true, "#3A8F6F": true}
)

type Annotation struct {
	ID string `json:"id"`
	SubmissionID string `json:"submission_id"`
	AuthorID string `json:"author_id"`
	AuthorName string `json:"author_name"`
	AuthorRole Role `json:"author_role"`
	Kind string `json:"kind"`
	LineStart int `json:"line_start,omitempty"`
	LineEnd int `json:"line_end,omitempty"`
	Comment string `json:"comment,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
	SentAt *time.Time `json:"sent_at,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type AnnotationInput struct {
	Kind string `json:"kind"`
	LineStart int `json:"line_start"`
	LineEnd int `json:"line_end"`
	Comment string `json:"comment"`
	Data json.RawMessage `json:"data"`
}

type drawing struct {
	Strokes []stroke `json:"strokes"`
}

type stroke struct {
	Color string `json:"color"`
	Width float64 `json:"width"`
	Points [][2]float64 `json:"points"`
}

func normalizeDrawing(raw json.RawMessage, lines int) (json.RawMessage, error) {
	var d drawing
	if err := json.Unmarshal(raw, &d); err != nil || len(d.Strokes) == 0 || len(d.Strokes) > maxDrawingStrokes {
		return nil, ErrInvalidAnnotation
	}
	for i, s := range d.Strokes {
		if !penColors[s.Color] || s.Width < 1 || s.Width > 8 || len(s.Points) == 0 || len(s.Points) > maxStrokePoints {
			return nil, ErrInvalidAnnotation
		}
		for _, p := range s.Points {
			if p[0] < -2 || p[0] > maxDrawingColumns || p[1] < -1 || p[1] > float64(lines)+1 {
				return nil, ErrInvalidAnnotation
			}
		}
		d.Strokes[i] = s
	}
	return json.Marshal(d)
}

func (s *Store) annotationTarget(u *User, submissionID string) (int, error) {
	var lines int
	var err error
	switch u.Role {
	case RoleTeacher:
		err = s.db.QueryRow(`SELECT s.lines FROM submissions s
			JOIN class_groups g ON g.id = s.group_id
			JOIN courses c ON c.id = g.course_id
			WHERE s.id = ? AND c.id IN (SELECT course_id FROM course_staff WHERE teacher_id = ?)`, submissionID, u.ID).Scan(&lines)
	case RoleMonitor:
		err = s.db.QueryRow(`SELECT s.lines FROM submissions s
			JOIN group_monitors gm ON gm.group_id = s.group_id
			WHERE s.id = ? AND gm.monitor_id = ? AND s.kind = 'official'`, submissionID, u.ID).Scan(&lines)
	default:
		return 0, ErrNotFound
	}
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	return lines, err
}

func (s *Store) AddAnnotation(u *User, submissionID string, in AnnotationInput) (*Annotation, error) {
	lines, err := s.annotationTarget(u, submissionID)
	if err != nil {
		return nil, err
	}
	a := &Annotation{
		SubmissionID: submissionID, AuthorID: u.ID, AuthorName: u.Name, AuthorRole: u.Role,
		Kind: in.Kind, Comment: strings.TrimSpace(in.Comment), CreatedAt: time.Now().UTC(),
	}
	if utf8.RuneCountInString(a.Comment) > maxAnnotationComment {
		return nil, ErrInvalidAnnotation
	}
	switch in.Kind {
	case AnnotationHighlight, AnnotationStrike, AnnotationGood:
		if in.LineStart < 1 || in.LineEnd < in.LineStart || in.LineEnd > lines {
			return nil, ErrInvalidAnnotation
		}
		a.LineStart, a.LineEnd = in.LineStart, in.LineEnd
	case AnnotationNote:
		if a.Comment == "" {
			return nil, ErrInvalidAnnotation
		}
	case AnnotationDrawing:
		if a.Data, err = normalizeDrawing(in.Data, lines); err != nil {
			return nil, err
		}
	default:
		return nil, ErrInvalidAnnotation
	}
	if a.ID, err = randomHex(12); err != nil {
		return nil, err
	}
	var lineStart, lineEnd any
	if a.LineStart > 0 {
		lineStart, lineEnd = a.LineStart, a.LineEnd
	}
	if _, err := s.db.Exec(`INSERT INTO code_annotations (id, submission_id, author_id, kind, line_start, line_end, comment, data, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.SubmissionID, a.AuthorID, a.Kind, lineStart, lineEnd, a.Comment, string(a.Data), fmtTime(a.CreatedAt)); err != nil {
		return nil, err
	}
	return a, nil
}

const annotationSelect = `SELECT a.id, a.submission_id, a.author_id, u.name, u.role, a.kind,
		COALESCE(a.line_start, 0), COALESCE(a.line_end, 0), a.comment, a.data, a.sent_at, a.created_at
	FROM code_annotations a JOIN users u ON u.id = a.author_id`

func (s *Store) queryAnnotations(where string, args ...any) ([]Annotation, error) {
	rows, err := s.db.Query(annotationSelect+` WHERE `+where+` ORDER BY a.created_at`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Annotation{}
	for rows.Next() {
		var a Annotation
		var data string
		var sent sql.NullString
		var created string
		if err := rows.Scan(&a.ID, &a.SubmissionID, &a.AuthorID, &a.AuthorName, &a.AuthorRole, &a.Kind,
			&a.LineStart, &a.LineEnd, &a.Comment, &data, &sent, &created); err != nil {
			return nil, err
		}
		if data != "" {
			a.Data = json.RawMessage(data)
		}
		a.SentAt, a.CreatedAt = timePtr(sent), parseTime(created)
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) AnnotationsFor(u *User, submissionID string) []Annotation {
	var list []Annotation
	var err error
	switch u.Role {
	case RoleTeacher:
		if _, err = s.annotationTarget(u, submissionID); err == nil {
			list, err = s.queryAnnotations(`a.submission_id = ?`, submissionID)
		}
	case RoleMonitor:
		if _, err = s.annotationTarget(u, submissionID); err == nil {
			list, err = s.queryAnnotations(`a.submission_id = ? AND (a.author_id = ? OR (u.role = 'monitor' AND a.sent_at IS NOT NULL))`, submissionID, u.ID)
		}
	case RoleStudent:
		list, err = s.queryAnnotations(`a.submission_id = ? AND u.role = 'monitor' AND a.sent_at IS NOT NULL
			AND EXISTS (SELECT 1 FROM submissions s WHERE s.id = a.submission_id AND s.student_id = ?)`, submissionID, u.ID)
	}
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			slog.Error("cargar anotaciones", "submission_id", submissionID, "error", err)
		}
		return []Annotation{}
	}
	if list == nil {
		return []Annotation{}
	}
	return list
}

func (s *Store) DeleteAnnotation(u *User, annotationID string) error {
	switch u.Role {
	case RoleTeacher:
		return affected(s.db.Exec(`DELETE FROM code_annotations WHERE id = ? AND submission_id IN (
			SELECT s.id FROM submissions s
			JOIN class_groups g ON g.id = s.group_id
			JOIN courses c ON c.id = g.course_id
			WHERE c.id IN (SELECT course_id FROM course_staff WHERE teacher_id = ?))`, annotationID, u.ID))
	case RoleMonitor:
		return affected(s.db.Exec(`DELETE FROM code_annotations WHERE id = ? AND author_id = ?`, annotationID, u.ID))
	}
	return ErrNotFound
}

func (s *Store) SendMonitorAnnotations(monitor *User, submissionID string) (int64, time.Time, error) {
	now := time.Now().UTC()
	if monitor.Role != RoleMonitor {
		return 0, now, ErrNotFound
	}
	if _, err := s.annotationTarget(monitor, submissionID); err != nil {
		return 0, now, err
	}
	res, err := s.db.Exec(`UPDATE code_annotations SET sent_at = ?
		WHERE submission_id = ? AND author_id = ? AND sent_at IS NULL`, fmtTime(now), submissionID, monitor.ID)
	if err != nil {
		return 0, now, err
	}
	n, err := res.RowsAffected()
	if err != nil || n == 0 {
		return n, now, err
	}
	detail := "Envió 1 marca al estudiante"
	if n > 1 {
		detail = fmt.Sprintf("Envió %d marcas al estudiante", n)
	}
	if err := logAudit(s.db, monitor.ID, AuditAnnotationsSent, "", submissionID, detail); err != nil {
		return n, now, err
	}
	return n, now, notifyStudent(s.db, submissionID, NoticeAnnotations, "Un monitor marcó el código de")
}
