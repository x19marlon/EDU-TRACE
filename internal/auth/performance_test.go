package auth

import (
	"errors"
	"testing"
	"time"
)

func perfGrade(v float64) *float64 { return &v }

func (f *classFixture) perfAdd(t *testing.T, u *User, assignment, kind, code string) string {
	t.Helper()
	sub := &Submission{SubmissionMeta: SubmissionMeta{Kind: kind, GroupID: f.group.ID, AssignmentID: assignment, StudentID: u.ID, Success: true}, Code: code}
	mustNoErr(t, f.s.AddSubmission(sub))
	return sub.ID
}

func (f *classFixture) performance(t *testing.T, assignment string) map[string]StudentPerformance {
	t.Helper()
	list, err := f.s.GroupPerformance(f.prof.ID, f.group.ID, assignment)
	mustNoErr(t, err)
	out := map[string]StudentPerformance{}
	for _, p := range list {
		out[p.Name] = p
	}
	return out
}

func (f *classFixture) grade(t *testing.T, submission string, g float64, returned bool) {
	t.Helper()
	_, err := f.s.SaveTeacherReview(f.prof.ID, submission, "ok", perfGrade(g), returned, nil, nil)
	mustNoErr(t, err)
}

func TestGroupPerformanceCountsTeacherGrades(t *testing.T) {
	f := newClass(t)
	t2, err := f.s.CreateAssignment(f.prof.ID, f.group.ID, "Taller 3", "Enunciado", "", nil, AssignmentPolicy{AllowLate: true}, nil)
	mustNoErr(t, err)
	a1, a2 := f.assignment.ID, t2.ID

	// Ana: en el taller 1 vale la entrega calificada más reciente.
	for _, code := range []string{"a", "b", "c"} {
		f.perfAdd(t, f.ana, a1, KindAttempt, code)
	}
	old := f.perfAdd(t, f.ana, a1, KindOfficial, "c")
	time.Sleep(2 * time.Millisecond)
	recent := f.perfAdd(t, f.ana, a1, KindOfficial, "d")
	f.perfAdd(t, f.ana, a2, KindAttempt, "e")
	second := f.perfAdd(t, f.ana, a2, KindOfficial, "e")
	f.grade(t, old, 1.0, true)
	f.grade(t, recent, 4.3, true)
	f.grade(t, second, 4.6, false) // borrador del docente: cuenta

	got := f.performance(t, "")["Ana"]
	if got.Grade == nil || *got.Grade != 4.45 || got.GradedCount != 2 {
		t.Fatalf("nota de Ana = %v en %d talleres; quería 4.45 en 2", got.Grade, got.GradedCount)
	}
	if got.Attempts != 4 || got.Assignments != 2 || got.Submissions != 3 || got.Effort != 2 {
		t.Fatalf("Ana = %+v", got)
	}

	got = f.performance(t, a1)["Ana"]
	if got.Grade == nil || *got.Grade != 4.3 || got.Attempts != 3 || got.Effort != 3 {
		t.Fatalf("Ana en el taller 1 = %+v", got)
	}

	if _, err := f.s.GroupPerformance(f.otro.ID, f.group.ID, ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro docente: %v", err)
	}
}

func TestGroupPerformanceAIGradeNeedsTeacher(t *testing.T) {
	f := newClass(t)
	sub := f.perfAdd(t, f.ana, f.assignment.ID, KindOfficial, "x")
	_, err := f.s.SaveAutoAIDraft(f.prof.ID, sub, AIDraft{Feedback: "propuesta", Grade: perfGrade(3.5)})
	mustNoErr(t, err)

	got := f.performance(t, "")["Ana"]
	if got.Grade != nil || got.Category != "por_calificar" {
		t.Fatalf("una propuesta sin revisar no es una nota: %+v", got)
	}

	// El docente la confirma sin cambiarla: ahora sí cuenta.
	time.Sleep(2 * time.Millisecond)
	f.grade(t, sub, 3.5, false)
	if got := f.performance(t, "")["Ana"]; got.Grade == nil || *got.Grade != 3.5 {
		t.Fatalf("nota confirmada: %+v", got)
	}
}

// La cola de la IA puede escribir su propuesta después de que el docente guardó un
// borrador con otra nota: esa nota sigue siendo del docente.
func TestGroupPerformanceKeepsTeacherDraftAfterLateAIDraft(t *testing.T) {
	f := newClass(t)
	sub := f.perfAdd(t, f.ana, f.assignment.ID, KindOfficial, "x")
	f.grade(t, sub, 2.0, false)
	time.Sleep(2 * time.Millisecond)
	_, err := f.s.SaveAutoAIDraft(f.prof.ID, sub, AIDraft{Feedback: "propuesta", Grade: perfGrade(3.0)})
	mustNoErr(t, err)
	if got := f.performance(t, "")["Ana"]; got.Grade == nil || *got.Grade != 2.0 {
		t.Fatalf("nota del docente tras la propuesta: %+v", got)
	}
}

func TestPerformanceCategories(t *testing.T) {
	if len(PerformanceCategories) != 12 {
		t.Fatalf("hay %d categorías", len(PerformanceCategories))
	}
	seen := map[string]bool{}
	for _, c := range PerformanceCategories {
		if seen[c.Key] {
			t.Errorf("categoría repetida: %s", c.Key)
		}
		seen[c.Key] = true
		if c.Title == "" || c.Status == "" || c.Needs == "" || c.Attention == "" {
			t.Errorf("%s: categoría incompleta", c.Key)
		}
		if len(c.Title) > 32 {
			t.Errorf("%s: el título no cabe en su cuadrante", c.Key)
		}
	}

	tests := []struct {
		p    StudentPerformance
		want string
	}{
		{StudentPerformance{Grade: perfGrade(2.9), Effort: 2.9}, "poca_practica_nota_baja"},
		{StudentPerformance{Grade: perfGrade(2), Effort: 3}, "practica_sin_comprension"},
		{StudentPerformance{Grade: perfGrade(0), Effort: 6}, "esfuerzo_sin_resultado"},
		{StudentPerformance{Grade: perfGrade(3), Effort: 0}, "aprueba_poca_practica"},
		{StudentPerformance{Grade: perfGrade(3.9), Effort: 5.9}, "en_progreso"},
		{StudentPerformance{Grade: perfGrade(3), Effort: 20}, "persistente"},
		{StudentPerformance{Grade: perfGrade(4), Effort: 1}, "domina_el_tema"},
		{StudentPerformance{Grade: perfGrade(5), Effort: 3}, "buen_ritmo"},
		{StudentPerformance{Grade: perfGrade(4), Effort: 6}, "logra_con_esfuerzo"},
		{StudentPerformance{Submissions: 1}, "por_calificar"},
		{StudentPerformance{Attempts: 1}, "sin_entregar"},
		{StudentPerformance{}, "sin_actividad"},
	}
	for _, tt := range tests {
		tt.p.categorize()
		if tt.p.Category != tt.want {
			t.Errorf("%+v → %s; quería %s", tt.p, tt.p.Category, tt.want)
		}
	}
}
