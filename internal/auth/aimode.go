package auth

import (
	"database/sql"
	"errors"
	"time"
)

const (
	AIModeSupervised = "supervisada"
	AIModeImmediate = "inmediata"
	AIBlockEvaluation = "evaluacion"
)

type OpenEvaluation struct {
	Title string `json:"title"`
	Until time.Time `json:"until"`
}

type AIBlock struct {
	Code string
	Evaluation *OpenEvaluation
}

const evaluationUntil = `CASE WHEN a.allow_late = 1 AND a.late_until IS NOT NULL THEN a.late_until ELSE a.due_at END`

func openEvaluation(q querier, where string, args ...any) (*OpenEvaluation, error) {
	var e OpenEvaluation
	var until string
	err := q.QueryRow(`SELECT a.title, `+evaluationUntil+` FROM assignments a
		WHERE a.evaluation = 1 AND `+evaluationUntil+` > ? AND `+where+`
		ORDER BY `+evaluationUntil+` DESC LIMIT 1`, append([]any{fmtTime(time.Now())}, args...)...).Scan(&e.Title, &until)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	e.Until = parseTime(until)
	return &e, nil
}

var ErrInvalidAIMode = errors.New("el modo de IA debe ser «supervisada» o «inmediata»")

func (s *Store) SetGroupAIMode(teacherID, groupID, mode string) error {
	if mode != AIModeSupervised && mode != AIModeImmediate {
		return ErrInvalidAIMode
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	g, err := s.ownedGroup(tx, teacherID, groupID)
	if err != nil {
		return err
	}
	if g.AIMode == mode {
		return nil
	}
	if _, err := tx.Exec(`UPDATE class_groups SET ai_mode = ? WHERE id = ?`, mode, groupID); err != nil {
		return err
	}
	if err := logAudit(tx, teacherID, AuditAIMode, groupID, "", "Retroalimentación de IA para el estudiante: "+g.AIMode+" → "+mode); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) StudentAIBlock(studentID, assignmentID string) (*AIBlock, error) {
	exam, err := openEvaluation(s.db, `a.group_id IN (SELECT group_id FROM group_students WHERE student_id = ?)`, studentID)
	if err != nil {
		return nil, err
	}
	if exam != nil {
		return &AIBlock{Code: AIBlockEvaluation, Evaluation: exam}, nil
	}
	allowed, err := s.studentAIAllowed(studentID, assignmentID)
	if err != nil || allowed {
		return nil, err
	}
	return &AIBlock{Code: AIModeSupervised}, nil
}

func (s *Store) studentAIAllowed(studentID, assignmentID string) (bool, error) {
	var supervised int
	var err error
	if assignmentID != "" {
		err = s.db.QueryRow(`SELECT COUNT(*) FROM assignments a
			JOIN class_groups g ON g.id = a.group_id
			JOIN group_students gs ON gs.group_id = g.id
			WHERE a.id = ? AND gs.student_id = ? AND g.ai_mode = 'supervisada'`, assignmentID, studentID).Scan(&supervised)
	} else {
		err = s.db.QueryRow(`SELECT COUNT(*) FROM group_students gs
			JOIN class_groups g ON g.id = gs.group_id
			JOIN courses c ON c.id = g.course_id
			WHERE gs.student_id = ? AND g.ai_mode = 'supervisada' AND c.closed_at IS NULL`, studentID).Scan(&supervised)
	}
	return supervised == 0, err
}

func (s *Store) AssignmentGroup(assignmentID string) string {
	var groupID string
	if err := s.db.QueryRow(`SELECT group_id FROM assignments WHERE id = ?`, assignmentID).Scan(&groupID); err != nil {
		return ""
	}
	return groupID
}

func (s *Store) SubmissionGroup(submissionID string) string {
	var groupID string
	if err := s.db.QueryRow(`SELECT group_id FROM submissions WHERE id = ?`, submissionID).Scan(&groupID); err != nil {
		return ""
	}
	return groupID
}
