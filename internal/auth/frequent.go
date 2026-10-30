package auth

import (
	"sort"
	"time"

	"github.com/marlon/edu-trace/internal/compiler"
)

const (
	maxFrequentErrors = 12
	maxHardTests = 10
	maxExampleAssignments = 3
)

type FrequentError struct {
	Message string `json:"message"`
	Explanation string `json:"explanation,omitempty"`
	Count int `json:"count"`
	Students int `json:"students"`
	LastAt time.Time `json:"last_at"`
	Assignments []string `json:"assignments"`
}

type HardTest struct {
	Assignment string `json:"assignment"`
	Test string `json:"test"`
	Hidden bool `json:"hidden"`
	Students int `json:"students"`
	NotPassing int `json:"not_passing"`
	Failures int `json:"failures"`
	Runs int `json:"runs"`
}

type ErrorReport struct {
	Scope string `json:"scope"`
	Groups int `json:"groups"`
	Analyzed int `json:"analyzed"`
	StudentsWithErrors int `json:"students_with_errors"`
	Compile []FrequentError `json:"compile"`
	Runtime []FrequentError `json:"runtime"`
	Tests []HardTest `json:"tests"`
}

type errorBucket struct {
	FrequentError
	students map[string]bool
	assignments map[string]bool
}

func addToBucket(buckets map[string]*errorBucket, key, explanation, student, assignment string, at time.Time) {
	b, ok := buckets[key]
	if !ok {
		b = &errorBucket{FrequentError: FrequentError{Message: key, Explanation: explanation, Assignments: []string{}},
			students: map[string]bool{}, assignments: map[string]bool{}}
		buckets[key] = b
	}
	b.Count++
	b.students[student] = true
	if at.After(b.LastAt) {
		b.LastAt = at
	}
	if assignment != "" && !b.assignments[assignment] && len(b.Assignments) < maxExampleAssignments {
		b.assignments[assignment] = true
		b.Assignments = append(b.Assignments, assignment)
	}
}

func rankBuckets(buckets map[string]*errorBucket) []FrequentError {
	out := make([]FrequentError, 0, len(buckets))
	for _, b := range buckets {
		b.Students = len(b.students)
		out = append(out, b.FrequentError)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Students != out[j].Students {
			return out[i].Students > out[j].Students
		}
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Message < out[j].Message
	})
	if len(out) > maxFrequentErrors {
		out = out[:maxFrequentErrors]
	}
	return out
}

func (s *Store) reportGroups(teacherID, groupID string, wholeCourse bool) ([]string, error) {
	g, err := s.ownedGroup(s.db, teacherID, groupID)
	if err != nil {
		return nil, err
	}
	if !wholeCourse {
		return []string{groupID}, nil
	}
	return queryIDs(s.db, `SELECT id FROM class_groups WHERE course_id = ?`, g.CourseID)
}

func (s *Store) FrequentErrors(teacherID, groupID string, wholeCourse bool) (*ErrorReport, error) {
	groups, err := s.reportGroups(teacherID, groupID, wholeCourse)
	if err != nil {
		return nil, err
	}
	report := &ErrorReport{Scope: "clase", Groups: len(groups), Compile: []FrequentError{}, Runtime: []FrequentError{}, Tests: []HardTest{}}
	if wholeCourse {
		report.Scope = "materia"
	}
	args, ph := inArgs(groups)
	rows, err := s.db.Query(`SELECT s.student_id, s.created_at, s.success, s.error_summary, s.run_error, s.exit_code, COALESCE(a.title, '')
		FROM submissions s LEFT JOIN assignments a ON a.id = s.assignment_id
		WHERE s.group_id IN (`+ph+`)`, args...)
	if err != nil {
		return nil, err
	}
	compile, runtime := map[string]*errorBucket{}, map[string]*errorBucket{}
	withErrors := map[string]bool{}
	for rows.Next() {
		var student, created, summary, runErr, title string
		var success bool
		var exit int
		if err := rows.Scan(&student, &created, &success, &summary, &runErr, &exit, &title); err != nil {
			rows.Close()
			return nil, err
		}
		report.Analyzed++
		at := parseTime(created)
		switch {
		case !success && summary != "":
			msg := compiler.NormalizeError(summary)
			addToBucket(compile, msg, compiler.ExplainError(msg), student, title, at)
			withErrors[student] = true
		case success && (runErr != "" || exit != 0):
			msg, explanation := compiler.DescribeRuntime(runErr, exit)
			addToBucket(runtime, msg, explanation, student, title, at)
			withErrors[student] = true
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	report.Compile, report.Runtime, report.StudentsWithErrors = rankBuckets(compile), rankBuckets(runtime), len(withErrors)
	if report.Tests, err = s.hardTests(args, ph); err != nil {
		return nil, err
	}
	return report, nil
}

func (s *Store) hardTests(args []any, ph string) ([]HardTest, error) {
	rows, err := s.db.Query(`SELECT t.id, t.name, t.hidden, a.title, s.student_id, st.passed
		FROM submission_tests st
		JOIN submissions s ON s.id = st.submission_id
		JOIN assignment_tests t ON t.id = st.test_id
		JOIN assignments a ON a.id = t.assignment_id
		WHERE s.group_id IN (`+ph+`) ORDER BY s.created_at`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type acc struct {
		HardTest
		latest map[string]bool
	}
	tests := map[string]*acc{}
	var order []string
	for rows.Next() {
		var id, name, title, student string
		var hidden, passed bool
		if err := rows.Scan(&id, &name, &hidden, &title, &student, &passed); err != nil {
			return nil, err
		}
		t, ok := tests[id]
		if !ok {
			t = &acc{HardTest: HardTest{Assignment: title, Test: name, Hidden: hidden}, latest: map[string]bool{}}
			tests[id] = t
			order = append(order, id)
		}
		t.Runs++
		if !passed {
			t.Failures++
		}
		t.latest[student] = passed
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := []HardTest{}
	for _, id := range order {
		t := tests[id]
		t.Students = len(t.latest)
		for _, ok := range t.latest {
			if !ok {
				t.NotPassing++
			}
		}
		if t.Failures > 0 {
			out = append(out, t.HardTest)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].NotPassing != out[j].NotPassing {
			return out[i].NotPassing > out[j].NotPassing
		}
		return out[i].Failures > out[j].Failures
	})
	if len(out) > maxHardTests {
		out = out[:maxHardTests]
	}
	return out, nil
}
