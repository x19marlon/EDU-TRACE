package auth

import (
	"database/sql"
	"errors"
)

const (
	historySubmissionsLimit = 1000
	historyAttemptsLimit = 500
)

type StudentHistory struct {
	Submissions []StudentSubmission `json:"submissions"`
	Attempts []StudentSubmission `json:"attempts"`
}

func (s *Store) History(studentID string) (*StudentHistory, error) {
	subs, err := s.studentSubmissions(studentID, KindOfficial, historySubmissionsLimit)
	if err != nil {
		return nil, err
	}
	attempts, err := s.studentSubmissions(studentID, KindAttempt, historyAttemptsLimit)
	if err != nil {
		return nil, err
	}
	return &StudentHistory{Submissions: subs, Attempts: attempts}, nil
}

func (s *Store) OwnSubmission(studentID, submissionID string) (*Submission, error) {
	sub, err := scanSubmission(s.db.QueryRow(`SELECT `+metaCols+`, `+contentCols+`
		FROM submissions s WHERE s.id = ? AND s.student_id = ?`, submissionID, studentID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return sub, err
}
