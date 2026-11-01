package auth

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	alertInactiveDays = 14
	alertFailStreak = 8
	alertLowRAEs = 2
	alertLowAverage = 3.0
	alertMinGrades = 2
)

type StudentAlert struct {
	StudentID string `json:"student_id"`
	Name string `json:"name"`
	Reasons []string `json:"reasons"`
}

type latestGrade struct {
	assignmentID string
	grade *float64
	returnedAt time.Time
}

type studentFacts struct {
	joined time.Time
	lastSubmission time.Time
	streak int
	streakOpen bool
	submitted map[string]bool
	grades map[string]latestGrade
	raeLevels map[string]string
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func (s *Store) groupFacts(groupID string, since time.Time) (map[string]*studentFacts, map[string]string, error) {
	facts := map[string]*studentFacts{}
	names := map[string]string{}
	rows, err := s.db.Query(`SELECT u.id, u.name, gs.joined_at FROM group_students gs JOIN users u ON u.id = gs.student_id
		WHERE gs.group_id = ? ORDER BY u.name`, groupID)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var id, name, joined string
		if err := rows.Scan(&id, &name, &joined); err != nil {
			rows.Close()
			return nil, nil, err
		}
		names[id] = name
		facts[id] = &studentFacts{joined: parseTime(joined), streakOpen: true, submitted: map[string]bool{}, grades: map[string]latestGrade{}, raeLevels: map[string]string{}}
	}
	rows.Close()

	rows, err = s.db.Query(`SELECT student_id, kind, COALESCE(assignment_id, ''), success, created_at FROM submissions
		WHERE group_id = ? ORDER BY created_at DESC`, groupID)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var student, kind, assignment, created string
		var success bool
		if err := rows.Scan(&student, &kind, &assignment, &success, &created); err != nil {
			rows.Close()
			return nil, nil, err
		}
		f := facts[student]
		if f == nil {
			continue
		}
		at := parseTime(created)
		if at.After(f.lastSubmission) {
			f.lastSubmission = at
		}
		if kind == KindOfficial && assignment != "" {
			f.submitted[assignment] = true
		}
		if f.streakOpen && at.After(since) {
			if success {
				f.streakOpen = false
			} else {
				f.streak++
			}
		}
	}
	rows.Close()

	rows, err = s.db.Query(`SELECT s.student_id, COALESCE(s.assignment_id, s.id), r.grade, r.returned_at FROM reviews r
		JOIN submissions s ON s.id = r.submission_id
		WHERE s.group_id = ? AND r.status = 'returned' ORDER BY r.returned_at`, groupID)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var student, assignment string
		var grade sql.NullFloat64
		var returned sql.NullString
		if err := rows.Scan(&student, &assignment, &grade, &returned); err != nil {
			rows.Close()
			return nil, nil, err
		}
		if f := facts[student]; f != nil {
			f.grades[assignment] = latestGrade{assignmentID: assignment, grade: floatPtr(grade), returnedAt: parseTime(returned.String)}
		}
	}
	rows.Close()

	rows, err = s.db.Query(`SELECT s.student_id, ra.code, rr.level FROM review_raes rr
		JOIN reviews r ON r.submission_id = rr.submission_id AND r.status = 'returned'
		JOIN submissions s ON s.id = rr.submission_id
		JOIN course_raes ra ON ra.id = rr.rae_id
		WHERE s.group_id = ? AND rr.level != '' ORDER BY r.returned_at`, groupID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var student, code, level string
		if err := rows.Scan(&student, &code, &level); err != nil {
			return nil, nil, err
		}
		if f := facts[student]; f != nil {
			f.raeLevels[code] = level
		}
	}
	return facts, names, rows.Err()
}

func average(grades map[string]latestGrade) (float64, int) {
	var sum float64
	n := 0
	for _, g := range grades {
		if g.grade != nil {
			sum += *g.grade
			n++
		}
	}
	if n == 0 {
		return 0, 0
	}
	return sum / float64(n), n
}

func (f *studentFacts) reasons(now time.Time, due []string) []string {
	var out []string
	missing := 0
	for _, id := range due {
		if !f.submitted[id] {
			missing++
		}
	}
	if missing > 0 {
		out = append(out, fmt.Sprintf("No entregó %d %s con la fecha límite vencida", missing, plural(missing, "taller", "talleres")))
	}
	inactive := alertInactiveDays * 24 * time.Hour
	if now.Sub(f.joined) > inactive {
		if f.lastSubmission.IsZero() {
			out = append(out, "No ha compilado ni enviado nada en esta clase")
		} else if now.Sub(f.lastSubmission) > inactive {
			out = append(out, fmt.Sprintf("Sin actividad en la clase desde hace %d días", int(now.Sub(f.lastSubmission).Hours()/24)))
		}
	}
	if f.streak >= alertFailStreak {
		out = append(out, fmt.Sprintf("Sus últimas %d compilaciones tienen errores", f.streak))
	}
	var low []string
	for code, level := range f.raeLevels {
		if level == "bajo" {
			low = append(low, code)
		}
	}
	if len(low) >= alertLowRAEs {
		sort.Strings(low)
		out = append(out, "Nivel bajo en "+strings.Join(low, ", "))
	}
	if avg, n := average(f.grades); n >= alertMinGrades && avg < alertLowAverage {
		out = append(out, fmt.Sprintf("Promedio de %.1f en %d talleres revisados", round1(avg), n))
	}
	return out
}

func (s *Store) dueAssignments(groupID string, now time.Time) ([]string, error) {
	return queryIDs(s.db, `SELECT id FROM assignments WHERE group_id = ? AND due_at IS NOT NULL AND due_at < ?`, groupID, fmtTime(now))
}

func (s *Store) StudentAlerts(teacherID, groupID string, now time.Time) ([]StudentAlert, error) {
	if _, err := s.ownedGroup(s.db, teacherID, groupID); err != nil {
		return nil, err
	}
	due, err := s.dueAssignments(groupID, now)
	if err != nil {
		return nil, err
	}
	facts, names, err := s.groupFacts(groupID, now.Add(-7*24*time.Hour))
	if err != nil {
		return nil, err
	}
	out := []StudentAlert{}
	for id, f := range facts {
		if r := f.reasons(now, due); len(r) > 0 {
			out = append(out, StudentAlert{StudentID: id, Name: names[id], Reasons: r})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if len(out[i].Reasons) != len(out[j].Reasons) {
			return len(out[i].Reasons) > len(out[j].Reasons)
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

type ReportAssignment struct {
	Title string `json:"title"`
	DueAt *time.Time `json:"due_at,omitempty"`
	SubmittedAt *time.Time `json:"submitted_at,omitempty"`
	LateSeconds int64 `json:"late_seconds,omitempty"`
	Submissions int `json:"submissions"`
	TestsPassed int `json:"tests_passed"`
	TestsTotal int `json:"tests_total"`
	Grade *float64 `json:"grade,omitempty"`
	ReturnedAt *time.Time `json:"returned_at,omitempty"`
	Origin string `json:"origin,omitempty"`
	Criteria []ReviewCriterion `json:"criteria,omitempty"`
}

type ReportRAE struct {
	Code string `json:"code"`
	Description string `json:"description"`
	Level string `json:"level,omitempty"`
}

type StudentReport struct {
	Name string `json:"name"`
	Email string `json:"email"`
	CourseName string `json:"course_name"`
	GroupName string `json:"group_name"`
	GeneratedAt time.Time `json:"generated_at"`
	Assignments []ReportAssignment `json:"assignments"`
	RAEs []ReportRAE `json:"raes"`
	Average *float64 `json:"average,omitempty"`
	Attempts int `json:"attempts"`
	Officials int `json:"officials"`
	LastActivity *time.Time `json:"last_activity,omitempty"`
	Questions int `json:"questions"`
	Alerts []string `json:"alerts"`
}

func (s *Store) StudentReport(teacherID, groupID, studentID string, now time.Time) (*StudentReport, error) {
	g, err := s.ownedGroup(s.db, teacherID, groupID)
	if err != nil {
		return nil, err
	}
	rep := &StudentReport{CourseName: g.CourseName, GroupName: g.Name, GeneratedAt: now.UTC(), Assignments: []ReportAssignment{}, RAEs: []ReportRAE{}, Alerts: []string{}}
	err = s.db.QueryRow(`SELECT u.name, u.email FROM group_students gs JOIN users u ON u.id = gs.student_id
		WHERE gs.group_id = ? AND gs.student_id = ?`, groupID, studentID).Scan(&rep.Name, &rep.Email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var last sql.NullString
	if err := s.db.QueryRow(`SELECT COALESCE(SUM(kind = 'attempt'), 0), COALESCE(SUM(kind = 'official'), 0), MAX(created_at),
			(SELECT COUNT(*) FROM review_questions q JOIN submissions qs ON qs.id = q.submission_id WHERE qs.group_id = ? AND qs.student_id = ?)
		FROM submissions WHERE group_id = ? AND student_id = ?`, groupID, studentID, groupID, studentID).
		Scan(&rep.Attempts, &rep.Officials, &last, &rep.Questions); err != nil {
		return nil, err
	}
	rep.LastActivity = timePtr(last)

	assignments, err := s.groupAssignments(groupID, false)
	if err != nil {
		return nil, err
	}
	byID := map[string]int{}
	for i, a := range assignments {
		byID[a.ID] = i
		rep.Assignments = append(rep.Assignments, ReportAssignment{Title: a.Title, DueAt: a.DueAt})
	}
	rows, err := s.db.Query(`SELECT `+metaCols+`, r.grade, r.returned_at, COALESCE(r.feedback_origin, '')
		FROM submissions s LEFT JOIN reviews r ON r.submission_id = s.id AND r.status = 'returned'
		WHERE s.group_id = ? AND s.student_id = ? AND s.kind = 'official' AND s.assignment_id IS NOT NULL
		ORDER BY s.created_at`, groupID, studentID)
	if err != nil {
		return nil, err
	}
	latest := map[int]string{}
	for rows.Next() {
		var grade sql.NullFloat64
		var returned sql.NullString
		var origin string
		m, err := scanMeta(rows, &grade, &returned, &origin)
		if err != nil {
			rows.Close()
			return nil, err
		}
		i, ok := byID[m.AssignmentID]
		if !ok {
			continue
		}
		a := &rep.Assignments[i]
		at := m.CreatedAt
		a.Submissions++
		a.SubmittedAt, a.LateSeconds, a.TestsPassed, a.TestsTotal = &at, m.LateSeconds, m.TestsPassed, m.TestsTotal
		if returned.Valid {
			a.Grade, a.ReturnedAt, a.Origin = floatPtr(grade), timePtr(returned), origin
			latest[i] = m.ID
		}
	}
	rows.Close()
	var sum float64
	graded := 0
	for i, id := range latest {
		if rep.Assignments[i].ReturnedAt == nil {
			continue
		}
		if rep.Assignments[i].Criteria, err = reviewCriteria(s.db, id); err != nil {
			return nil, err
		}
		if g := rep.Assignments[i].Grade; g != nil {
			sum += *g
			graded++
		}
	}
	if graded > 0 {
		avg := round1(sum / float64(graded))
		rep.Average = &avg
	}

	due, err := s.dueAssignments(groupID, now)
	if err != nil {
		return nil, err
	}
	facts, _, err := s.groupFacts(groupID, now.Add(-7*24*time.Hour))
	if err != nil {
		return nil, err
	}
	f := facts[studentID]
	raes, err := s.db.Query(`SELECT code, description FROM course_raes WHERE course_id = ? ORDER BY position, created_at`, g.CourseID)
	if err != nil {
		return nil, err
	}
	defer raes.Close()
	for raes.Next() {
		var r ReportRAE
		if err := raes.Scan(&r.Code, &r.Description); err != nil {
			return nil, err
		}
		if f != nil {
			r.Level = f.raeLevels[r.Code]
		}
		rep.RAEs = append(rep.RAEs, r)
	}
	if f != nil {
		rep.Alerts = append(rep.Alerts, f.reasons(now, due)...)
	}
	return rep, raes.Err()
}

type ProgressPoint struct {
	Title string `json:"title"`
	ReturnedAt time.Time `json:"returned_at"`
	Grade *float64 `json:"grade,omitempty"`
	TestsPassed int `json:"tests_passed"`
	TestsTotal int `json:"tests_total"`
	Levels map[string]string `json:"levels"`
}

type ProgressRAE struct {
	Code string `json:"code"`
	Description string `json:"description"`
	Latest string `json:"latest,omitempty"`
}

type ProgressGroup struct {
	GroupID string `json:"group_id"`
	GroupName string `json:"group_name"`
	CourseName string `json:"course_name"`
	Points []ProgressPoint `json:"points"`
	RAEs []ProgressRAE `json:"raes"`
	Average *float64 `json:"average,omitempty"`
}

func (s *Store) StudentProgress(studentID string) ([]ProgressGroup, error) {
	groups, err := s.studentGroups(studentID)
	if err != nil {
		return nil, err
	}
	out := []ProgressGroup{}
	for _, g := range groups {
		pg := ProgressGroup{GroupID: g.ID, GroupName: g.Name, CourseName: g.CourseName, Points: []ProgressPoint{}, RAEs: []ProgressRAE{}}
		rows, err := s.db.Query(`SELECT `+metaCols+`, COALESCE(a.title, 'Práctica libre'), r.grade, r.returned_at
			FROM submissions s JOIN reviews r ON r.submission_id = s.id AND r.status = 'returned'
			LEFT JOIN assignments a ON a.id = s.assignment_id
			WHERE s.group_id = ? AND s.student_id = ? ORDER BY r.returned_at`, g.ID, studentID)
		if err != nil {
			return nil, err
		}
		var ids []string
		for rows.Next() {
			var p ProgressPoint
			var grade sql.NullFloat64
			var returned string
			m, err := scanMeta(rows, &p.Title, &grade, &returned)
			if err != nil {
				rows.Close()
				return nil, err
			}
			p.Grade, p.ReturnedAt, p.TestsPassed, p.TestsTotal, p.Levels = floatPtr(grade), parseTime(returned), m.TestsPassed, m.TestsTotal, map[string]string{}
			pg.Points = append(pg.Points, p)
			ids = append(ids, m.ID)
		}
		rows.Close()
		var sum float64
		n := 0
		for i, id := range ids {
			lrows, err := s.db.Query(`SELECT ra.code, rr.level FROM review_raes rr JOIN course_raes ra ON ra.id = rr.rae_id
				WHERE rr.submission_id = ? AND rr.level != ''`, id)
			if err != nil {
				return nil, err
			}
			for lrows.Next() {
				var code, level string
				if err := lrows.Scan(&code, &level); err != nil {
					lrows.Close()
					return nil, err
				}
				pg.Points[i].Levels[code] = level
			}
			lrows.Close()
			if g := pg.Points[i].Grade; g != nil {
				sum += *g
				n++
			}
		}
		if n > 0 {
			avg := round1(sum / float64(n))
			pg.Average = &avg
		}
		rrows, err := s.db.Query(`SELECT ra.code, ra.description FROM course_raes ra JOIN class_groups cg ON cg.course_id = ra.course_id
			WHERE cg.id = ? ORDER BY ra.position, ra.created_at`, g.ID)
		if err != nil {
			return nil, err
		}
		for rrows.Next() {
			var r ProgressRAE
			if err := rrows.Scan(&r.Code, &r.Description); err != nil {
				rrows.Close()
				return nil, err
			}
			for _, p := range pg.Points {
				if l := p.Levels[r.Code]; l != "" {
					r.Latest = l
				}
			}
			pg.RAEs = append(pg.RAEs, r)
		}
		rrows.Close()
		out = append(out, pg)
	}
	return out, nil
}

type PracticeResult struct {
	ExerciseID string `json:"exercise_id"`
	Attempts int `json:"attempts"`
	BestPassed int `json:"best_passed"`
	Total int `json:"total"`
	SolvedAt *time.Time `json:"solved_at,omitempty"`
	LastAt time.Time `json:"last_at"`
}

func (s *Store) RecordPractice(studentID, exerciseID string, passed, total int) error {
	now := fmtTime(time.Now())
	var solved any
	if total > 0 && passed == total {
		solved = now
	}
	_, err := s.db.Exec(`INSERT INTO practice_results (student_id, exercise_id, attempts, best_passed, total, solved_at, last_at)
		VALUES (?, ?, 1, ?, ?, ?, ?)
		ON CONFLICT(student_id, exercise_id) DO UPDATE SET attempts = attempts + 1, best_passed = MAX(best_passed, excluded.best_passed),
			total = excluded.total, solved_at = COALESCE(solved_at, excluded.solved_at), last_at = excluded.last_at`,
		studentID, exerciseID, passed, total, solved, now)
	return err
}

func (s *Store) PracticeResults(studentID string) (map[string]PracticeResult, error) {
	rows, err := s.db.Query(`SELECT exercise_id, attempts, best_passed, total, solved_at, last_at FROM practice_results WHERE student_id = ?`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]PracticeResult{}
	for rows.Next() {
		var r PracticeResult
		var solved sql.NullString
		var last string
		if err := rows.Scan(&r.ExerciseID, &r.Attempts, &r.BestPassed, &r.Total, &solved, &last); err != nil {
			return nil, err
		}
		r.SolvedAt, r.LastAt = timePtr(solved), parseTime(last)
		out[r.ExerciseID] = r
	}
	return out, rows.Err()
}

type SimilarityDoc struct {
	SubmissionID string
	StudentID string
	StudentName string
	Code string
}

func (s *Store) SimilarityInput(teacherID, groupID, assignmentID string) ([]SimilarityDoc, error) {
	if _, err := s.ownedGroup(s.db, teacherID, groupID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT s.id, s.student_id, u.name, s.code FROM submissions s
		JOIN users u ON u.id = s.student_id
		WHERE s.group_id = ? AND s.assignment_id = ? AND s.kind = 'official'
			AND s.created_at = (SELECT MAX(s2.created_at) FROM submissions s2
				WHERE s2.group_id = s.group_id AND s2.assignment_id = s.assignment_id AND s2.student_id = s.student_id AND s2.kind = 'official')`,
		groupID, assignmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SimilarityDoc
	for rows.Next() {
		var d SimilarityDoc
		if err := rows.Scan(&d.SubmissionID, &d.StudentID, &d.StudentName, &d.Code); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
