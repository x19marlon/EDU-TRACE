package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

type legacySession struct {
	UserID string `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

type legacyData struct {
	Users map[string]*User `json:"users"`
	Sessions map[string]*legacySession `json:"sessions"` 
	Courses map[string]*Course `json:"courses"`
	Groups map[string]*Group `json:"groups"`
	Submissions map[string]*SubmissionMeta `json:"submissions"` 
	Assignments map[string]*Assignment `json:"assignments"`
	Reviews map[string]*Review `json:"reviews"`
}

func (s *Store) importLegacyJSON() error {
	path := filepath.Join(s.dir, "edutrace.json")
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil 
	}
	if err != nil {
		return err
	}
	var users int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&users); err != nil {
		return err
	}
	if users > 0 {
		return errors.New("la base ya tiene datos y sigue existiendo edutrace.json: revísalo a mano")
	}
	var old legacyData
	if err := json.Unmarshal(raw, &old); err != nil {
		return fmt.Errorf("leer %s: %w", path, err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	
	for _, u := range old.Users {
		if _, err := tx.Exec(`INSERT INTO users (id, name, email, role, password_hash, created_at, compiles, feedbacks, last_active_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, u.ID, u.Name, normalizeEmail(u.Email), u.Role, u.PasswordHash,
			fmtTime(u.CreatedAt), u.Activity.Compiles, u.Activity.Feedbacks, fmtTimePtr(&u.Activity.LastActiveAt)); err != nil {
			return fmt.Errorf("usuario %s: %w", u.ID, err)
		}
	}

	now := time.Now()
	sessions := 0
	for tokenHash, se := range old.Sessions {
		if se.ExpiresAt.Before(now) || old.Users[se.UserID] == nil {
			continue 
		}
		if _, err := tx.Exec(`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES (?, ?, ?)`,
			tokenHash, se.UserID, fmtTime(se.ExpiresAt)); err != nil {
			return fmt.Errorf("sesión: %w", err)
		}
		sessions++
	}

	for _, c := range old.Courses {
		if _, err := tx.Exec(`INSERT INTO courses (id, teacher_id, name, created_at) VALUES (?, ?, ?, ?)`,
			c.ID, c.TeacherID, c.Name, fmtTime(c.CreatedAt)); err != nil {
			return fmt.Errorf("materia %s: %w", c.ID, err)
		}
	}

	for _, g := range old.Groups {
		if _, err := tx.Exec(`INSERT INTO class_groups (id, course_id, name, join_code, created_at) VALUES (?, ?, ?, ?, ?)`,
			g.ID, g.CourseID, g.Name, g.JoinCode, fmtTime(g.CreatedAt)); err != nil {
			return fmt.Errorf("clase %s: %w", g.ID, err)
		}
		
		for _, sid := range g.StudentIDs {
			if old.Users[sid] == nil {
				continue
			}
			if _, err := tx.Exec(`INSERT INTO group_students (group_id, student_id, joined_at) VALUES (?, ?, ?)
				ON CONFLICT DO NOTHING`, g.ID, sid, fmtTime(g.CreatedAt)); err != nil {
				return fmt.Errorf("inscripción en %s: %w", g.ID, err)
			}
		}
	}

	for _, a := range old.Assignments {
		if _, err := tx.Exec(`INSERT INTO assignments (id, group_id, title, statement, rubric, due_at, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`, a.ID, a.GroupID, a.Title, a.Statement, a.Rubric,
			fmtTimePtr(a.DueAt), fmtTime(a.CreatedAt)); err != nil {
			return fmt.Errorf("taller %s: %w", a.ID, err)
		}
		for i, f := range a.Files {
			if _, err := tx.Exec(`INSERT INTO assignment_files (id, assignment_id, name, size, content_type, position)
				VALUES (?, ?, ?, ?, ?, ?)`, f.ID, a.ID, f.Name, f.Size, f.ContentType, i); err != nil {
				return fmt.Errorf("archivo del taller %s: %w", a.ID, err)
			}
		}
	}

	imported, skipped := map[string]bool{}, 0
	for id, meta := range old.Submissions {
		
		body, err := os.ReadFile(filepath.Join(s.dir, "submissions", id+".json"))
		if err != nil {
			slog.Warn("envío sin archivo: se omite", "submission_id", id, "error", err)
			skipped++
			continue
		}
		var sub Submission
		if err := json.Unmarshal(body, &sub); err != nil {
			return fmt.Errorf("envío %s: %w", id, err)
		}
		sub.SubmissionMeta = *meta 
		if sub.Kind != KindAttempt {
			sub.Kind = KindOfficial 
		}
		if old.Groups[sub.GroupID] == nil || old.Users[sub.StudentID] == nil {
			slog.Warn("envío de una clase o estudiante que ya no existe: se omite", "submission_id", id)
			skipped++
			continue
		}
		if sub.AssignmentID != "" && old.Assignments[sub.AssignmentID] == nil {
			sub.AssignmentID = ""
		}
		if _, err := tx.Exec(`INSERT INTO submissions (id, kind, group_id, assignment_id, student_id, created_at,
				success, run_error, exit_code, lines, has_ai_feedback, error_summary, code_hash,
				code, stdin, compiler_output, program_output, ai_feedback, ai_feedback_informal)
			VALUES (?, ?, ?, NULLIF(?, ''), ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			sub.ID, sub.Kind, sub.GroupID, sub.AssignmentID, sub.StudentID, fmtTime(sub.CreatedAt),
			sub.Success, sub.RunError, sub.ExitCode, sub.Lines, sub.HasAIFeedback, sub.ErrorSummary, sub.CodeHash,
			sub.Code, sub.Stdin, sub.CompilerOutput, sub.ProgramOutput, sub.AIFeedback, sub.AIFeedbackInf); err != nil {
			return fmt.Errorf("envío %s: %w", id, err)
		}
		imported[id] = true
	}

	reviews := 0
	for id, r := range old.Reviews {
		if !imported[id] || old.Users[r.TeacherID] == nil {
			continue
		}
		r.SubmissionID = id
		if r.Status != ReviewReturned {
			r.Status = ReviewDraft
		}
		if err := saveReview(tx, r); err != nil {
			return fmt.Errorf("revisión de %s: %w", id, err)
		}
		reviews++
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	slog.Info("datos migrados de edutrace.json a SQLite",
		"usuarios", len(old.Users), "sesiones", sessions, "materias", len(old.Courses), "clases", len(old.Groups),
		"talleres", len(old.Assignments), "envios", len(imported), "envios_omitidos", skipped, "revisiones", reviews)
	
	return os.Rename(path, path+".migrated")
}
