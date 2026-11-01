package auth

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/marlon/edu-trace/internal/model"
)

const (
	MaxTests = 10
	maxTestName = 80
	maxTestText = 4000
)

var ErrInvalidTests = errors.New("los casos de prueba no son válidos")

func NormalizeTests(in []model.TestCase) ([]model.TestCase, error) {
	if len(in) > MaxTests {
		return nil, fmt.Errorf("%w: máximo %d casos por taller", ErrInvalidTests, MaxTests)
	}
	out := make([]model.TestCase, 0, len(in))
	for i, t := range in {
		t.Name = strings.TrimSpace(t.Name)
		if t.Name == "" {
			t.Name = fmt.Sprintf("Caso %d", i+1)
		}
		if utf8.RuneCountInString(t.Name) > maxTestName {
			return nil, fmt.Errorf("%w: el nombre del caso %d supera los %d caracteres", ErrInvalidTests, i+1, maxTestName)
		}
		t.Stdin = strings.ReplaceAll(t.Stdin, "\r\n", "\n")
		t.Expected = strings.ReplaceAll(t.Expected, "\r\n", "\n")
		if strings.TrimSpace(t.Expected) == "" {
			return nil, fmt.Errorf("%w: escribe la salida esperada del caso %d", ErrInvalidTests, i+1)
		}
		if len(t.Stdin) > maxTestText || len(t.Expected) > maxTestText {
			return nil, fmt.Errorf("%w: la entrada y la salida de cada caso tienen máximo %d caracteres", ErrInvalidTests, maxTestText)
		}
		switch t.Match {
		case "":
			t.Match = model.MatchContains
		case model.MatchContains, model.MatchExact:
		default:
			return nil, fmt.Errorf("%w: la comparación debe ser «contiene» o «exacta»", ErrInvalidTests)
		}
		t.ID = ""
		out = append(out, t)
	}
	return out, nil
}

func assignmentTests(q querier, assignmentID string) ([]model.TestCase, error) {
	rows, err := q.Query(`SELECT id, name, stdin, expected, match, hidden FROM assignment_tests
		WHERE assignment_id = ? ORDER BY position`, assignmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.TestCase{}
	for rows.Next() {
		var t model.TestCase
		if err := rows.Scan(&t.ID, &t.Name, &t.Stdin, &t.Expected, &t.Match, &t.Hidden); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func sameTests(a, b []model.TestCase) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		x, y := a[i], b[i]
		if x.Name != y.Name || x.Stdin != y.Stdin || x.Expected != y.Expected || x.Match != y.Match || x.Hidden != y.Hidden {
			return false
		}
	}
	return true
}

func (s *Store) SetAssignmentTests(teacherID, assignmentID string, tests []model.TestCase) ([]model.TestCase, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := s.ownedAssignment(tx, teacherID, assignmentID); err != nil {
		return nil, err
	}
	current, err := assignmentTests(tx, assignmentID)
	if err != nil {
		return nil, err
	}
	if sameTests(current, tests) {
		return current, nil
	}
	if _, err := tx.Exec(`DELETE FROM assignment_tests WHERE assignment_id = ?`, assignmentID); err != nil {
		return nil, err
	}
	hidden := 0
	for _, t := range tests {
		if t.Hidden {
			hidden++
		}
	}
	var groupID, title string
	if err := tx.QueryRow(`SELECT group_id, title FROM assignments WHERE id = ?`, assignmentID).Scan(&groupID, &title); err != nil {
		return nil, err
	}
	detail := quoted(title) + ": " + countLabel(len(tests), "caso", "casos")
	if hidden > 0 {
		detail += " (" + countLabel(hidden, "oculto", "ocultos") + ")"
	}
	if err := logAudit(tx, teacherID, AuditTestsUpdated, groupID, "", detail); err != nil {
		return nil, err
	}
	out := make([]model.TestCase, len(tests))
	for i, t := range tests {
		if t.ID, err = randomHex(12); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(`INSERT INTO assignment_tests (id, assignment_id, position, name, stdin, expected, match, hidden)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, t.ID, assignmentID, i, t.Name, t.Stdin, t.Expected, t.Match, t.Hidden); err != nil {
			return nil, err
		}
		out[i] = t
	}
	return out, tx.Commit()
}

func (s *Store) TestsForRun(assignmentID string) ([]model.TestCase, error) {
	if assignmentID == "" {
		return nil, nil
	}
	return assignmentTests(s.db, assignmentID)
}

func (s *Store) attachTests(list []Assignment, full bool) error {
	for i := range list {
		tests, err := assignmentTests(s.db, list[i].ID)
		if err != nil {
			return err
		}
		if !full {
			for j := range tests {
				if tests[j].Hidden {
					tests[j] = model.TestCase{ID: tests[j].ID, Name: tests[j].Name, Hidden: true}
				}
			}
		}
		list[i].Tests = tests
	}
	return nil
}

func saveTestResults(q querier, submissionID string, results []model.TestResult) error {
	if _, err := q.Exec(`DELETE FROM submission_tests WHERE submission_id = ?`, submissionID); err != nil {
		return err
	}
	for _, r := range results {
		if r.TestID == "" {
			continue
		}
		if _, err := q.Exec(`INSERT INTO submission_tests (submission_id, test_id, passed, output, error)
			SELECT ?, id, ?, ?, ? FROM assignment_tests WHERE id = ?`,
			submissionID, r.Passed, r.Output, r.Error, r.TestID); err != nil {
			return err
		}
	}
	return nil
}

func submissionTests(q querier, submissionID string) ([]model.TestResult, error) {
	rows, err := q.Query(`SELECT t.id, t.name, t.hidden, st.passed, t.stdin, t.expected, st.output, st.error
		FROM submission_tests st JOIN assignment_tests t ON t.id = st.test_id
		WHERE st.submission_id = ? ORDER BY t.position`, submissionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.TestResult{}
	for rows.Next() {
		var r model.TestResult
		if err := rows.Scan(&r.TestID, &r.Name, &r.Hidden, &r.Passed, &r.Stdin, &r.Expected, &r.Output, &r.Error); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) SubmissionTests(submissionID string) ([]model.TestResult, error) {
	return submissionTests(s.db, submissionID)
}

func (s *Store) SaveSubmissionTests(teacherID, submissionID string, results []model.TestResult) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var n int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM submissions s
		JOIN class_groups g ON g.id = s.group_id
		JOIN courses c ON c.id = g.course_id
		WHERE s.id = ? AND c.id IN (SELECT course_id FROM course_staff WHERE teacher_id = ?)`, submissionID, teacherID).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	if err := saveTestResults(tx, submissionID, results); err != nil {
		return err
	}
	if err := logAudit(tx, teacherID, AuditTestsRerun, "", submissionID, testsSummary(results)); err != nil {
		return err
	}
	return tx.Commit()
}

func testsSummary(results []model.TestResult) string {
	passed := 0
	for _, r := range results {
		if r.Passed {
			passed++
		}
	}
	return fmt.Sprintf("Pasa %d de %d casos de prueba", passed, len(results))
}

func (s *Store) withTests(sub *Submission, full bool) error {
	if sub == nil {
		return nil
	}
	results, err := submissionTests(s.db, sub.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if !full {
		results = model.PublicResults(results)
	}
	sub.Tests = results
	return nil
}

func countLabel(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}
