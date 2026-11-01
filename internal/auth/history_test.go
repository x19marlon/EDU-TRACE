package auth

import (
	"errors"
	"testing"
)

func TestStudentHistory(t *testing.T) {
	s := newTestStore(t)
	prof, err := s.CreateUser("Elena", "elena@u.edu", "password123", RoleTeacher)
	mustNoErr(t, err)
	ana, err := s.CreateUser("Ana", "ana@u.edu", "password123", RoleStudent)
	mustNoErr(t, err)
	beto, err := s.CreateUser("Beto", "beto@u.edu", "password123", RoleStudent)
	mustNoErr(t, err)
	c, err := s.CreateCourse(prof.ID, "Programación I")
	mustNoErr(t, err)
	g1, err := s.CreateGroup(prof.ID, c.ID, "Grupo 1")
	mustNoErr(t, err)
	g2, err := s.CreateGroup(prof.ID, c.ID, "Grupo 2")
	mustNoErr(t, err)
	for _, g := range []*Group{g1, g2} {
		_, err := s.JoinGroup(ana.ID, g.JoinCode)
		mustNoErr(t, err)
	}
	_, err = s.JoinGroup(beto.ID, g1.JoinCode)
	mustNoErr(t, err)
	a, err := s.CreateAssignment(prof.ID, g1.ID, "Taller 1", "Suma", "", nil, AssignmentPolicy{AllowLate: true}, nil)
	mustNoErr(t, err)

	add := func(u *User, g *Group, kind, assignment, code string) string {
		sub := &Submission{SubmissionMeta: SubmissionMeta{Kind: kind, GroupID: g.ID, AssignmentID: assignment, StudentID: u.ID, Success: true}, Code: code, ProgramOutput: "ok"}
		mustNoErr(t, s.AddSubmission(sub))
		return sub.ID
	}
	att1 := add(ana, g1, KindAttempt, a.ID, "int main(){")
	att2 := add(ana, g1, KindAttempt, a.ID, "int main(){}")
	off1 := add(ana, g1, KindOfficial, a.ID, "int main(){}")
	off2 := add(ana, g2, KindOfficial, "", "int main(){ return 0; }")
	betoSub := add(beto, g1, KindOfficial, a.ID, "int main(){}")

	g45 := 4.5
	_, err = s.SaveTeacherReview(prof.ID, off1, "bien", &g45, true, nil, nil)
	mustNoErr(t, err)
	_, err = s.SaveTeacherReview(prof.ID, off2, "borrador", &g45, false, nil, nil)
	mustNoErr(t, err)

	h, err := s.History(ana.ID)
	mustNoErr(t, err)
	if len(h.Submissions) != 2 || len(h.Attempts) != 2 {
		t.Fatalf("historial: %d entregas, %d intentos", len(h.Submissions), len(h.Attempts))
	}
	if h.Attempts[0].ID != att2 || h.Attempts[1].ID != att1 || h.Attempts[0].Review != nil {
		t.Fatalf("intentos: %+v", h.Attempts)
	}
	byID := map[string]StudentSubmission{}
	for _, x := range h.Submissions {
		byID[x.ID] = x
	}
	if x := byID[off1]; x.AssignmentTitle != "Taller 1" || x.GroupName != "Grupo 1" || x.Review == nil || *x.Review.Grade != 4.5 {
		t.Fatalf("entrega con taller: %+v", x)
	}
	if x := byID[off2]; x.AssignmentTitle != "" || x.GroupName != "Grupo 2" || x.Review != nil {
		t.Fatalf("práctica libre en borrador: %+v", x)
	}
	if _, ok := byID[betoSub]; ok {
		t.Fatal("aparece la entrega de otro estudiante")
	}

	sub, err := s.OwnSubmission(ana.ID, att1)
	mustNoErr(t, err)
	if sub.Code != "int main(){" || sub.ProgramOutput != "ok" {
		t.Fatalf("detalle propio: %+v", sub)
	}
	if _, err := s.OwnSubmission(ana.ID, betoSub); !errors.Is(err, ErrNotFound) {
		t.Fatalf("vio la entrega de otro estudiante: %v", err)
	}
	if _, err := s.OwnSubmission(beto.ID, off1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro estudiante vio la entrega de Ana: %v", err)
	}

	if got := s.StudentSubmissions(ana.ID, 20); len(got) != 2 || got[0].AssignmentTitle+got[1].AssignmentTitle != "Taller 1" {
		t.Fatalf("Mis entregas trae el título del taller: %+v", got)
	}
}
