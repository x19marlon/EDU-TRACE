package auth

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

const (
	ReviewDraft = "draft" 
	ReviewReturned = "returned" 
	OriginAI = "ia"
	OriginAIEdited = "ia_editada"
	OriginTeacher = "docente"
	maxReviewMs = 2 * 60 * 60 * 1000
)

var (
	ErrAlreadyReturned = errors.New("la revisión ya se devolvió al estudiante")
	ErrNoAIDraft = errors.New("no hay una propuesta de la IA para descartar")
)

// Review: propuesta de la IA (solo la ve el profesor) 
type Review struct {
	SubmissionID string `json:"submission_id"`
	TeacherID string `json:"teacher_id"`
	AIFeedback string `json:"ai_feedback,omitempty"`
	AIGrade *float64 `json:"ai_grade,omitempty"`
	AIReason string `json:"ai_reason,omitempty"`
	AIConfidence string `json:"ai_confidence,omitempty"`
	AIAt *time.Time `json:"ai_at,omitempty"`
	Feedback string `json:"feedback"`
	Grade *float64 `json:"grade,omitempty"`
	Status string `json:"status"`
	ReturnedAt *time.Time `json:"returned_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
	RAEs []ReviewRAE `json:"raes"`
	AIDiscardedAt *time.Time `json:"ai_discarded_at,omitempty"`
	Origin string `json:"origin,omitempty"`
	ReviewMs int64 `json:"review_ms"`
	AIMinutes *int `json:"ai_minutes,omitempty"`
}

func (r *Review) origin() string {
	switch {
	case r.AIAt == nil || r.AIFeedback == "":
		return OriginTeacher
	case strings.TrimSpace(r.Feedback) == strings.TrimSpace(r.AIFeedback):
		return OriginAI
	default:
		return OriginAIEdited
	}
}


type ReviewState struct {
	Status string `json:"status"`
	Grade *float64 `json:"grade,omitempty"`
}


func (r *Review) GradeSource() string {
	switch {
	case r.Grade == nil:
		return ""
	case r.AIGrade != nil && *r.AIGrade == *r.Grade && r.Status != ReviewReturned:
		return ""
	case r.AIGrade != nil && *r.AIGrade == *r.Grade:
		return "ai_confirmed"
	default:
		return "profesor"
	}
}

type StudentReview struct {
	Feedback string `json:"feedback"`
	Grade *float64 `json:"grade,omitempty"`
	ReturnedAt *time.Time `json:"returned_at"`
	AIAssisted bool `json:"ai_assisted"` // transparencia: hubo apoyo de IA
	RAEs []StudentRAE `json:"raes,omitempty"`
	Origin string `json:"origin,omitempty"`
	TeacherName string `json:"teacher_name,omitempty"`
	Understood string `json:"understood,omitempty"`
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
	var aiMinutes sql.NullInt64
	var aiAt, returnedAt, discarded sql.NullString
	var updated string
	err := q.QueryRow(`SELECT teacher_id, ai_feedback, ai_grade, ai_reason, ai_confidence, ai_at,
			feedback, grade, status, returned_at, updated_at, ai_discarded_at, feedback_origin, review_ms, ai_minutes
		FROM reviews WHERE submission_id = ?`, submissionID).Scan(&r.TeacherID, &r.AIFeedback, &aiGrade,
		&r.AIReason, &r.AIConfidence, &aiAt, &r.Feedback, &grade, &r.Status, &returnedAt, &updated,
		&discarded, &r.Origin, &r.ReviewMs, &aiMinutes)
	if errors.Is(err, sql.ErrNoRows) {
		return r, nil
	}
	if err != nil {
		return nil, err
	}
	r.AIGrade, r.AIAt, r.Grade, r.ReturnedAt = floatPtr(aiGrade), timePtr(aiAt), floatPtr(grade), timePtr(returnedAt)
	r.UpdatedAt, r.AIDiscardedAt = parseTime(updated), timePtr(discarded)
	if aiMinutes.Valid {
		m := int(aiMinutes.Int64)
		r.AIMinutes = &m
	}
	return r, nil
}

// saveReview inserta o actualiza la revisión completa.
func saveReview(q querier, r *Review) error {
	_, err := q.Exec(`INSERT INTO reviews (submission_id, teacher_id, ai_feedback, ai_grade, ai_reason,
			ai_confidence, ai_at, feedback, grade, status, returned_at, updated_at, ai_discarded_at, feedback_origin, review_ms, ai_minutes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(submission_id) DO UPDATE SET
			teacher_id = excluded.teacher_id, ai_feedback = excluded.ai_feedback, ai_grade = excluded.ai_grade,
			ai_reason = excluded.ai_reason, ai_confidence = excluded.ai_confidence, ai_at = excluded.ai_at,
			feedback = excluded.feedback, grade = excluded.grade, status = excluded.status,
			returned_at = excluded.returned_at, updated_at = excluded.updated_at,
			ai_discarded_at = excluded.ai_discarded_at, feedback_origin = excluded.feedback_origin, review_ms = excluded.review_ms,
			ai_minutes = excluded.ai_minutes`,
		r.SubmissionID, r.TeacherID, r.AIFeedback, floatArg(r.AIGrade), r.AIReason, r.AIConfidence,
		fmtTimePtr(r.AIAt), r.Feedback, floatArg(r.Grade), r.Status, fmtTimePtr(r.ReturnedAt), fmtTime(r.UpdatedAt),
		fmtTimePtr(r.AIDiscardedAt), r.Origin, r.ReviewMs, intArg(r.AIMinutes))
	return err
}

// ReviewForTeacher devuelve la revisión de una entrega oficial del profesor.
func (s *Store) ReviewForTeacher(teacherID, submissionID string) (*Review, error) {
	if err := ownedOfficial(s.db, teacherID, submissionID); err != nil {
		return nil, err
	}
	r, err := loadReview(s.db, teacherID, submissionID)
	if err != nil {
		return nil, err
	}
	if r.RAEs, err = reviewRAEs(s.db, submissionID); err != nil {
		return nil, err
	}
	return r, nil
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
func (s *Store) updateReview(teacherID, submissionID string, change func(r *Review, now time.Time) error, levels func(q querier, r *Review) error, audit func(q querier, before, after *Review) error) (*Review, error) {
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
	before := *r
	now := time.Now().UTC()
	if err := change(r, now); err != nil {
		return nil, err
	}
	r.UpdatedAt = now
	if err := saveReview(tx, r); err != nil {
		return nil, err
	}
	if levels != nil {
		if err := levels(tx, r); err != nil {
			return nil, err
		}
	}
	if audit != nil {
		if err := audit(tx, &before, r); err != nil {
			return nil, err
		}
	}
	if r.RAEs, err = reviewRAEs(tx, submissionID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r, nil
}

func gradeText(g *float64) string {
	if g == nil {
		return "sin nota"
	}
	return fmt.Sprintf("%.1f", *g)
}

func originText(origin string) string {
	switch origin {
	case OriginAI:
		return "texto de la IA aprobado sin cambios"
	case OriginAIEdited:
		return "texto de la IA editado por el profesor"
	}
	return "texto escrito por el profesor"
}

func (s *Store) saveAIDraft(teacherID, actorID, submissionID, feedback string, grade *float64, reason, confidence string, levels map[string]string) (*Review, error) {
	return s.updateReview(teacherID, submissionID, func(r *Review, now time.Time) error {
		if actorID == "" && (r.Status == ReviewReturned || r.AIDiscardedAt != nil) {
			return ErrAlreadyReturned
		}
		r.AIFeedback, r.AIGrade, r.AIReason, r.AIConfidence, r.AIAt = feedback, grade, reason, confidence, &now
		r.AIDiscardedAt = nil
		if r.Status != ReviewReturned && r.Feedback == "" {
			r.Feedback, r.Grade = feedback, grade
		}
		return nil
	}, func(q querier, r *Review) error {
		return saveAILevels(q, submissionID, levels, r.Status != ReviewReturned)
	}, func(q querier, _, r *Review) error {
		detail := "Nota propuesta " + gradeText(r.AIGrade)
		if confidence != "" {
			detail += " · confianza " + confidence
		}
		return logAudit(q, actorID, AuditAIDraft, "", submissionID, detail)
	})
}

// SaveAIDraft guarda la propuesta de la IA y, si el profesor aún no escribió nada,
// precarga su formulario con ella.
func (s *Store) SaveAIDraft(teacherID, submissionID, feedback string, grade *float64, reason, confidence string, levels map[string]string) (*Review, error) {
	return s.saveAIDraft(teacherID, teacherID, submissionID, feedback, grade, reason, confidence, levels)
}

func (s *Store) SaveAutoAIDraft(teacherID, submissionID, feedback string, grade *float64, reason, confidence string, levels map[string]string) (*Review, error) {
	return s.saveAIDraft(teacherID, "", submissionID, feedback, grade, reason, confidence, levels)
}

// SaveTeacherReview guarda la decisión del profesor; returnIt la hace visible al estudiante.
func (s *Store) SaveTeacherReview(teacherID, submissionID, feedback string, grade *float64, returnIt bool, levels map[string]string) (*Review, error) {
	return s.updateReview(teacherID, submissionID, func(r *Review, now time.Time) error {
		r.Feedback, r.Grade = feedback, grade
		if returnIt {
			r.Status, r.ReturnedAt = ReviewReturned, &now
		}
		r.Origin = r.origin()
		return nil
	}, func(q querier, r *Review) error {
		return saveTeacherLevels(q, submissionID, levels)
	}, func(q querier, before, r *Review) error {
		action := AuditDraftSaved
		switch {
		case returnIt && before.Status == ReviewReturned:
			action = AuditReturnedUpdated
		case returnIt:
			action = AuditApproved
		case before.Status == ReviewReturned:
			action = AuditReturnedUpdated
		}
		detail := "Nota " + gradeText(r.Grade)
		if before.Status == ReviewReturned && gradeText(before.Grade) != gradeText(r.Grade) {
			detail = "Nota " + gradeText(before.Grade) + " → " + gradeText(r.Grade)
		} else if r.AIGrade != nil && gradeText(r.AIGrade) != gradeText(r.Grade) {
			detail += " (la IA proponía " + gradeText(r.AIGrade) + ")"
		}
		detail += " · " + originText(r.Origin)
		if action == AuditDraftSaved && before.Feedback == r.Feedback && gradeText(before.Grade) == gradeText(r.Grade) {
			return nil
		}
		return logAudit(q, teacherID, action, "", submissionID, detail)
	})
}

func (s *Store) DiscardAIDraft(teacherID, submissionID string) (*Review, error) {
	return s.updateReview(teacherID, submissionID, func(r *Review, now time.Time) error {
		if r.Status == ReviewReturned {
			return ErrAlreadyReturned
		}
		if r.AIAt == nil {
			return ErrNoAIDraft
		}
		if strings.TrimSpace(r.Feedback) == strings.TrimSpace(r.AIFeedback) {
			r.Feedback = ""
			if gradeText(r.Grade) == gradeText(r.AIGrade) {
				r.Grade = nil
			}
		}
		r.AIFeedback, r.AIGrade, r.AIReason, r.AIConfidence, r.AIAt, r.AIMinutes = "", nil, "", "", nil, nil
		r.AIDiscardedAt = &now
		return nil
	}, func(q querier, r *Review) error {
		_, err := q.Exec(`UPDATE review_raes SET ai_level = '' WHERE submission_id = ?`, submissionID)
		return err
	}, func(q querier, before, _ *Review) error {
		return logAudit(q, teacherID, AuditAIDiscarded, "", submissionID, "Descartó la propuesta (nota propuesta "+gradeText(before.AIGrade)+")")
	})
}

func intArg(n *int) any {
	if n == nil {
		return nil
	}
	return *n
}

func (s *Store) SetAIMinutes(teacherID, submissionID string, minutes int) error {
	if err := ownedOfficial(s.db, teacherID, submissionID); err != nil {
		return err
	}
	var value any
	if minutes > 0 {
		value = minutes
	}
	_, err := s.db.Exec(`UPDATE reviews SET ai_minutes = ? WHERE submission_id = ? AND ai_at IS NOT NULL`, value, submissionID)
	return err
}

func (s *Store) AddReviewTime(teacherID, submissionID string, ms int64) (int64, error) {
	if ms <= 0 {
		return 0, nil
	}
	ms = min(ms, maxReviewMs)
	if err := ownedOfficial(s.db, teacherID, submissionID); err != nil {
		return 0, err
	}
	_, err := s.db.Exec(`INSERT INTO reviews (submission_id, teacher_id, status, updated_at, review_ms) VALUES (?, ?, 'draft', ?, ?)
		ON CONFLICT(submission_id) DO UPDATE SET review_ms = review_ms + excluded.review_ms`,
		submissionID, teacherID, fmtTime(time.Now()), ms)
	return ms, err
}
