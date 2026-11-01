package auth

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

const (
	MaxCriteria = 10
	maxCriterionName = 120
	maxCriterionPoints = 100
)

var (
	ErrInvalidCriteria = errors.New("los criterios de evaluación no son válidos")
	ErrInvalidScore = errors.New("cada puntaje debe estar entre 0 y el máximo de su criterio")
)

type Criterion struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Points float64 `json:"points"`
}

type ReviewCriterion struct {
	Criterion
	Score *float64 `json:"score,omitempty"`
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }

func NormalizeCriteria(in []Criterion) ([]Criterion, error) {
	if len(in) > MaxCriteria {
		return nil, fmt.Errorf("%w: máximo %d criterios por taller", ErrInvalidCriteria, MaxCriteria)
	}
	out := make([]Criterion, 0, len(in))
	for i, c := range in {
		c.Name = strings.TrimSpace(c.Name)
		if c.Name == "" || utf8.RuneCountInString(c.Name) > maxCriterionName {
			return nil, fmt.Errorf("%w: el criterio %d necesita un nombre de hasta %d caracteres", ErrInvalidCriteria, i+1, maxCriterionName)
		}
		c.Points = round1(c.Points)
		if math.IsNaN(c.Points) || c.Points <= 0 || c.Points > maxCriterionPoints {
			return nil, fmt.Errorf("%w: el puntaje de «%s» debe ser mayor que 0 y hasta %d", ErrInvalidCriteria, c.Name, maxCriterionPoints)
		}
		c.ID = ""
		out = append(out, c)
	}
	return out, nil
}

func assignmentCriteria(q querier, assignmentID string) ([]Criterion, error) {
	rows, err := q.Query(`SELECT id, name, points FROM assignment_criteria WHERE assignment_id = ? ORDER BY position`, assignmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Criterion{}
	for rows.Next() {
		var c Criterion
		if err := rows.Scan(&c.ID, &c.Name, &c.Points); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) SetAssignmentCriteria(teacherID, assignmentID string, list []Criterion) ([]Criterion, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := s.ownedAssignment(tx, teacherID, assignmentID); err != nil {
		return nil, err
	}
	current, err := assignmentCriteria(tx, assignmentID)
	if err != nil {
		return nil, err
	}
	out := make([]Criterion, len(list))
	for i, c := range list {
		if i < len(current) {
			c.ID = current[i].ID
			if _, err := tx.Exec(`UPDATE assignment_criteria SET name = ?, points = ? WHERE id = ?`, c.Name, c.Points, c.ID); err != nil {
				return nil, err
			}
			if _, err := tx.Exec(`UPDATE review_criteria SET score = MIN(score, ?) WHERE criterion_id = ?`, c.Points, c.ID); err != nil {
				return nil, err
			}
		} else {
			if c.ID, err = randomHex(12); err != nil {
				return nil, err
			}
			if _, err := tx.Exec(`INSERT INTO assignment_criteria (id, assignment_id, position, name, points) VALUES (?, ?, ?, ?, ?)`,
				c.ID, assignmentID, i, c.Name, c.Points); err != nil {
				return nil, err
			}
		}
		out[i] = c
	}
	for _, c := range current[min(len(list), len(current)):] {
		if _, err := tx.Exec(`DELETE FROM assignment_criteria WHERE id = ?`, c.ID); err != nil {
			return nil, err
		}
	}
	return out, tx.Commit()
}

func (s *Store) attachCriteria(list []Assignment) error {
	for i := range list {
		c, err := assignmentCriteria(s.db, list[i].ID)
		if err != nil {
			return err
		}
		list[i].Criteria = c
	}
	return nil
}

func reviewCriteria(q querier, submissionID string) ([]ReviewCriterion, error) {
	rows, err := q.Query(`SELECT c.id, c.name, c.points, rc.score FROM submissions s
		JOIN assignment_criteria c ON c.assignment_id = s.assignment_id
		LEFT JOIN review_criteria rc ON rc.criterion_id = c.id AND rc.submission_id = s.id
		WHERE s.id = ? ORDER BY c.position`, submissionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ReviewCriterion{}
	for rows.Next() {
		var c ReviewCriterion
		var score *float64
		if err := rows.Scan(&c.ID, &c.Name, &c.Points, &score); err != nil {
			return nil, err
		}
		c.Score = score
		out = append(out, c)
	}
	return out, rows.Err()
}

func saveCriteriaScores(q querier, submissionID string, scores map[string]float64) error {
	if scores == nil {
		return nil
	}
	criteria, err := reviewCriteria(q, submissionID)
	if err != nil {
		return err
	}
	for _, c := range criteria {
		v, ok := scores[c.ID]
		if !ok {
			if _, err := q.Exec(`DELETE FROM review_criteria WHERE submission_id = ? AND criterion_id = ?`, submissionID, c.ID); err != nil {
				return err
			}
			continue
		}
		v = round1(v)
		if math.IsNaN(v) || v < 0 || v > c.Points {
			return fmt.Errorf("%w («%s» vale hasta %.1f)", ErrInvalidScore, c.Name, c.Points)
		}
		if _, err := q.Exec(`INSERT INTO review_criteria (submission_id, criterion_id, score) VALUES (?, ?, ?)
			ON CONFLICT(submission_id, criterion_id) DO UPDATE SET score = excluded.score`, submissionID, c.ID, v); err != nil {
			return err
		}
	}
	return nil
}

func CriteriaGrade(list []ReviewCriterion, max float64) *float64 {
	var got, total float64
	for _, c := range list {
		if c.Score == nil {
			return nil
		}
		got += *c.Score
		total += c.Points
	}
	if total == 0 {
		return nil
	}
	g := round1(got / total * max)
	return &g
}

func criteriaText(rubric string, criteria []Criterion) string {
	if len(criteria) == 0 {
		return rubric
	}
	var b strings.Builder
	b.WriteString(rubric)
	if rubric != "" {
		b.WriteString("\n")
	}
	b.WriteString("Criterios con puntaje:")
	for _, c := range criteria {
		fmt.Fprintf(&b, "\n- %s (%.1f puntos)", c.Name, c.Points)
	}
	return b.String()
}
