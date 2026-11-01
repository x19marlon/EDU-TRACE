package auth

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"
)

const MaxLatePenalty = 5.0

var ErrInvalidDeadline = errors.New("la configuración de la fecha límite no es válida")

type AssignmentPolicy struct {
	AllowLate bool
	LateUntil *time.Time
	LatePenalty *float64
	Evaluation bool
}

func (p AssignmentPolicy) validate(dueAt *time.Time) error {
	if p.Evaluation && dueAt == nil {
		return fmt.Errorf("%w: un taller en modo evaluación necesita fecha límite, porque la IA queda desactivada hasta esa fecha", ErrInvalidDeadline)
	}
	if p.LateUntil != nil && (dueAt == nil || !p.AllowLate || !p.LateUntil.After(*dueAt)) {
		return fmt.Errorf("%w: el cierre de entregas tardías debe ser posterior a la fecha límite", ErrInvalidDeadline)
	}
	if p.LatePenalty != nil && (math.IsNaN(*p.LatePenalty) || *p.LatePenalty < 0 || *p.LatePenalty > MaxLatePenalty) {
		return fmt.Errorf("%w: la penalización por día debe estar entre 0.0 y 5.0", ErrInvalidDeadline)
	}
	if p.LatePenalty != nil && (dueAt == nil || !p.AllowLate) {
		return fmt.Errorf("%w: la penalización solo aplica si el taller tiene fecha límite y acepta entregas tardías", ErrInvalidDeadline)
	}
	return nil
}

type DeadlineError struct {
	ClosedAt time.Time
	Late bool
}

func (e *DeadlineError) Error() string {
	if e.Late {
		return "el plazo para entregas tardías de este taller ya terminó"
	}
	return "la fecha límite de este taller ya pasó y no acepta entregas tardías"
}

func checkDeadline(q querier, assignmentID, groupID string, at time.Time) error {
	var due, lateUntil sql.NullString
	var allowLate bool
	err := q.QueryRow(`SELECT due_at, allow_late, late_until FROM assignments WHERE id = ? AND group_id = ?`,
		assignmentID, groupID).Scan(&due, &allowLate, &lateUntil)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil || !due.Valid {
		return err
	}
	dueAt := parseTime(due.String)
	if !at.After(dueAt) {
		return nil
	}
	if !allowLate {
		return &DeadlineError{ClosedAt: dueAt}
	}
	if lateUntil.Valid {
		if until := parseTime(lateUntil.String); at.After(until) {
			return &DeadlineError{ClosedAt: until, Late: true}
		}
	}
	return nil
}
