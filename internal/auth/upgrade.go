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

func (s *Store) upgradeSchema() error {
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
