package auth

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

const PrivacyVersion = "2026-11"

var ErrOwnsCourses = errors.New("primero elimina tus materias y sal de las que compartes con otros profesores: al borrar la cuenta se borrarían las entregas de tus estudiantes")

func (s *Store) RecordConsent(userID string) (time.Time, error) {
	now := time.Now().UTC()
	return now, affected(s.db.Exec(`UPDATE users SET consent_at = ?, consent_version = ? WHERE id = ?`, fmtTime(now), PrivacyVersion, userID))
}

func (u *User) ConsentCurrent() bool {
	return u.ConsentAt != nil && u.ConsentVersion == PrivacyVersion
}

func (s *Store) UpdateName(userID, name string) error {
	return affected(s.db.Exec(`UPDATE users SET name = ? WHERE id = ?`, strings.TrimSpace(name), userID))
}

func (s *Store) DeleteAccount(userID, password string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var hash string
	err = tx.QueryRow(`SELECT password_hash FROM users WHERE id = ?`, userID).Scan(&hash)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if !CheckPassword(password, hash) {
		return ErrInvalidCredentials
	}
	var courses int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM course_staff WHERE teacher_id = ?`, userID).Scan(&courses); err != nil {
		return err
	}
	if courses > 0 {
		return ErrOwnsCourses
	}
	if err := deleteUser(tx, userID); err != nil {
		return err
	}
	return tx.Commit()
}

func deleteUser(q querier, userID string) error {
	for _, stmt := range []string{
		`UPDATE audit_log SET actor_name = 'Cuenta eliminada' WHERE actor_id = ?`,
		`UPDATE review_questions SET answered_by_name = 'Cuenta eliminada' WHERE answered_by = ?`,
	} {
		if _, err := q.Exec(stmt, userID); err != nil {
			return err
		}
	}
	return affected(q.Exec(`DELETE FROM users WHERE id = ?`, userID))
}

type ExportProfile struct {
	Name string `json:"nombre"`
	Email string `json:"email"`
	Role Role `json:"rol"`
	CreatedAt time.Time `json:"cuenta_creada"`
	ConsentAt *time.Time `json:"autorizacion_tratamiento,omitempty"`
	ConsentVersion string `json:"version_politica_aceptada,omitempty"`
	Activity Activity `json:"actividad"`
}

type ExportSubmission struct {
	Submission
	Review *StudentReview `json:"revision_del_profesor,omitempty"`
	MonitorFeedback []MonitorFeedback `json:"retroalimentacion_de_monitores,omitempty"`
	Annotations []Annotation `json:"marcas_enviadas,omitempty"`
	Questions []Question `json:"preguntas,omitempty"`
}

type ExportRating struct {
	At time.Time `json:"fecha"`
	Kind string `json:"retroalimentacion"`
	Understood string `json:"entendio"`
	Comment string `json:"comentario,omitempty"`
}

type ExportClass struct {
	Course string `json:"materia"`
	Group string `json:"clase"`
	Role string `json:"participacion"`
}

type DataExport struct {
	GeneratedAt time.Time `json:"generado"`
	PolicyVersion string `json:"version_politica"`
	Profile ExportProfile `json:"perfil"`
	Classes []ExportClass `json:"clases"`
	Submissions []ExportSubmission `json:"entregas_e_intentos,omitempty"`
	Ratings []ExportRating `json:"valoraciones,omitempty"`
	MonitorFeedback []MonitorFeedback `json:"retroalimentacion_escrita,omitempty"`
	Notifications []Notification `json:"avisos,omitempty"`
}

func (s *Store) ExportData(u *User) (*DataExport, error) {
	out := &DataExport{GeneratedAt: time.Now().UTC(), PolicyVersion: PrivacyVersion, Classes: []ExportClass{}}
	var created string
	var consent sql.NullString
	err := s.db.QueryRow(`SELECT name, email, role, created_at, consent_at, consent_version, compiles, feedbacks FROM users WHERE id = ?`, u.ID).
		Scan(&out.Profile.Name, &out.Profile.Email, &out.Profile.Role, &created, &consent, &out.Profile.ConsentVersion,
			&out.Profile.Activity.Compiles, &out.Profile.Activity.Feedbacks)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	out.Profile.CreatedAt, out.Profile.ConsentAt = parseTime(created), timePtr(consent)

	rows, err := s.db.Query(`SELECT c.name, g.name, 'estudiante' FROM group_students gs
			JOIN class_groups g ON g.id = gs.group_id JOIN courses c ON c.id = g.course_id WHERE gs.student_id = ?
		UNION ALL SELECT c.name, g.name, 'monitor' FROM group_monitors gm
			JOIN class_groups g ON g.id = gm.group_id JOIN courses c ON c.id = g.course_id WHERE gm.monitor_id = ?
		UNION ALL SELECT c.name, g.name, 'profesor' FROM class_groups g
			JOIN courses c ON c.id = g.course_id WHERE c.id IN (SELECT course_id FROM course_staff WHERE teacher_id = ?)`, u.ID, u.ID, u.ID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var c ExportClass
		if err := rows.Scan(&c.Course, &c.Group, &c.Role); err != nil {
			rows.Close()
			return nil, err
		}
		out.Classes = append(out.Classes, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if u.Role == RoleStudent {
		if out.Submissions, err = s.exportSubmissions(u); err != nil {
			return nil, err
		}
		if out.Ratings, err = s.exportRatings(u.ID); err != nil {
			return nil, err
		}
	}
	if u.Role == RoleMonitor {
		rows, err := s.db.Query(`SELECT f.id, f.submission_id, f.monitor_id, u.name, f.body, f.created_at
			FROM monitor_feedback f JOIN users u ON u.id = f.monitor_id WHERE f.monitor_id = ? ORDER BY f.created_at`, u.ID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var f MonitorFeedback
			var at string
			if err := rows.Scan(&f.ID, &f.SubmissionID, &f.MonitorID, &f.MonitorName, &f.Body, &at); err != nil {
				return nil, err
			}
			f.CreatedAt = parseTime(at)
			out.MonitorFeedback = append(out.MonitorFeedback, f)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	if out.Notifications, err = exportNotifications(s.db, u.ID); err != nil {
		return nil, err
	}
	return out, nil
}

func exportNotifications(q querier, userID string) ([]Notification, error) {
	rows, err := q.Query(`SELECT id, kind, title, link, created_at, read_at FROM notifications WHERE user_id = ? ORDER BY id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Notification
	for rows.Next() {
		var n Notification
		var created string
		var read sql.NullString
		if err := rows.Scan(&n.ID, &n.Kind, &n.Title, &n.Link, &created, &read); err != nil {
			return nil, err
		}
		n.CreatedAt, n.ReadAt = parseTime(created), timePtr(read)
		out = append(out, n)
	}
	return out, rows.Err()
}

func (s *Store) exportSubmissions(u *User) ([]ExportSubmission, error) {
	rows, err := s.db.Query(`SELECT `+metaCols+`, `+contentCols+` FROM submissions s
		WHERE s.student_id = ? ORDER BY s.created_at`, u.ID)
	if err != nil {
		return nil, err
	}
	var list []ExportSubmission
	for rows.Next() {
		sub, err := scanSubmission(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		list = append(list, ExportSubmission{Submission: *sub})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	reviews, err := s.studentSubmissions(u.ID, KindOfficial, 100000)
	if err != nil {
		return nil, err
	}
	byID := map[string]StudentSubmission{}
	for _, r := range reviews {
		byID[r.ID] = r
	}
	for i := range list {
		if err := s.withTests(&list[i].Submission, false); err != nil {
			return nil, err
		}
		if r, ok := byID[list[i].ID]; ok {
			list[i].Review, list[i].MonitorFeedback, list[i].Questions = r.Review, r.MonitorFeedback, r.Questions
		}
		if list[i].Kind == KindOfficial {
			list[i].Annotations = s.AnnotationsFor(u, list[i].ID)
		}
	}
	return list, nil
}

func (s *Store) exportRatings(studentID string) ([]ExportRating, error) {
	rows, err := s.db.Query(`SELECT at, kind, understood, comment FROM feedback_ratings WHERE student_id = ? ORDER BY at`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ExportRating
	for rows.Next() {
		var r ExportRating
		var at string
		if err := rows.Scan(&at, &r.Kind, &r.Understood, &r.Comment); err != nil {
			return nil, err
		}
		r.At = parseTime(at)
		out = append(out, r)
	}
	return out, rows.Err()
}
