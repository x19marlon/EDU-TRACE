package auth

import (
	"errors"
	"testing"
	"time"
)

func TestLateSubmissions(t *testing.T) {
	s := newTestStore(t)
	prof, err := s.CreateUser("Elena", "elena@u.edu", "password123", RoleTeacher)
	mustNoErr(t, err)
	ana, err := s.CreateUser("Ana", "ana@u.edu", "password123", RoleStudent)
	mustNoErr(t, err)
	c, err := s.CreateCourse(prof.ID, "Programación I")
	mustNoErr(t, err)
	g, err := s.CreateGroup(prof.ID, c.ID, "Grupo 1")
	mustNoErr(t, err)
	_, err = s.JoinGroup(ana.ID, g.JoinCode)
	mustNoErr(t, err)

	now := time.Now().UTC()
	past := now.Add(-26*time.Hour - 30*time.Minute)
	future := now.Add(48 * time.Hour)
	soon := now.Add(time.Hour)
	before := now.Add(-time.Hour)
	earlier := past.Add(-time.Hour)
	half := 0.5
	tooMuch := 6.0

	bad := []LatePolicy{
		{AllowLate: true, LateUntil: &earlier},
		{AllowLate: false, LateUntil: &future},
		{AllowLate: true, LatePenalty: &tooMuch},
		{AllowLate: false, LatePenalty: &half},
	}
	for i, p := range bad {
		if _, err := s.CreateAssignment(prof.ID, g.ID, "T", "S", "", &past, p, nil); !errors.Is(err, ErrInvalidDeadline) {
			t.Errorf("política inválida %d aceptada: %v", i, err)
		}
	}
	if _, err := s.CreateAssignment(prof.ID, g.ID, "T", "S", "", nil, LatePolicy{AllowLate: true, LatePenalty: &half}, nil); !errors.Is(err, ErrInvalidDeadline) {
		t.Errorf("penalización sin fecha límite aceptada: %v", err)
	}

	closed, err := s.CreateAssignment(prof.ID, g.ID, "Cerrado", "S", "", &past, LatePolicy{AllowLate: false}, nil)
	mustNoErr(t, err)
	open, err := s.CreateAssignment(prof.ID, g.ID, "Tardías", "S", "", &past, LatePolicy{AllowLate: true, LateUntil: &soon, LatePenalty: &half}, nil)
	mustNoErr(t, err)
	expired, err := s.CreateAssignment(prof.ID, g.ID, "Tardías vencidas", "S", "", &past, LatePolicy{AllowLate: true, LateUntil: &before}, nil)
	mustNoErr(t, err)
	onTime, err := s.CreateAssignment(prof.ID, g.ID, "A tiempo", "S", "", &future, LatePolicy{AllowLate: false}, nil)
	mustNoErr(t, err)
	if !open.AllowLate || open.LateUntil == nil || *open.LatePenalty != 0.5 || closed.AllowLate {
		t.Fatalf("política guardada: %+v / %+v", open, closed)
	}

	send := func(kind, assignmentID string) (*Submission, error) {
		sub := &Submission{SubmissionMeta: SubmissionMeta{Kind: kind, GroupID: g.ID, AssignmentID: assignmentID, StudentID: ana.ID, Success: true}, Code: "int main(){} // " + assignmentID + kind}
		return sub, s.AddSubmission(sub)
	}

	var de *DeadlineError
	if _, err := send(KindOfficial, closed.ID); !errors.As(err, &de) || de.Late || !de.ClosedAt.Equal(past.Truncate(time.Nanosecond)) {
		t.Fatalf("entrega a un taller cerrado: %v", err)
	}
	if _, err := send(KindOfficial, expired.ID); !errors.As(err, &de) || !de.Late {
		t.Fatalf("entrega tras el cierre de tardías: %v", err)
	}
	if _, err := send(KindAttempt, closed.ID); err != nil {
		t.Fatalf("los intentos de compilación nunca se bloquean: %v", err)
	}
	if _, err := send(KindOfficial, ""); err != nil {
		t.Fatalf("la práctica libre no tiene fecha: %v", err)
	}
	late, err := send(KindOfficial, open.ID)
	mustNoErr(t, err)
	timely, err := send(KindOfficial, onTime.ID)
	mustNoErr(t, err)

	subs, err := s.GroupSubmissions(prof.ID, g.ID, "", "")
	mustNoErr(t, err)
	byID := map[string]SubmissionMeta{}
	for _, m := range subs {
		byID[m.ID] = m
	}
	lm := byID[late.ID]
	want := int64(now.Sub(past) / time.Second)
	if lm.LateSeconds < want-5 || lm.LateSeconds > want+60 || lm.LatePenalty == nil || *lm.LatePenalty != 0.5 || lm.DueAt == nil {
		t.Fatalf("entrega tardía: %+v (esperaba ~%d s)", lm, want)
	}
	if tm := byID[timely.ID]; tm.LateSeconds != 0 || tm.LatePenalty != nil || tm.DueAt == nil {
		t.Fatalf("entrega a tiempo: %+v", tm)
	}

	mine := s.StudentSubmissions(ana.ID, 10)
	for _, m := range mine {
		if m.ID == late.ID && m.LateSeconds == 0 {
			t.Fatal("el estudiante no ve que su entrega fue tardía")
		}
	}

	_, err = s.UpdateAssignment(prof.ID, open.ID, "Tardías", "S", "", &future, LatePolicy{AllowLate: true}, nil, nil)
	mustNoErr(t, err)
	sub, err := s.SubmissionForTeacher(prof.ID, late.ID)
	mustNoErr(t, err)
	if sub.LateSeconds != 0 {
		t.Fatalf("al ampliar el plazo la entrega deja de ser tardía: %+v", sub.SubmissionMeta)
	}
}
