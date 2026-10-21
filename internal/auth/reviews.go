package auth

import (
	"time"
)

const (
	ReviewDraft    = "draft"
	ReviewReturned = "returned"
)

type Review struct {
	SubmissionID string     `json:"submission_id"`
	TeacherID    string     `json:"teacher_id"`
	AIFeedback   string     `json:"ai_feedback,omitempty"`
	AIGrade      *float64   `json:"ai_grade,omitempty"`
	AIReason     string     `json:"ai_reason,omitempty"`
	AIConfidence string     `json:"ai_confidence,omitempty"`
	AIAt         *time.Time `json:"ai_at,omitempty"`
	Feedback     string     `json:"feedback"`
	Grade        *float64   `json:"grade,omitempty"`
	Status       string     `json:"status"`
	ReturnedAt   *time.Time `json:"returned_at,omitempty"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type ReviewState struct {
	Status string   `json:"status"`
	Grade  *float64 `json:"grade,omitempty"`
}

func (r *Review) GradeSource() string {
	switch {
	case r.Grade == nil:
		return ""
	case r.AIGrade != nil && *r.AIGrade == *r.Grade:
		return "ai_confirmed"
	default:
		return "profesor"
	}
}

func (s *Store) ownedOfficialLocked(teacherID, submissionID string) error {
	m, ok := s.data.Submissions[submissionID]
	if !ok || m.IsAttempt() {
		return ErrNotFound
	}
	_, _, err := s.ownedGroupLocked(teacherID, m.GroupID)
	return err
}

func (s *Store) reviewLocked(teacherID, submissionID string) *Review {
	r, ok := s.data.Reviews[submissionID]
	if !ok {
		r = &Review{SubmissionID: submissionID, TeacherID: teacherID, Status: ReviewDraft}
		s.data.Reviews[submissionID] = r
	}
	return r
}

func (s *Store) ReviewForTeacher(teacherID, submissionID string) (*Review, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ownedOfficialLocked(teacherID, submissionID); err != nil {
		return nil, err
	}
	r, ok := s.data.Reviews[submissionID]
	if !ok {
		return &Review{SubmissionID: submissionID, TeacherID: teacherID, Status: ReviewDraft}, nil
	}
	cp := *r
	return &cp, nil
}

func (s *Store) ReviewStates(submissionIDs []string) map[string]ReviewState {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]ReviewState{}
	for _, id := range submissionIDs {
		if r, ok := s.data.Reviews[id]; ok {
			out[id] = ReviewState{Status: r.Status, Grade: r.Grade}
		}
	}
	return out
}

func (s *Store) SaveAIDraft(teacherID, submissionID, feedback string, grade *float64, reason, confidence string) (*Review, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ownedOfficialLocked(teacherID, submissionID); err != nil {
		return nil, err
	}
	r := s.reviewLocked(teacherID, submissionID)
	now := time.Now().UTC()
	r.AIFeedback, r.AIGrade, r.AIReason, r.AIConfidence, r.AIAt = feedback, grade, reason, confidence, &now
	if r.Status != ReviewReturned && r.Feedback == "" {
		r.Feedback, r.Grade = feedback, grade
	}
	r.UpdatedAt = now
	if err := s.saveLocked(); err != nil {
		return nil, err
	}
	cp := *r
	return &cp, nil
}

func (s *Store) SaveTeacherReview(teacherID, submissionID, feedback string, grade *float64, returnIt bool) (*Review, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ownedOfficialLocked(teacherID, submissionID); err != nil {
		return nil, err
	}
	r := s.reviewLocked(teacherID, submissionID)
	now := time.Now().UTC()
	r.Feedback, r.Grade, r.UpdatedAt = feedback, grade, now
	if returnIt {
		r.Status, r.ReturnedAt = ReviewReturned, &now
	}
	if err := s.saveLocked(); err != nil {
		return nil, err
	}
	cp := *r
	return &cp, nil
}

type StudentReview struct {
	Feedback   string     `json:"feedback"`
	Grade      *float64   `json:"grade,omitempty"`
	ReturnedAt *time.Time `json:"returned_at"`
	AIAssisted bool       `json:"ai_assisted"`
}

func (s *Store) returnedReviewLocked(SubmissionID string) *StudentReview {
	r, ok := s.data.Reviews[SubmissionID]
	if !ok || r.Status != ReviewReturned {
		return nil
	}
	return &StudentReview{Feedback: r.Feedback, Grade: r.Grade, ReturnedAt: r.ReturnedAt, AIAssisted: r.AIAt != nil}
}
