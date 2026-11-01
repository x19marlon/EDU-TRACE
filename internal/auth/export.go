package auth

import (
	"database/sql"
	"math"
	"sort"
	"time"
)

type GradeSheet struct {
	GroupName string
	CourseName string
	Assignments []Assignment
	Rows []GradeRow
}

type GradeRow struct {
	Name string
	Email string
	Grades map[string]*float64
	Average *float64
}

type ReviewExport struct {
	StudentName string
	Email string
	Assignment string
	SubmittedAt time.Time
	LateSeconds int64
	TestsPassed int
	TestsTotal int
	Grade *float64
	ReturnedAt *time.Time
	Origin string
	AIGrade *float64
	ReviewMs int64
	Feedback string
	AIModel string
}

func (s *Store) GradeSheet(teacherID, groupID string) (*GradeSheet, error) {
	g, err := s.ownedGroup(s.db, teacherID, groupID)
	if err != nil {
		return nil, err
	}
	if err := s.fillGroup(g); err != nil {
		return nil, err
	}
	assignments, err := s.groupAssignments(groupID, false)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(assignments, func(i, j int) bool { return assignments[i].CreatedAt.Before(assignments[j].CreatedAt) })
	sheet := &GradeSheet{GroupName: g.Name, CourseName: g.CourseName, Assignments: assignments}
	index := map[string]int{}
	for _, st := range g.Students {
		index[st.ID] = len(sheet.Rows)
		sheet.Rows = append(sheet.Rows, GradeRow{Name: st.Name, Email: st.Email, Grades: map[string]*float64{}})
	}
	rows, err := s.db.Query(`SELECT s.student_id, s.assignment_id, r.grade FROM submissions s
		JOIN reviews r ON r.submission_id = s.id AND r.status = 'returned'
		WHERE s.group_id = ? AND s.kind = 'official' AND s.assignment_id IS NOT NULL
		ORDER BY s.created_at`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var student, assignment string
		var grade sql.NullFloat64
		if err := rows.Scan(&student, &assignment, &grade); err != nil {
			return nil, err
		}
		if i, ok := index[student]; ok && grade.Valid {
			sheet.Rows[i].Grades[assignment] = floatPtr(grade)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range sheet.Rows {
		if n := len(sheet.Rows[i].Grades); n > 0 {
			sum := 0.0
			for _, g := range sheet.Rows[i].Grades {
				sum += *g
			}
			avg := math.Round(sum/float64(n)*100) / 100
			sheet.Rows[i].Average = &avg
		}
	}
	return sheet, nil
}

func (s *Store) ReviewsExport(teacherID, groupID string) ([]ReviewExport, error) {
	if _, err := s.ownedGroup(s.db, teacherID, groupID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT `+metaCols+`, u.name, u.email, COALESCE(a.title, ''),
			r.grade, r.returned_at, r.feedback_origin, r.ai_grade, r.review_ms, r.feedback,
			CASE WHEN r.feedback_origin IN ('ia', 'ia_editada') THEN r.ai_model ELSE '' END
		FROM submissions s
		JOIN users u ON u.id = s.student_id
		JOIN reviews r ON r.submission_id = s.id AND r.status = 'returned'
		LEFT JOIN assignments a ON a.id = s.assignment_id
		WHERE s.group_id = ? AND s.kind = 'official'
		ORDER BY u.name, s.created_at`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ReviewExport{}
	for rows.Next() {
		var e ReviewExport
		var grade, aiGrade sql.NullFloat64
		var returned sql.NullString
		m, err := scanMeta(rows, &e.StudentName, &e.Email, &e.Assignment, &grade, &returned, &e.Origin, &aiGrade, &e.ReviewMs, &e.Feedback, &e.AIModel)
		if err != nil {
			return nil, err
		}
		e.SubmittedAt, e.LateSeconds, e.TestsPassed, e.TestsTotal = m.CreatedAt, m.LateSeconds, m.TestsPassed, m.TestsTotal
		e.Grade, e.ReturnedAt, e.AIGrade = floatPtr(grade), timePtr(returned), floatPtr(aiGrade)
		out = append(out, e)
	}
	return out, rows.Err()
}
