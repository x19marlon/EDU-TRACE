package auth

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestCodeAnnotations(t *testing.T) {
	s := newTestStore(t)
	prof, err := s.CreateUser("Elena", "elena@u.edu", "password123", RoleTeacher)
	mustNoErr(t, err)
	otro, err := s.CreateUser("Otro", "otro@u.edu", "password123", RoleTeacher)
	mustNoErr(t, err)
	ana, err := s.CreateUser("Ana", "ana@u.edu", "password123", RoleStudent)
	mustNoErr(t, err)
	beto, err := s.CreateUser("Beto", "beto@u.edu", "password123", RoleStudent)
	mustNoErr(t, err)
	moni, err := s.CreateUser("Moni", "moni@u.edu", "password123", RoleMonitor)
	mustNoErr(t, err)
	mario, err := s.CreateUser("Mario", "mario@u.edu", "password123", RoleMonitor)
	mustNoErr(t, err)

	c, err := s.CreateCourse(prof.ID, "Programación I")
	mustNoErr(t, err)
	g, err := s.CreateGroup(prof.ID, c.ID, "Grupo 1")
	mustNoErr(t, err)
	for _, u := range []*User{ana, beto} {
		_, err := s.JoinGroup(u.ID, g.JoinCode)
		mustNoErr(t, err)
	}
	for _, m := range []string{"moni@u.edu", "mario@u.edu"} {
		_, err := s.AddMonitor(prof.ID, g.ID, m)
		mustNoErr(t, err)
	}
	code := "#include <iostream>\nint main() {\n    int x = 1\n    return 0;\n}"
	sub := &Submission{SubmissionMeta: SubmissionMeta{GroupID: g.ID, StudentID: ana.ID}, Code: code}
	mustNoErr(t, s.AddSubmission(sub))
	attempt := &Submission{SubmissionMeta: SubmissionMeta{Kind: KindAttempt, GroupID: g.ID, StudentID: ana.ID}, Code: code}
	mustNoErr(t, s.AddSubmission(attempt))

	pen := json.RawMessage(`{"strokes":[{"color":"#D64545","width":3,"points":[[4,2.5],[12,2.6]],"extra":true}]}`)
	cases := []struct {
		name string
		in AnnotationInput
	}{
		{"línea fuera del código", AnnotationInput{Kind: AnnotationHighlight, LineStart: 2, LineEnd: 9}},
		{"rango invertido", AnnotationInput{Kind: AnnotationStrike, LineStart: 3, LineEnd: 2}},
		{"nota vacía", AnnotationInput{Kind: AnnotationNote, Comment: "  "}},
		{"tipo desconocido", AnnotationInput{Kind: "circle"}},
		{"color no permitido", AnnotationInput{Kind: AnnotationDrawing, Data: json.RawMessage(`{"strokes":[{"color":"#000000","width":3,"points":[[1,1]]}]}`)}},
		{"dibujo fuera del código", AnnotationInput{Kind: AnnotationDrawing, Data: json.RawMessage(`{"strokes":[{"color":"#7154CC","width":3,"points":[[1,40]]}]}`)}},
		{"dibujo vacío", AnnotationInput{Kind: AnnotationDrawing, Data: json.RawMessage(`{"strokes":[]}`)}},
	}
	for _, tc := range cases {
		if _, err := s.AddAnnotation(prof, sub.ID, tc.in); !errors.Is(err, ErrInvalidAnnotation) {
			t.Errorf("%s: %v", tc.name, err)
		}
	}

	private, err := s.AddAnnotation(prof, sub.ID, AnnotationInput{Kind: AnnotationStrike, LineStart: 3, LineEnd: 3, Comment: "falta el punto y coma"})
	mustNoErr(t, err)
	drawingNote, err := s.AddAnnotation(prof, sub.ID, AnnotationInput{Kind: AnnotationDrawing, Data: pen})
	mustNoErr(t, err)
	if string(drawingNote.Data) == string(pen) {
		t.Fatal("el dibujo no se normalizó (quedó el campo extra)")
	}
	_, err = s.AddAnnotation(prof, attempt.ID, AnnotationInput{Kind: AnnotationNote, Comment: "nota privada sobre un intento"})
	mustNoErr(t, err)
	if _, err := s.AddAnnotation(otro, sub.ID, AnnotationInput{Kind: AnnotationNote, Comment: "x"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor anotó: %v", err)
	}
	if _, err := s.AddAnnotation(ana, sub.ID, AnnotationInput{Kind: AnnotationNote, Comment: "x"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("la estudiante anotó: %v", err)
	}
	if _, err := s.AddAnnotation(moni, attempt.ID, AnnotationInput{Kind: AnnotationNote, Comment: "x"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("el monitor anotó un intento: %v", err)
	}

	draft, err := s.AddAnnotation(moni, sub.ID, AnnotationInput{Kind: AnnotationHighlight, LineStart: 2, LineEnd: 4, Comment: "revisa este bloque"})
	mustNoErr(t, err)
	_, err = s.AddAnnotation(moni, sub.ID, AnnotationInput{Kind: AnnotationNote, Comment: "buen inicio"})
	mustNoErr(t, err)

	ids := func(list []Annotation) map[string]bool {
		out := map[string]bool{}
		for _, a := range list {
			out[a.ID] = true
		}
		return out
	}
	if l := s.AnnotationsFor(prof, sub.ID); len(l) != 4 || !ids(l)[draft.ID] {
		t.Fatalf("el profesor debe ver todo (lo suyo y los borradores del monitor): %d", len(l))
	}
	if l := s.AnnotationsFor(moni, sub.ID); len(l) != 2 || ids(l)[private.ID] {
		t.Fatalf("el monitor ve las notas privadas del profesor: %+v", l)
	}
	if l := s.AnnotationsFor(mario, sub.ID); len(l) != 0 {
		t.Fatalf("otro monitor ve borradores ajenos: %+v", l)
	}
	if l := s.AnnotationsFor(ana, sub.ID); len(l) != 0 {
		t.Fatalf("la estudiante ve borradores o notas del profesor: %+v", l)
	}

	if _, _, err := s.SendMonitorAnnotations(mario, attempt.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("enviar sobre un intento: %v", err)
	}
	n, _, err := s.SendMonitorAnnotations(moni, sub.ID)
	mustNoErr(t, err)
	if n != 2 {
		t.Fatalf("enviadas: %d", n)
	}
	if n, _, _ := s.SendMonitorAnnotations(moni, sub.ID); n != 0 {
		t.Fatalf("reenvió las mismas: %d", n)
	}
	l := s.AnnotationsFor(ana, sub.ID)
	if len(l) != 2 || l[0].SentAt == nil || l[0].AuthorRole != RoleMonitor || ids(l)[private.ID] {
		t.Fatalf("la estudiante ve lo enviado por el monitor: %+v", l)
	}
	if mine := s.StudentSubmissions(ana.ID, 10); len(mine) != 1 || mine[0].MonitorAnnotations != 2 {
		t.Fatalf("conteo de anotaciones enviadas en Mis entregas: %+v", mine)
	}
	if l := s.AnnotationsFor(beto, sub.ID); len(l) != 0 {
		t.Fatalf("otro estudiante ve anotaciones ajenas: %+v", l)
	}
	if l := s.AnnotationsFor(mario, sub.ID); len(l) != 2 {
		t.Fatalf("otro monitor ve lo ya enviado: %d", len(l))
	}

	if err := s.DeleteAnnotation(mario, draft.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("un monitor borró lo de otro: %v", err)
	}
	if err := s.DeleteAnnotation(ana, draft.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("la estudiante borró: %v", err)
	}
	if err := s.DeleteAnnotation(otro, draft.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor borró: %v", err)
	}
	if err := s.DeleteAnnotation(moni, private.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("el monitor borró la nota del profesor: %v", err)
	}
	mustNoErr(t, s.DeleteAnnotation(prof, draft.ID))
	if l := s.AnnotationsFor(ana, sub.ID); len(l) != 1 {
		t.Fatalf("tras borrar el profesor, la estudiante ve %d", len(l))
	}

	mustNoErr(t, s.RemoveStudent(prof.ID, g.ID, ana.ID))
	var left int
	mustNoErr(t, s.db.QueryRow(`SELECT COUNT(*) FROM code_annotations`).Scan(&left))
	if left != 0 {
		t.Fatalf("quedaron %d anotaciones de entregas borradas", left)
	}
}
