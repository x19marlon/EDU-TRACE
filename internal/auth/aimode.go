package auth

import "errors"

const (
	AIModeSupervised = "supervisada"
	AIModeImmediate = "inmediata"
)

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

func (s *Store) StudentAIAllowed(studentID, assignmentID string) (bool, error) {
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
			WHERE gs.student_id = ? AND g.ai_mode = 'supervisada'`, studentID).Scan(&supervised)
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
