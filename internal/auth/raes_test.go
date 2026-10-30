package auth

import (
	"errors"
	"testing"
)

func TestRAELevelsAndMatrix(t *testing.T) {
	s := newTestStore(t)
	prof, err := s.CreateUser("Elena", "elena@u.edu", "password123", RoleTeacher)
	mustNoErr(t, err)
	otro, err := s.CreateUser("Otro", "otro@u.edu", "password123", RoleTeacher)
	mustNoErr(t, err)
	ana, err := s.CreateUser("Ana", "ana@u.edu", "password123", RoleStudent)
	mustNoErr(t, err)
	beto, err := s.CreateUser("Beto", "beto@u.edu", "password123", RoleStudent)
	mustNoErr(t, err)
	caro, err := s.CreateUser("Caro", "caro@u.edu", "password123", RoleStudent)
	mustNoErr(t, err)

	c, err := s.CreateCourse(prof.ID, "Programación I")
	mustNoErr(t, err)
	otherCourse, err := s.CreateCourse(otro.ID, "Otra")
	mustNoErr(t, err)
	g, err := s.CreateGroup(prof.ID, c.ID, "Grupo 1")
	mustNoErr(t, err)
	for _, u := range []*User{ana, beto, caro} {
		_, err := s.JoinGroup(u.ID, g.JoinCode)
		mustNoErr(t, err)
	}

	if _, err := s.CreateRAE(otro.ID, c.ID, "RAE 1", "Algoritmos"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor creó un RAE: %v", err)
	}
	r1, err := s.CreateRAE(prof.ID, c.ID, "RAE 1", "Diseña algoritmos")
	mustNoErr(t, err)
	r2, err := s.CreateRAE(prof.ID, c.ID, "RAE 2", "Implementa en C++")
	mustNoErr(t, err)
	foreign, err := s.CreateRAE(otro.ID, otherCourse.ID, "X", "De otra materia")
	mustNoErr(t, err)
	mustNoErr(t, s.UpdateRAE(prof.ID, r2.ID, "RAE 2", "Implementa programas en C++"))
	if err := s.UpdateRAE(otro.ID, r1.ID, "X", "Y"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor editó el RAE: %v", err)
	}
	list, err := s.CourseRAEs(prof.ID, c.ID)
	mustNoErr(t, err)
	if len(list) != 2 || list[0].ID != r1.ID || list[1].Description != "Implementa programas en C++" {
		t.Fatalf("RAE de la materia: %+v", list)
	}
	if courses := s.TeacherCourses(prof.ID); len(courses[0].RAEs) != 2 {
		t.Fatalf("la materia no trae sus RAE: %+v", courses[0].RAEs)
	}

	submit := func(u *User) string {
		sub := &Submission{SubmissionMeta: SubmissionMeta{GroupID: g.ID, StudentID: u.ID, Success: true}, Code: "int main(){} //" + u.ID}
		mustNoErr(t, s.AddSubmission(sub))
		return sub.ID
	}
	ana1, ana2, beto1, caro1 := submit(ana), submit(ana), submit(beto), submit(caro)

	g45, g32, g50, g20 := 4.5, 3.2, 5.0, 2.0
	rev, err := s.SaveAIDraft(prof.ID, ana1, "propuesta", &g45, "razón", "alta", map[string]string{r1.ID: "medio", r2.ID: "alto"})
	mustNoErr(t, err)
	if len(rev.RAEs) != 2 || rev.RAEs[0].AILevel != "medio" || rev.RAEs[0].Level != "medio" {
		t.Fatalf("la IA no precargó los niveles: %+v", rev.RAEs)
	}
	rev, err = s.SaveTeacherReview(prof.ID, ana1, "bien", &g45, false, map[string]string{r1.ID: "alto", foreign.ID: "bajo"})
	mustNoErr(t, err)
	if rev.RAEs[0].Level != "alto" || rev.RAEs[0].AILevel != "medio" || len(rev.RAEs) != 2 {
		t.Fatalf("niveles del profesor: %+v", rev.RAEs)
	}
	rev, err = s.SaveAIDraft(prof.ID, ana1, "otra", &g45, "razón", "alta", map[string]string{r1.ID: "bajo"})
	mustNoErr(t, err)
	if rev.RAEs[0].Level != "alto" || rev.RAEs[0].AILevel != "bajo" {
		t.Fatalf("la IA pisó el nivel del profesor: %+v", rev.RAEs)
	}
	rev, err = s.SaveAIDraft(prof.ID, ana1, "otra", &g45, "razón", "alta", map[string]string{r1.ID: "", r2.ID: ""})
	mustNoErr(t, err)
	if rev.RAEs[0].AILevel != "" || rev.RAEs[0].Level != "alto" || rev.RAEs[1].AILevel != "" || rev.RAEs[1].Level != "alto" {
		t.Fatalf("un RAE que ya no aplica debe quedar sin propuesta de la IA, sin tocar el nivel del profesor: %+v", rev.RAEs)
	}
	if _, err := s.SaveTeacherReview(prof.ID, ana1, "bien", &g45, false, map[string]string{r1.ID: "excelente"}); err == nil {
		t.Fatal("aceptó un nivel inválido")
	}

	if mine := s.StudentSubmissions(ana.ID, 10); mine[1].Review != nil {
		t.Fatal("el estudiante ve una revisión sin devolver")
	}
	_, err = s.SaveTeacherReview(prof.ID, ana1, "bien", &g45, true, nil)
	mustNoErr(t, err)
	_, err = s.SaveTeacherReview(prof.ID, ana2, "regular", &g32, true, map[string]string{r1.ID: "medio", r2.ID: "bajo"})
	mustNoErr(t, err)
	_, err = s.SaveTeacherReview(prof.ID, beto1, "excelente", &g50, true, map[string]string{r1.ID: "alto"})
	mustNoErr(t, err)
	_, err = s.SaveTeacherReview(prof.ID, caro1, "borrador", &g20, false, map[string]string{r1.ID: "bajo"})
	mustNoErr(t, err)

	mine := s.StudentSubmissions(ana.ID, 10)
	var first *StudentSubmission
	for i := range mine {
		if mine[i].ID == ana1 {
			first = &mine[i]
		}
	}
	if first == nil || first.Review == nil || len(first.Review.RAEs) != 2 || first.Review.RAEs[0].Code != "RAE 1" || first.Review.RAEs[0].Level != "alto" {
		t.Fatalf("el estudiante no ve sus niveles: %+v", first)
	}

	if _, err := s.GroupRAEMatrix(otro.ID, g.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor ve la matriz: %v", err)
	}
	m, err := s.GroupRAEMatrix(prof.ID, g.ID)
	mustNoErr(t, err)
	if len(m.RAEs) != 2 || len(m.Students) != 3 || m.Students[0].Name != "Ana" {
		t.Fatalf("matriz: %+v", m)
	}
	pa, pb, pc := m.Students[0], m.Students[1], m.Students[2]
	if pa.AverageGrade == nil || *pa.AverageGrade != 3.85 || pa.Graded != 2 {
		t.Fatalf("promedio de Ana: %+v", pa)
	}
	if pa.Levels[r1.ID] != "alto" || pa.Levels[r2.ID] != "medio" {
		t.Fatalf("niveles de Ana (alto+medio redondea a alto, alto+bajo a medio): %+v", pa.Levels)
	}
	if pb.AverageGrade == nil || *pb.AverageGrade != 5 || pb.Levels[r1.ID] != "alto" || pb.Levels[r2.ID] != "" {
		t.Fatalf("Beto: %+v", pb)
	}
	if pc.AverageGrade != nil || pc.Graded != 0 || len(pc.Levels) != 0 {
		t.Fatalf("Caro solo tiene un borrador y no debe contar: %+v", pc)
	}

	mustNoErr(t, s.DeleteRAE(prof.ID, r2.ID))
	var n int
	mustNoErr(t, s.db.QueryRow(`SELECT COUNT(*) FROM review_raes WHERE rae_id = ?`, r2.ID).Scan(&n))
	if n != 0 {
		t.Fatalf("quedaron %d niveles del RAE borrado", n)
	}
	if err := s.DeleteRAE(otro.ID, r1.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor borró el RAE: %v", err)
	}
}
