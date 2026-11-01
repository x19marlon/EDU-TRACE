package auth

import (
	"database/sql"
	"errors"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	LevelLow = "bajo"
	LevelMid = "medio"
	LevelHigh = "alto"
)

var levelScore = map[string]float64{LevelLow: 1, LevelMid: 2, LevelHigh: 3}

func ValidLevel(level string) bool {
	return level == "" || levelScore[level] > 0
}

type RAE struct {
	ID string `json:"id"`
	CourseID string `json:"course_id"`
	Code string `json:"code"`
	Description string `json:"description"`
}

type ReviewRAE struct {
	RAEID string `json:"rae_id"`
	Code string `json:"code"`
	Description string `json:"description"`
	Level string `json:"level,omitempty"`
	AILevel string `json:"ai_level,omitempty"`
}

type StudentRAE struct {
	Code string `json:"code"`
	Description string `json:"description"`
	Level string `json:"level"`
}

func scanRAEs(rows *sql.Rows) ([]RAE, error) {
	defer rows.Close()
	out := []RAE{}
	for rows.Next() {
		var r RAE
		if err := rows.Scan(&r.ID, &r.CourseID, &r.Code, &r.Description); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func ownsCourse(q querier, teacherID, courseID string) error {
	var n int
	if err := q.QueryRow(`SELECT COUNT(*) FROM course_staff WHERE course_id = ? AND teacher_id = ?`, courseID, teacherID).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) CourseRAEs(teacherID, courseID string) ([]RAE, error) {
	if err := ownsCourse(s.db, teacherID, courseID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT id, course_id, code, description FROM course_raes
		WHERE course_id = ? ORDER BY position, created_at`, courseID)
	if err != nil {
		return nil, err
	}
	return scanRAEs(rows)
}

func (s *Store) teacherRAEs(teacherID string) (map[string][]RAE, error) {
	rows, err := s.db.Query(`SELECT r.id, r.course_id, r.code, r.description FROM course_raes r
		JOIN courses c ON c.id = r.course_id
		WHERE c.id IN (SELECT course_id FROM course_staff WHERE teacher_id = ?) ORDER BY r.position, r.created_at`, teacherID)
	if err != nil {
		return nil, err
	}
	list, err := scanRAEs(rows)
	if err != nil {
		return nil, err
	}
	out := map[string][]RAE{}
	for _, r := range list {
		out[r.CourseID] = append(out[r.CourseID], r)
	}
	return out, nil
}

func (s *Store) CreateRAE(teacherID, courseID, code, description string) (*RAE, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := ownsCourse(tx, teacherID, courseID); err != nil {
		return nil, err
	}
	id, err := randomHex(12)
	if err != nil {
		return nil, err
	}
	r := &RAE{ID: id, CourseID: courseID, Code: strings.TrimSpace(code), Description: strings.TrimSpace(description)}
	if _, err := tx.Exec(`INSERT INTO course_raes (id, course_id, code, description, position, created_at)
		VALUES (?, ?, ?, ?, (SELECT COALESCE(MAX(position), -1) + 1 FROM course_raes WHERE course_id = ?), ?)`,
		r.ID, r.CourseID, r.Code, r.Description, courseID, fmtTime(time.Now())); err != nil {
		return nil, err
	}
	return r, tx.Commit()
}

func (s *Store) UpdateRAE(teacherID, raeID, code, description string) error {
	return affected(s.db.Exec(`UPDATE course_raes SET code = ?, description = ?
		WHERE id = ? AND course_id IN (SELECT course_id FROM course_staff WHERE teacher_id = ?)`,
		strings.TrimSpace(code), strings.TrimSpace(description), raeID, teacherID))
}

func (s *Store) DeleteRAE(teacherID, raeID string) error {
	return affected(s.db.Exec(`DELETE FROM course_raes
		WHERE id = ? AND course_id IN (SELECT course_id FROM course_staff WHERE teacher_id = ?)`, raeID, teacherID))
}

func reviewRAEs(q querier, submissionID string) ([]ReviewRAE, error) {
	rows, err := q.Query(`SELECT r.id, r.code, r.description, COALESCE(l.level, ''), COALESCE(l.ai_level, '')
		FROM submissions s
		JOIN class_groups g ON g.id = s.group_id
		JOIN course_raes r ON r.course_id = g.course_id
		LEFT JOIN review_raes l ON l.rae_id = r.id AND l.submission_id = s.id
		WHERE s.id = ? ORDER BY r.position, r.created_at`, submissionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ReviewRAE{}
	for rows.Next() {
		var r ReviewRAE
		if err := rows.Scan(&r.RAEID, &r.Code, &r.Description, &r.Level, &r.AILevel); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

const raeOfSubmission = `SELECT ?, r.id, %s FROM course_raes r
	JOIN class_groups g ON g.course_id = r.course_id
	JOIN submissions s ON s.group_id = g.id
	WHERE s.id = ? AND r.id = ?`

func saveTeacherLevels(q querier, submissionID string, levels map[string]string) error {
	for raeID, level := range levels {
		if !ValidLevel(level) {
			return errors.New("nivel de RAE inválido")
		}
		if _, err := q.Exec(`INSERT INTO review_raes (submission_id, rae_id, level)
			`+strings.Replace(raeOfSubmission, "%s", "?", 1)+`
			ON CONFLICT(submission_id, rae_id) DO UPDATE SET level = excluded.level`,
			submissionID, level, submissionID, raeID); err != nil {
			return err
		}
	}
	return nil
}

func saveAILevels(q querier, submissionID string, levels map[string]string, prefill bool) error {
	for raeID, level := range levels {
		if !ValidLevel(level) {
			return errors.New("nivel de RAE inválido")
		}
		initial := ""
		if prefill {
			initial = level
		}
		if _, err := q.Exec(`INSERT INTO review_raes (submission_id, rae_id, level, ai_level)
			`+strings.Replace(raeOfSubmission, "%s", "?, ?", 1)+`
			ON CONFLICT(submission_id, rae_id) DO UPDATE SET ai_level = excluded.ai_level,
				level = CASE WHEN review_raes.level = '' THEN excluded.level ELSE review_raes.level END`,
			submissionID, initial, level, submissionID, raeID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) returnedRAELevels(submissionIDs []string) (map[string][]StudentRAE, error) {
	out := map[string][]StudentRAE{}
	if len(submissionIDs) == 0 {
		return out, nil
	}
	args, ph := inArgs(submissionIDs)
	rows, err := s.db.Query(`SELECT l.submission_id, r.code, r.description, l.level
		FROM review_raes l
		JOIN course_raes r ON r.id = l.rae_id
		JOIN reviews v ON v.submission_id = l.submission_id AND v.status = 'returned'
		WHERE l.level != '' AND l.submission_id IN (`+ph+`)
		ORDER BY r.position, r.created_at`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var r StudentRAE
		if err := rows.Scan(&id, &r.Code, &r.Description, &r.Level); err != nil {
			return nil, err
		}
		out[id] = append(out[id], r)
	}
	return out, rows.Err()
}

type StudentPoint struct {
	ID string `json:"id"`
	Name string `json:"name"`
	AverageGrade *float64 `json:"average_grade,omitempty"`
	Graded int `json:"graded"`
	Levels map[string]string `json:"levels"`
}

type RAEMatrix struct {
	RAEs []RAE `json:"raes"`
	Students []StudentPoint `json:"students"`
}

func levelFromScore(score float64) string {
	switch math.Round(score) {
	case 1:
		return LevelLow
	case 2:
		return LevelMid
	}
	return LevelHigh
}

func (s *Store) GroupRAEMatrix(teacherID, groupID string) (*RAEMatrix, error) {
	g, err := s.ownedGroup(s.db, teacherID, groupID)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT id, course_id, code, description FROM course_raes
		WHERE course_id = ? ORDER BY position, created_at`, g.CourseID)
	if err != nil {
		return nil, err
	}
	m := &RAEMatrix{Students: []StudentPoint{}}
	if m.RAEs, err = scanRAEs(rows); err != nil {
		return nil, err
	}

	rows, err = s.db.Query(`SELECT u.id, u.name, AVG(v.grade), COUNT(v.grade)
		FROM group_students gs
		JOIN users u ON u.id = gs.student_id
		LEFT JOIN submissions s ON s.group_id = gs.group_id AND s.student_id = gs.student_id AND s.kind = 'official'
		LEFT JOIN reviews v ON v.submission_id = s.id AND v.status = 'returned'
		WHERE gs.group_id = ?
		GROUP BY u.id`, groupID)
	if err != nil {
		return nil, err
	}
	index := map[string]int{}
	for rows.Next() {
		var p StudentPoint
		var avg sql.NullFloat64
		if err := rows.Scan(&p.ID, &p.Name, &avg, &p.Graded); err != nil {
			rows.Close()
			return nil, err
		}
		if avg.Valid {
			v := math.Round(avg.Float64*100) / 100
			p.AverageGrade = &v
		}
		p.Levels = map[string]string{}
		index[p.ID] = len(m.Students)
		m.Students = append(m.Students, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = s.db.Query(`SELECT s.student_id, l.rae_id,
			AVG(CASE l.level WHEN 'bajo' THEN 1 WHEN 'medio' THEN 2 ELSE 3 END)
		FROM review_raes l
		JOIN submissions s ON s.id = l.submission_id
		JOIN reviews v ON v.submission_id = s.id AND v.status = 'returned'
		WHERE s.group_id = ? AND s.kind = 'official' AND l.level != ''
		GROUP BY s.student_id, l.rae_id`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var studentID, raeID string
		var score float64
		if err := rows.Scan(&studentID, &raeID, &score); err != nil {
			return nil, err
		}
		if i, ok := index[studentID]; ok {
			m.Students[i].Levels[raeID] = levelFromScore(score)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(m.Students, func(i, j int) bool {
		return strings.ToLower(m.Students[i].Name) < strings.ToLower(m.Students[j].Name)
	})
	return m, nil
}
