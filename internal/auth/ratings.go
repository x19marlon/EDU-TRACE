package auth

import (
	"database/sql"
	"errors"
	"sort"
	"strings"
	"time"
)

const (
	RatingReview = "revision"
	maxRatingComments = 8
	ComprehensionMinYes = 0.8
	ComprehensionMaxNo = 0.1
	ComprehensionMinResponses = 20
	VerdictMeets = "cumple"
	VerdictFails = "no_cumple"
	VerdictTooFew = "pocas_respuestas"
)

var ErrInvalidRating = errors.New("la respuesta debe ser «si», «parcial» o «no»")

func validRating(kind, understood string) error {
	switch kind {
	case AIKindFormal, AIKindInformal, RatingReview:
	default:
		return ErrInvalidRating
	}
	switch understood {
	case "si", "parcial", "no":
		return nil
	}
	return ErrInvalidRating
}

func (s *Store) RateReview(studentID, submissionID, understood, comment string) error {
	if err := validRating(RatingReview, understood); err != nil {
		return err
	}
	var groupID string
	var assignmentID sql.NullString
	err := s.db.QueryRow(`SELECT s.group_id, s.assignment_id FROM submissions s
		JOIN reviews r ON r.submission_id = s.id AND r.status = 'returned'
		WHERE s.id = ? AND s.student_id = ?`, submissionID, studentID).Scan(&groupID, &assignmentID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO feedback_ratings (at, kind, group_id, submission_id, student_id, understood, comment, assignment_id)
		VALUES (?, 'revision', ?, ?, ?, ?, ?, ?)
		ON CONFLICT(submission_id, student_id) WHERE submission_id IS NOT NULL
		DO UPDATE SET at = excluded.at, understood = excluded.understood, comment = excluded.comment`,
		fmtTime(time.Now()), groupID, submissionID, studentID, understood, strings.TrimSpace(comment), assignmentID)
	return err
}

func (s *Store) RateInstant(studentID, kind, assignmentID, understood, comment string) error {
	if kind == RatingReview {
		return ErrInvalidRating
	}
	if err := validRating(kind, understood); err != nil {
		return err
	}
	_, err := s.db.Exec(`INSERT INTO feedback_ratings (at, kind, group_id, assignment_id, student_id, understood, comment)
		SELECT ?, ?, a.group_id, a.id, ?, ?, ? FROM (SELECT 1) LEFT JOIN assignments a ON a.id = ?
			AND EXISTS (SELECT 1 FROM group_students gs WHERE gs.group_id = a.group_id AND gs.student_id = ?)`,
		fmtTime(time.Now()), kind, studentID, understood, strings.TrimSpace(comment), assignmentID, studentID)
	return err
}

type RatingCount struct {
	Yes int `json:"si"`
	Partial int `json:"parcial"`
	No int `json:"no"`
}

type RatingComment struct {
	At time.Time `json:"at"`
	Kind string `json:"kind"`
	Understood string `json:"understood"`
	Comment string `json:"comment"`
}

func (c *RatingCount) add(understood string) {
	switch understood {
	case "si":
		c.Yes++
	case "parcial":
		c.Partial++
	default:
		c.No++
	}
}

func (c RatingCount) total() int { return c.Yes + c.Partial + c.No }

type ComprehensionGoal struct {
	MinYes float64 `json:"min_si"`
	MaxNo float64 `json:"max_no"`
	MinResponses int `json:"min_respuestas"`
}

type AssignmentComprehension struct {
	AssignmentID string `json:"assignment_id"`
	Title string `json:"title"`
	RatingCount
	Total int `json:"total"`
	Enough bool `json:"suficientes"`
	Meets bool `json:"cumple"`
}

type Comprehension struct {
	Reviews RatingCount `json:"revisiones"`
	Instant RatingCount `json:"inmediata"`
	Comments []RatingComment `json:"comentarios"`
	Goal ComprehensionGoal `json:"meta"`
	ByAssignment []AssignmentComprehension `json:"por_taller"`
	Verdict string `json:"veredicto"`
}

func (s *Store) comprehension(groupID string) (Comprehension, error) {
	c := Comprehension{Comments: []RatingComment{}, ByAssignment: []AssignmentComprehension{}, Verdict: VerdictTooFew,
		Goal: ComprehensionGoal{MinYes: ComprehensionMinYes, MaxNo: ComprehensionMaxNo, MinResponses: ComprehensionMinResponses}}
	rows, err := s.db.Query(`SELECT fr.at, fr.kind, fr.understood, fr.comment, COALESCE(fr.assignment_id, ''), COALESCE(a.title, '')
		FROM feedback_ratings fr LEFT JOIN assignments a ON a.id = fr.assignment_id
		WHERE fr.group_id = ? ORDER BY fr.at DESC`, groupID)
	if err != nil {
		return c, err
	}
	defer rows.Close()
	index := map[string]int{}
	for rows.Next() {
		var rc RatingComment
		var at, assignmentID, title string
		if err := rows.Scan(&at, &rc.Kind, &rc.Understood, &rc.Comment, &assignmentID, &title); err != nil {
			return c, err
		}
		if rc.Kind == RatingReview {
			c.Reviews.add(rc.Understood)
		} else {
			c.Instant.add(rc.Understood)
		}
		i, ok := index[assignmentID]
		if !ok {
			if title == "" {
				title = "Práctica libre"
			}
			i = len(c.ByAssignment)
			index[assignmentID] = i
			c.ByAssignment = append(c.ByAssignment, AssignmentComprehension{AssignmentID: assignmentID, Title: title})
		}
		c.ByAssignment[i].add(rc.Understood)
		if rc.Comment != "" && len(c.Comments) < maxRatingComments {
			rc.At = parseTime(at)
			c.Comments = append(c.Comments, rc)
		}
	}
	if err := rows.Err(); err != nil {
		return c, err
	}
	enough, meeting := 0, 0
	for i := range c.ByAssignment {
		a := &c.ByAssignment[i]
		a.Total = a.total()
		a.Enough = a.Total >= ComprehensionMinResponses
		if a.Total > 0 {
			yes, no := float64(a.Yes)/float64(a.Total), float64(a.No)/float64(a.Total)
			a.Meets = yes >= ComprehensionMinYes-1e-9 && no <= ComprehensionMaxNo+1e-9
		}
		if a.Enough {
			enough++
			if a.Meets {
				meeting++
			}
		}
	}
	switch {
	case enough == 0:
		c.Verdict = VerdictTooFew
	case meeting == enough:
		c.Verdict = VerdictMeets
	default:
		c.Verdict = VerdictFails
	}
	sort.SliceStable(c.ByAssignment, func(i, j int) bool { return c.ByAssignment[i].Total > c.ByAssignment[j].Total })
	return c, nil
}
