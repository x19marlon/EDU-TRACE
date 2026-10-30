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
