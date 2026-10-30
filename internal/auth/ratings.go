package auth

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

const (
	RatingReview = "revision"
	maxRatingComments = 8
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
	err := s.db.QueryRow(`SELECT s.group_id FROM submissions s
		JOIN reviews r ON r.submission_id = s.id AND r.status = 'returned'
		WHERE s.id = ? AND s.student_id = ?`, submissionID, studentID).Scan(&groupID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO feedback_ratings (at, kind, group_id, submission_id, student_id, understood, comment)
		VALUES (?, 'revision', ?, ?, ?, ?, ?)
		ON CONFLICT(submission_id, student_id) WHERE submission_id IS NOT NULL
		DO UPDATE SET at = excluded.at, understood = excluded.understood, comment = excluded.comment`,
		fmtTime(time.Now()), groupID, submissionID, studentID, understood, strings.TrimSpace(comment))
	return err
}

func (s *Store) RateInstant(studentID, kind, assignmentID, understood, comment string) error {
	if kind == RatingReview {
		return ErrInvalidRating
	}
	if err := validRating(kind, understood); err != nil {
		return err
	}
	_, err := s.db.Exec(`INSERT INTO feedback_ratings (at, kind, group_id, student_id, understood, comment)
		VALUES (?, ?, (SELECT a.group_id FROM assignments a JOIN group_students gs ON gs.group_id = a.group_id
			WHERE a.id = ? AND gs.student_id = ?), ?, ?, ?)`,
		fmtTime(time.Now()), kind, assignmentID, studentID, studentID, understood, strings.TrimSpace(comment))
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

type Comprehension struct {
	Reviews RatingCount `json:"revisiones"`
	Instant RatingCount `json:"inmediata"`
	Comments []RatingComment `json:"comentarios"`
}

func (s *Store) comprehension(groupID string) (Comprehension, error) {
	c := Comprehension{Comments: []RatingComment{}}
	rows, err := s.db.Query(`SELECT at, kind, understood, comment FROM feedback_ratings WHERE group_id = ? ORDER BY at DESC`, groupID)
	if err != nil {
		return c, err
	}
	defer rows.Close()
	for rows.Next() {
		var rc RatingComment
		var at string
		if err := rows.Scan(&at, &rc.Kind, &rc.Understood, &rc.Comment); err != nil {
			return c, err
		}
		count := &c.Instant
		if rc.Kind == RatingReview {
			count = &c.Reviews
		}
		switch rc.Understood {
		case "si":
			count.Yes++
		case "parcial":
			count.Partial++
		default:
			count.No++
		}
		if rc.Comment != "" && len(c.Comments) < maxRatingComments {
			rc.At = parseTime(at)
			c.Comments = append(c.Comments, rc)
		}
	}
	return c, rows.Err()
}
