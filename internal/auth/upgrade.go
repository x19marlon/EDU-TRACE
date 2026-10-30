package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const usersWithMonitorSQL = `CREATE TABLE users_new (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  email TEXT NOT NULL UNIQUE,
  role TEXT NOT NULL CHECK (role IN ('student', 'teacher', 'monitor')),
  password_hash TEXT NOT NULL,
  created_at TEXT NOT NULL,
  compiles INTEGER NOT NULL DEFAULT 0,
  feedbacks INTEGER NOT NULL DEFAULT 0,
  last_active_at TEXT
)`

var addedColumns = []struct{ table, column, ddl string }{
	{"users", "recovery_hash", `ALTER TABLE users ADD COLUMN recovery_hash TEXT NOT NULL DEFAULT ''`},
	{"assignments", "allow_late", `ALTER TABLE assignments ADD COLUMN allow_late INTEGER NOT NULL DEFAULT 1`},
	{"assignments", "late_until", `ALTER TABLE assignments ADD COLUMN late_until TEXT`},
	{"assignments", "late_penalty", `ALTER TABLE assignments ADD COLUMN late_penalty REAL`},
	{"users", "consent_at", `ALTER TABLE users ADD COLUMN consent_at TEXT`},
	{"users", "consent_version", `ALTER TABLE users ADD COLUMN consent_version TEXT NOT NULL DEFAULT ''`},
	{"courses", "review_minutes_max", `ALTER TABLE courses ADD COLUMN review_minutes_max REAL`},
	{"courses", "batch_hours_max", `ALTER TABLE courses ADD COLUMN batch_hours_max REAL`},
	{"courses", "feedback_hours_max", `ALTER TABLE courses ADD COLUMN feedback_hours_max REAL`},
	{"class_groups", "ai_mode", `ALTER TABLE class_groups ADD COLUMN ai_mode TEXT NOT NULL DEFAULT 'supervisada' CHECK (ai_mode IN ('supervisada', 'inmediata'))`},
	{"reviews", "ai_discarded_at", `ALTER TABLE reviews ADD COLUMN ai_discarded_at TEXT`},
	{"reviews", "feedback_origin", `ALTER TABLE reviews ADD COLUMN feedback_origin TEXT NOT NULL DEFAULT ''`},
	{"reviews", "review_ms", `ALTER TABLE reviews ADD COLUMN review_ms INTEGER NOT NULL DEFAULT 0`},
}

func (s *Store) upgradeSchema() error {
	if err := s.upgradeUsersRole(); err != nil {
		return err
	}
	for _, c := range addedColumns {
		var n int
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, c.table, c.column).Scan(&n); err != nil {
			return err
		}
		if n == 0 {
			if _, err := s.db.Exec(c.ddl); err != nil {
				return fmt.Errorf("columna %s.%s: %w", c.table, c.column, err)
			}
		}
	}
	if _, err := s.db.Exec(`INSERT INTO ai_jobs (submission_id, status, enqueued_at)
		SELECT s.id, 'pendiente', s.created_at FROM submissions s
		LEFT JOIN reviews r ON r.submission_id = s.id
		WHERE s.kind = 'official' AND NOT EXISTS (SELECT 1 FROM ai_jobs j WHERE j.submission_id = s.id)
			AND (r.submission_id IS NULL OR (r.status = 'draft' AND r.ai_at IS NULL AND r.ai_discarded_at IS NULL))`); err != nil {
		return fmt.Errorf("cola de propuestas: %w", err)
	}
	return nil
}

func (s *Store) upgradeUsersRole() error {
	var ddl string
	if err := s.db.QueryRow(`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'users'`).Scan(&ddl); err != nil {
		return err
	}
	if strings.Contains(ddl, "'monitor'") {
		return nil
	}
	ctx := context.Background()
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		return err
	}
	defer conn.ExecContext(ctx, `PRAGMA foreign_keys = ON`)
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, stmt := range []string{
		usersWithMonitorSQL,
		`INSERT INTO users_new (id, name, email, role, password_hash, created_at, compiles, feedbacks, last_active_at)
			SELECT id, name, email, role, password_hash, created_at, compiles, feedbacks, last_active_at FROM users`,
		`DROP TABLE users`,
		`ALTER TABLE users_new RENAME TO users`,
	} {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("tabla users: %w", err)
		}
	}
	rows, err := tx.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		return err
	}
	broken := rows.Next()
	rows.Close()
	if broken {
		return errors.New("tabla users: referencias rotas")
	}
	return tx.Commit()
}
