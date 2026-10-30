package auth

import (
	"database/sql"
	"time"
)

const (
	AuditAIDraft = "propuesta_ia"
	AuditAIDiscarded = "propuesta_descartada"
	AuditDraftSaved = "borrador_guardado"
	AuditApproved = "aprobada"
	AuditReturnedUpdated = "devuelta_actualizada"
	AuditAnnotationsSent = "anotaciones_enviadas"
	AuditAIMode = "modo_ia"
	AuditTestsRerun = "casos_reejecutados"
	SystemActorName = "EduTrace (IA)"
	SystemActorRole = "sistema"
	auditDefaultLimit = 200
)

type AuditEntry struct {
	ID int64 `json:"id"`
	At time.Time `json:"at"`
	ActorName string `json:"actor_name"`
	ActorRole string `json:"actor_role"`
	Action string `json:"action"`
	SubmissionID string `json:"submission_id,omitempty"`
	StudentName string `json:"student_name,omitempty"`
	AssignmentTitle string `json:"assignment_title,omitempty"`
	Detail string `json:"detail,omitempty"`
}

func logAudit(q querier, actorID, action, groupID, submissionID, detail string) error {
	_, err := q.Exec(`INSERT INTO audit_log (at, actor_id, actor_name, actor_role, action, group_id, submission_id, detail)
		VALUES (?, NULLIF(?, ''),
			COALESCE((SELECT name FROM users WHERE id = ?), ?),
			COALESCE((SELECT role FROM users WHERE id = ?), ?),
			?, COALESCE(NULLIF(?, ''), (SELECT group_id FROM submissions WHERE id = ?)), NULLIF(?, ''), ?)`,
		fmtTime(time.Now()), actorID, actorID, SystemActorName, actorID, SystemActorRole,
		action, groupID, submissionID, submissionID, detail)
	return err
}

const auditSelect = `SELECT l.id, l.at, l.actor_name, l.actor_role, l.action, COALESCE(l.submission_id, ''),
		COALESCE(u.name, ''), COALESCE(a.title, ''), l.detail
	FROM audit_log l
	LEFT JOIN submissions s ON s.id = l.submission_id
	LEFT JOIN users u ON u.id = s.student_id
	LEFT JOIN assignments a ON a.id = s.assignment_id`

func scanAudit(rows *sql.Rows) ([]AuditEntry, error) {
	defer rows.Close()
	out := []AuditEntry{}
	for rows.Next() {
		var e AuditEntry
		var at string
		if err := rows.Scan(&e.ID, &at, &e.ActorName, &e.ActorRole, &e.Action, &e.SubmissionID, &e.StudentName, &e.AssignmentTitle, &e.Detail); err != nil {
			return nil, err
		}
		e.At = parseTime(at)
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) GroupAudit(teacherID, groupID string, limit int) ([]AuditEntry, error) {
	if _, err := s.ownedGroup(s.db, teacherID, groupID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 1000 {
		limit = auditDefaultLimit
	}
	rows, err := s.db.Query(auditSelect+` WHERE l.group_id = ? ORDER BY l.id DESC LIMIT ?`, groupID, limit)
	if err != nil {
		return nil, err
	}
	return scanAudit(rows)
}

func (s *Store) SubmissionAudit(teacherID, submissionID string) ([]AuditEntry, error) {
	if err := ownedOfficial(s.db, teacherID, submissionID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(auditSelect+` WHERE l.submission_id = ? ORDER BY l.id DESC`, submissionID)
	if err != nil {
		return nil, err
	}
	return scanAudit(rows)
}
