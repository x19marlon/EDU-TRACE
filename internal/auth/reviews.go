package auth

import (
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"time"
)

const (
	ReviewDraft    = "draft"    // el profesor aún no la devuelve
	ReviewReturned = "returned" // el estudiante ya la ve
)

// Review: propuesta de la IA (solo la ve el profesor) + decisión del profesor
// (lo que ve el estudiante cuando se devuelve).
type Review struct {
	SubmissionID string     `json:"submission_id"`
	TeacherID    string     `json:"teacher_id"`
	AIFeedback   string     `json:"ai_feedback,omitempty"`
	AIGrade      *float64   `json:"ai_grade,omitempty"`
	AIReason     string     `json:"ai_reason,omitempty"`
	AIConfidence string     `json:"ai_confidence,omitempty"`
	AIAt         *time.Time `json:"ai_at,omitempty"`
	Feedback     string     `json:"feedback"`
	Grade        *float64   `json:"grade,omitempty"`
	Status       string     `json:"status"`
	ReturnedAt   *time.Time `json:"returned_at,omitempty"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// ReviewState es el estado de revisión de una entrega, para los listados del profesor.
type ReviewState struct {
	Status string   `json:"status"`
	Grade  *float64 `json:"grade,omitempty"`
}

// GradeSource: "ai_confirmed", "profesor" o "" (sin nota).
func (r *Review) GradeSource() string {
	switch {
	case r.Grade == nil:
		return ""
	case r.AIGrade != nil && *r.AIGrade == *r.Grade:
		return "ai_confirmed"
	default:
		return "profesor"
	}
}

// StudentReview es lo único que ve el estudiante: nunca la propuesta de la IA.
type StudentReview struct {
	Feedback   string     `json:"feedback"`
	Grade      *float64   `json:"grade,omitempty"`
	ReturnedAt *time.Time `json:"returned_at"`
	AIAssisted bool       `json:"ai_assisted"` // transparencia: hubo apoyo de IA
}

// ownedOfficial comprueba que la entrega existe, es oficial y es de una clase del profesor.
func ownedOfficial(q querier, teacherID, submissionID string) error {
	var n int
	err := q.QueryRow(`SELECT COUNT(*) FROM submissions s
		JOIN class_groups g ON g.id = s.group_id
		JOIN courses c ON c.id = g.course_id
		WHERE s.id = ? AND s.kind = 'official' AND c.teacher_id = ?`, submissionID, teacherID).Scan(&n)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// loadReview devuelve la revisión guardada, o una vacía en borrador si aún no existe.
func loadReview(q querier, teacherID, submissionID string) (*Review, error) {
	r := &Review{SubmissionID: submissionID, TeacherID: teacherID, Status: ReviewDraft}
	var aiGrade, grade sql.NullFloat64
	var aiAt, returnedAt sql.NullString
	var updated string
	err := q.QueryRow(`SELECT teacher_id, ai_feedback, ai_grade, ai_reason, ai_confidence, ai_at,
			feedback, grade, status, returned_at, updated_at
		FROM reviews WHERE submission_id = ?`, submissionID).Scan(&r.TeacherID, &r.AIFeedback, &aiGrade,
		&r.AIReason, &r.AIConfidence, &aiAt, &r.Feedback, &grade, &r.Status, &returnedAt, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return r, nil
	}
	if err != nil {
		return nil, err
	}
	r.AIGrade, r.AIAt, r.Grade, r.ReturnedAt = floatPtr(aiGrade), timePtr(aiAt), floatPtr(grade), timePtr(returnedAt)
	r.UpdatedAt = parseTime(updated)
	return r, nil
}

// saveReview inserta o actualiza la revisión completa.
func saveReview(q querier, r *Review) error {
	_, err := q.Exec(`INSERT INTO reviews (submission_id, teacher_id, ai_feedback, ai_grade, ai_reason,
			ai_confidence, ai_at, feedback, grade, status, returned_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(submission_id) DO UPDATE SET
			teacher_id = excluded.teacher_id, ai_feedback = excluded.ai_feedback, ai_grade = excluded.ai_grade,
			ai_reason = excluded.ai_reason, ai_confidence = excluded.ai_confidence, ai_at = excluded.ai_at,
			feedback = excluded.feedback, grade = excluded.grade, status = excluded.status,
			returned_at = excluded.returned_at, updated_at = excluded.updated_at`,
		r.SubmissionID, r.TeacherID, r.AIFeedback, floatArg(r.AIGrade), r.AIReason, r.AIConfidence,
		fmtTimePtr(r.AIAt), r.Feedback, floatArg(r.Grade), r.Status, fmtTimePtr(r.ReturnedAt), fmtTime(r.UpdatedAt))
	return err
}

// ReviewForTeacher devuelve la revisión de una entrega oficial del profesor.
func (s *Store) ReviewForTeacher(teacherID, submissionID string) (*Review, error) {
	if err := ownedOfficial(s.db, teacherID, submissionID); err != nil {
		return nil, err
	}
	return loadReview(s.db, teacherID, submissionID)
}

// ReviewStates devuelve el estado de revisión de las entregas indicadas (las que no
// tienen revisión no aparecen en el mapa).
func (s *Store) ReviewStates(submissionIDs []string) map[string]ReviewState {
	out := map[string]ReviewState{}
	if len(submissionIDs) == 0 {
		return out
	}
	args := make([]any, len(submissionIDs))
	for i, id := range submissionIDs {
		args[i] = id
	}
	ph := strings.TrimSuffix(strings.Repeat("?, ", len(args)), ", ")
	rows, err := s.db.Query(`SELECT submission_id, status, grade FROM reviews WHERE submission_id IN (`+ph+`)`, args...)
	if err != nil {
		slog.Error("estado de revisiones", "error", err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var st ReviewState
		var grade sql.NullFloat64
		if err := rows.Scan(&id, &st.Status, &grade); err != nil {
			slog.Error("estado de revisiones", "error", err)
			return out
		}
		st.Grade = floatPtr(grade)
		out[id] = st
	}
	return out
}

// updateReview carga la revisión, la modifica con change y la guarda, todo en una transacción.
func (s *Store) updateReview(teacherID, submissionID string, change func(r *Review, now time.Time)) (*Review, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := ownedOfficial(tx, teacherID, submissionID); err != nil {
		return nil, err
	}
	r, err := loadReview(tx, teacherID, submissionID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	change(r, now)
	r.UpdatedAt = now
	if err := saveReview(tx, r); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r, nil
}

// SaveAIDraft guarda la propuesta de la IA y, si el profesor aún no escribió nada,
// precarga su formulario con ella.
func (s *Store) SaveAIDraft(teacherID, submissionID, feedback string, grade *float64, reason, confidence string) (*Review, error) {
	return s.updateReview(teacherID, submissionID, func(r *Review, now time.Time) {
		r.AIFeedback, r.AIGrade, r.AIReason, r.AIConfidence, r.AIAt = feedback, grade, reason, confidence, &now
		if r.Status != ReviewReturned && r.Feedback == "" {
			r.Feedback, r.Grade = feedback, grade
		}
	})
}

// SaveTeacherReview guarda la decisión del profesor; returnIt la hace visible al estudiante.
func (s *Store) SaveTeacherReview(teacherID, submissionID, feedback string, grade *float64, returnIt bool) (*Review, error) {
	return s.updateReview(teacherID, submissionID, func(r *Review, now time.Time) {
		r.Feedback, r.Grade = feedback, grade
		if returnIt {
			r.Status, r.ReturnedAt = ReviewReturned, &now
		}
	})
}
