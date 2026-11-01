package auth

import (
	"database/sql"
	"sort"
	"time"
)

type ReviewEstimate struct {
	AssignmentID string `json:"assignment_id"`
	Title string `json:"title"`
	DueAt *time.Time `json:"due_at,omitempty"`
	Students int `json:"students"`
	Returned int `json:"returned"`
	Pending int `json:"pending"`
	WithAI int `json:"with_ai"`
	NotCompiling int `json:"not_compiling"`
	FailingTests int `json:"failing_tests"`
	AllPassing int `json:"all_passing"`
	TotalMinutes float64 `json:"total_minutes"`
	PerSubmission float64 `json:"per_submission"`
}

type ReviewPlan struct {
	Assignments []ReviewEstimate `json:"assignments"`
	Measured int `json:"measured"`
	MeasuredMinutes *float64 `json:"measured_minutes,omitempty"`
	EstimatedMinutes *float64 `json:"estimated_minutes,omitempty"`
}

type planRow struct {
	assignment string
	success bool
	failed bool
	lines int
	testsTotal int
	testsPassed int
	status string
	aiMinutes sql.NullInt64
	reviewMs int64
}

func fallbackMinutes(r planRow) float64 {
	minutes := 3.0
	switch {
	case !r.success:
		minutes = 6
	case r.failed || r.testsPassed < r.testsTotal:
		minutes = 5
	}
	if r.lines > 60 {
		minutes += 2
	}
	return minutes
}

func median(values []float64) *float64 {
	if len(values) == 0 {
		return nil
	}
	sort.Float64s(values)
	m := values[len(values)/2]
	if len(values)%2 == 0 {
		m = (values[len(values)/2-1] + values[len(values)/2]) / 2
	}
	return &m
}

func (s *Store) ReviewPlan(teacherID, groupID string) (*ReviewPlan, error) {
	if _, err := s.ownedGroup(s.db, teacherID, groupID); err != nil {
		return nil, err
	}
	assignments, err := s.groupAssignments(groupID, false)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT s.student_id, s.assignment_id, s.success, s.run_error != '' OR s.exit_code != 0, s.lines,
			(SELECT COUNT(*) FROM submission_tests st WHERE st.submission_id = s.id),
			(SELECT COALESCE(SUM(st.passed), 0) FROM submission_tests st WHERE st.submission_id = s.id),
			COALESCE(r.status, ''), r.ai_minutes, COALESCE(r.review_ms, 0)
		FROM submissions s LEFT JOIN reviews r ON r.submission_id = s.id
		WHERE s.group_id = ? AND s.kind = 'official' AND s.assignment_id IS NOT NULL
		ORDER BY s.created_at`, groupID)
	if err != nil {
		return nil, err
	}
	latest := map[[2]string]planRow{}
	var measured, estimated []float64
	for rows.Next() {
		var r planRow
		var student string
		if err := rows.Scan(&student, &r.assignment, &r.success, &r.failed, &r.lines, &r.testsTotal, &r.testsPassed,
			&r.status, &r.aiMinutes, &r.reviewMs); err != nil {
			rows.Close()
			return nil, err
		}
		latest[[2]string{student, r.assignment}] = r
		if r.status == ReviewReturned && r.reviewMs > 0 {
			measured = append(measured, float64(r.reviewMs)/60000)
			if r.aiMinutes.Valid {
				estimated = append(estimated, float64(r.aiMinutes.Int64))
			}
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	plan := &ReviewPlan{Assignments: []ReviewEstimate{}, Measured: len(measured), MeasuredMinutes: median(measured), EstimatedMinutes: median(estimated)}

	byAssignment := map[string]*ReviewEstimate{}
	for _, a := range assignments {
		byAssignment[a.ID] = &ReviewEstimate{AssignmentID: a.ID, Title: a.Title, DueAt: a.DueAt}
	}
	for _, r := range latest {
		e, ok := byAssignment[r.assignment]
		if !ok {
			continue
		}
		e.Students++
		if r.status == ReviewReturned {
			e.Returned++
			continue
		}
		e.Pending++
		switch {
		case !r.success:
			e.NotCompiling++
		case r.testsTotal > 0 && r.testsPassed == r.testsTotal:
			e.AllPassing++
		case r.testsTotal > 0:
			e.FailingTests++
		}
		if r.aiMinutes.Valid {
			e.WithAI++
			e.TotalMinutes += float64(r.aiMinutes.Int64)
		} else if plan.MeasuredMinutes != nil {
			e.TotalMinutes += *plan.MeasuredMinutes
		} else {
			e.TotalMinutes += fallbackMinutes(r)
		}
	}
	for _, a := range assignments {
		e := byAssignment[a.ID]
		if e.Students == 0 {
			continue
		}
		if e.Pending > 0 {
			e.PerSubmission = e.TotalMinutes / float64(e.Pending)
		}
		plan.Assignments = append(plan.Assignments, *e)
	}
	sort.SliceStable(plan.Assignments, func(i, j int) bool {
		a, b := plan.Assignments[i], plan.Assignments[j]
		if (a.Pending > 0) != (b.Pending > 0) {
			return a.Pending > 0
		}
		if a.DueAt != nil && b.DueAt != nil {
			return a.DueAt.Before(*b.DueAt)
		}
		return a.DueAt != nil
	})
	return plan, nil
}
