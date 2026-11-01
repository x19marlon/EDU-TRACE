package auth

import (
	"errors"
	"strings"
	"testing"

	"github.com/marlon/edu-trace/internal/model"
)

func (f *classFixture) attempt(t *testing.T, u *User, success bool, summary, runErr string, exit int) {
	t.Helper()
	sub := &Submission{SubmissionMeta: SubmissionMeta{Kind: KindAttempt, GroupID: f.group.ID, AssignmentID: f.assignment.ID, StudentID: u.ID,
		Success: success, ErrorSummary: summary, RunError: runErr, ExitCode: exit}, Code: summary + runErr + string(rune('a'+exit%26)) + u.ID}
	mustNoErr(t, f.s.AddSubmission(sub))
}

func TestFrequentErrors(t *testing.T) {
	f := newClass(t)
	s := f.s
	f.attempt(t, f.ana, false, "‘suma’ was not declared in this scope", "", 1)
	f.attempt(t, f.beto, false, "‘total’ was not declared in this scope", "", 1)
	f.attempt(t, f.ana, false, "expected ‘;’ before ‘cout’", "", 1)
	f.attempt(t, f.ana, true, "", "se agotó el tiempo de ejecución (¿un ciclo infinito o esperando entrada?)", -1)
	f.attempt(t, f.beto, true, "", "", 136)
	f.attempt(t, f.beto, true, "", "", 0)

	tests, err := s.SetAssignmentTests(f.prof.ID, f.assignment.ID, []model.TestCase{{Name: "Cero", Expected: "0", Match: model.MatchContains}, {Name: "Uno", Expected: "1", Match: model.MatchContains}})
	mustNoErr(t, err)
	f.submit(t, f.ana, KindOfficial, []model.TestResult{{TestID: tests[0].ID, Passed: false}, {TestID: tests[1].ID, Passed: true}})
	f.submit(t, f.beto, KindOfficial, []model.TestResult{{TestID: tests[0].ID, Passed: false}, {TestID: tests[1].ID, Passed: true}})
	f.submit(t, f.beto, KindOfficial, []model.TestResult{{TestID: tests[0].ID, Passed: true}, {TestID: tests[1].ID, Passed: true}})

	if _, err := s.FrequentErrors(f.otro.ID, f.group.ID, false); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor vio los errores: %v", err)
	}
	r, err := s.FrequentErrors(f.prof.ID, f.group.ID, false)
	mustNoErr(t, err)
	if r.Analyzed != 9 || r.StudentsWithErrors != 2 || len(r.Compile) != 2 {
		t.Fatalf("reporte: %+v", r)
	}
	top := r.Compile[0]
	if top.Message != "'…' was not declared in this scope" || top.Students != 2 || top.Count != 2 || !strings.Contains(top.Explanation, "no está declarada") || top.Assignments[0] != "Taller 2" {
		t.Fatalf("error más frecuente: %+v", top)
	}
	if r.Compile[1].Message != "expected ';' before 'cout'" || !strings.Contains(r.Compile[1].Explanation, "punto y coma") {
		t.Fatalf("segundo error: %+v", r.Compile[1])
	}
	if len(r.Runtime) != 2 || r.Runtime[0].Students != 1 {
		t.Fatalf("problemas de ejecución: %+v", r.Runtime)
	}
	if len(r.Tests) != 1 || r.Tests[0].Test != "Cero" || r.Tests[0].Students != 2 || r.Tests[0].NotPassing != 1 || r.Tests[0].Failures != 2 {
		t.Fatalf("casos difíciles: %+v", r.Tests)
	}

	other, err := s.CreateGroup(f.prof.ID, f.course.ID, "Grupo 2")
	mustNoErr(t, err)
	carla, err := s.CreateUser("Carla", "carla@u.edu", "password123", RoleStudent)
	mustNoErr(t, err)
	_, err = s.JoinGroup(carla.ID, other.JoinCode)
	mustNoErr(t, err)
	mustNoErr(t, s.AddSubmission(&Submission{SubmissionMeta: SubmissionMeta{Kind: KindAttempt, GroupID: other.ID, StudentID: carla.ID,
		ErrorSummary: "‘x’ was not declared in this scope"}, Code: "x"}))
	course, err := s.FrequentErrors(f.prof.ID, f.group.ID, true)
	mustNoErr(t, err)
	if course.Scope != "materia" || course.Groups != 2 || course.Compile[0].Students != 3 {
		t.Fatalf("toda la materia: %+v", course.Compile)
	}
}

func TestQuestionsAndRatings(t *testing.T) {
	f := newClass(t)
	s := f.s
	sub := f.submit(t, f.ana, KindOfficial, nil)
	if _, err := s.AskQuestion(f.beto.ID, sub.ID, "¿Por qué?"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro estudiante preguntó sobre la entrega ajena: %v", err)
	}
	q, err := s.AskQuestion(f.ana.ID, sub.ID, "  ¿Qué significa división entera?  ")
	mustNoErr(t, err)
	if q.Body != "¿Qué significa división entera?" {
		t.Fatalf("pregunta: %+v", q)
	}
	if n := s.OpenQuestionCounts([]string{sub.ID}); n[sub.ID] != 1 {
		t.Fatalf("preguntas abiertas: %v", n)
	}
	open, err := s.TeacherOpenQuestions(f.prof.ID)
	mustNoErr(t, err)
	if len(open) != 1 || open[0].StudentName != "Ana" || open[0].GroupID != f.group.ID {
		t.Fatalf("pendientes del profesor: %+v", open)
	}
	if _, err := s.AnswerQuestion(f.otro, q.ID, "Respuesta"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor respondió: %v", err)
	}
	mon, err := s.CreateUser("Mona", "mona@u.edu", "password123", RoleMonitor)
	mustNoErr(t, err)
	if _, err := s.AnswerQuestion(mon, q.ID, "Respuesta"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("un monitor sin la clase respondió: %v", err)
	}
	_, err = s.AddMonitor(f.prof.ID, f.group.ID, "mona@u.edu")
	mustNoErr(t, err)
	answered, err := s.AnswerQuestion(mon, q.ID, "Cuando divides dos int, C++ descarta los decimales.")
	mustNoErr(t, err)
	if answered.AnsweredBy != "Mona" || answered.AnsweredRole != "monitor" || answered.AnsweredAt == nil {
		t.Fatalf("respuesta: %+v", answered)
	}
	if n := s.OpenQuestionCounts([]string{sub.ID}); n[sub.ID] != 0 {
		t.Fatalf("la pregunta respondida sigue abierta: %v", n)
	}
	mine := s.StudentSubmissions(f.ana.ID, 5)
	if len(mine[0].Questions) != 1 || mine[0].Questions[0].Answer == "" {
		t.Fatalf("el estudiante ve la respuesta: %+v", mine[0].Questions)
	}
	log, err := s.SubmissionAudit(f.prof.ID, sub.ID)
	mustNoErr(t, err)
	if log[0].Action != AuditQuestionAnswered || log[0].ActorName != "Mona" {
		t.Fatalf("la respuesta queda en la bitácora: %+v", log)
	}

	if err := s.RateReview(f.ana.ID, sub.ID, "si", ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("valoró una revisión sin devolver: %v", err)
	}
	g := 4.0
	_, err = s.SaveTeacherReview(f.prof.ID, sub.ID, "Bien", &g, true, nil, nil)
	mustNoErr(t, err)
	if err := s.RateReview(f.ana.ID, sub.ID, "quizas", ""); !errors.Is(err, ErrInvalidRating) {
		t.Fatalf("aceptó una respuesta inválida: %v", err)
	}
	mustNoErr(t, s.RateReview(f.ana.ID, sub.ID, "no", "No entiendo la línea 7"))
	mustNoErr(t, s.RateReview(f.ana.ID, sub.ID, "parcial", "Ya casi"))
	mustNoErr(t, s.RateInstant(f.ana.ID, AIKindFormal, f.assignment.ID, "si", ""))
	mustNoErr(t, s.RateInstant(f.beto.ID, AIKindInformal, "", "no", "Muy largo"))
	if err := s.RateInstant(f.ana.ID, RatingReview, "", "si", ""); !errors.Is(err, ErrInvalidRating) {
		t.Fatalf("una valoración inmediata no puede ser de revisión: %v", err)
	}
	if mine := s.StudentSubmissions(f.ana.ID, 5); mine[0].Review.Understood != "parcial" {
		t.Fatalf("la valoración se actualiza, no se duplica: %+v", mine[0].Review)
	}
	data, err := s.ExportData(f.ana)
	mustNoErr(t, err)
	if len(data.Ratings) != 2 || len(data.Submissions[0].Questions) != 1 {
		t.Fatalf("la exportación incluye preguntas y valoraciones: %+v %+v", data.Ratings, data.Submissions[0].Questions)
	}
	mustNoErr(t, s.DeleteAccount(mon.ID, "password123"))
	if q := s.QuestionsFor(sub.ID); q[0].AnsweredBy != "Cuenta eliminada" {
		t.Fatalf("el nombre de una cuenta eliminada no debe quedar: %+v", q)
	}
	audit, err := s.SubmissionAudit(f.prof.ID, sub.ID)
	mustNoErr(t, err)
	for _, e := range audit {
		if e.Action == AuditQuestionAnswered && e.ActorName != "Cuenta eliminada" {
			t.Fatalf("la bitácora conserva el nombre de la cuenta eliminada: %+v", e)
		}
	}
	m, err := s.GroupMetrics(f.prof.ID, f.group.ID)
	mustNoErr(t, err)
	c := m.Comprehension
	if c.Reviews.Partial != 1 || c.Reviews.No != 0 || c.Instant.Yes != 1 || len(c.Comments) != 1 || c.Comments[0].Comment != "Ya casi" {
		t.Fatalf("comprensión (RNF-09): %+v", c)
	}
}

func TestExports(t *testing.T) {
	f := newClass(t)
	s := f.s
	second, err := s.CreateAssignment(f.prof.ID, f.group.ID, "Taller 3", "Mayor de tres", "", nil, AssignmentPolicy{AllowLate: true}, nil)
	mustNoErr(t, err)
	g1, g2, g3 := 3.0, 4.5, 5.0
	old := f.submit(t, f.ana, KindOfficial, nil)
	_, err = s.SaveTeacherReview(f.prof.ID, old.ID, "Primera", &g1, true, nil, nil)
	mustNoErr(t, err)
	last := f.submit(t, f.ana, KindOfficial, nil)
	_, err = s.SaveTeacherReview(f.prof.ID, last.ID, "Segunda", &g2, true, nil, nil)
	mustNoErr(t, err)
	other := &Submission{SubmissionMeta: SubmissionMeta{Kind: KindOfficial, GroupID: f.group.ID, AssignmentID: second.ID, StudentID: f.ana.ID, Success: true}, Code: "y"}
	mustNoErr(t, s.AddSubmission(other))
	_, err = s.SaveTeacherReview(f.prof.ID, other.ID, "=HYPERLINK(\"x\")", &g3, true, nil, nil)
	mustNoErr(t, err)
	draft := f.submit(t, f.beto, KindOfficial, nil)
	_, err = s.SaveTeacherReview(f.prof.ID, draft.ID, "Borrador", &g3, false, nil, nil)
	mustNoErr(t, err)

	if _, err := s.GradeSheet(f.otro.ID, f.group.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor exportó: %v", err)
	}
	sheet, err := s.GradeSheet(f.prof.ID, f.group.ID)
	mustNoErr(t, err)
	if len(sheet.Assignments) != 2 || sheet.Assignments[0].Title != "Taller 2" || len(sheet.Rows) != 2 {
		t.Fatalf("planilla: %+v", sheet)
	}
	ana, beto := sheet.Rows[0], sheet.Rows[1]
	if *ana.Grades[f.assignment.ID] != 4.5 || *ana.Grades[second.ID] != 5 || *ana.Average != 4.75 || ana.Email != "ana@u.edu" {
		t.Fatalf("Ana usa la última entrega devuelta de cada taller: %+v", ana)
	}
	if len(beto.Grades) != 0 || beto.Average != nil {
		t.Fatalf("un borrador no se exporta: %+v", beto)
	}
	reviews, err := s.ReviewsExport(f.prof.ID, f.group.ID)
	mustNoErr(t, err)
	if len(reviews) != 3 || reviews[0].StudentName != "Ana" || reviews[0].Origin != OriginTeacher || reviews[2].Feedback != "=HYPERLINK(\"x\")" {
		t.Fatalf("revisiones: %+v", reviews)
	}
}

func TestReviewTimeWithoutSaving(t *testing.T) {
	f := newClass(t)
	s := f.s
	sub := f.submit(t, f.ana, KindOfficial, nil)
	if _, err := s.AddReviewTime(f.otro.ID, sub.ID, 60000); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor sumó tiempo: %v", err)
	}
	_, err := s.AddReviewTime(f.prof.ID, sub.ID, 90000)
	mustNoErr(t, err)
	_, err = s.AddReviewTime(f.prof.ID, sub.ID, 30000)
	mustNoErr(t, err)
	rev, err := s.ReviewForTeacher(f.prof.ID, sub.ID)
	mustNoErr(t, err)
	if rev.ReviewMs != 120000 || rev.Status != ReviewDraft || rev.Feedback != "" {
		t.Fatalf("el tiempo se guarda aunque no haya revisión: %+v", rev)
	}
	if needed, _ := s.NeedsAIDraft(sub.ID); !needed {
		t.Fatal("abrir la entrega no debe impedir la propuesta de la IA")
	}
}

func TestReviewPlan(t *testing.T) {
	f := newClass(t)
	s := f.s
	tests, err := s.SetAssignmentTests(f.prof.ID, f.assignment.ID, []model.TestCase{{Name: "A", Expected: "1", Match: model.MatchContains}})
	mustNoErr(t, err)
	carla, err := s.CreateUser("Carla", "carla@u.edu", "password123", RoleStudent)
	mustNoErr(t, err)
	_, err = s.JoinGroup(carla.ID, f.group.JoinCode)
	mustNoErr(t, err)

	f.submit(t, f.ana, KindOfficial, []model.TestResult{{TestID: tests[0].ID, Passed: false}})
	ana := f.submit(t, f.ana, KindOfficial, []model.TestResult{{TestID: tests[0].ID, Passed: true}})
	beto := f.submit(t, f.beto, KindOfficial, []model.TestResult{{TestID: tests[0].ID, Passed: false}})
	broken := &Submission{SubmissionMeta: SubmissionMeta{Kind: KindOfficial, GroupID: f.group.ID, AssignmentID: f.assignment.ID, StudentID: carla.ID}, Code: "x"}
	mustNoErr(t, s.AddSubmission(broken))

	g := 4.0
	_, err = s.SaveAIDraft(f.prof.ID, ana.ID, AIDraft{Feedback: "IA", Grade: &g, Confidence: "alta", Minutes: 4})
	mustNoErr(t, err)
	_, err = s.SaveAIDraft(f.prof.ID, beto.ID, AIDraft{Feedback: "IA", Grade: &g, Confidence: "media", Minutes: 9, Model: "llama3.1:8b", Prompt: "abcd1234"})
	mustNoErr(t, err)
	if _, err := s.SaveAIDraft(f.otro.ID, beto.ID, AIDraft{Feedback: "x", Minutes: 1}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor cambió la estimación: %v", err)
	}
	if rev, _ := s.ReviewForTeacher(f.prof.ID, beto.ID); rev.AIMinutes == nil || *rev.AIMinutes != 9 || rev.AIModel != "llama3.1:8b" || rev.AIPrompt != "abcd1234" {
		t.Fatalf("estimación y origen guardados: %+v", rev)
	}

	if _, err := s.ReviewPlan(f.otro.ID, f.group.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor vio el plan: %v", err)
	}
	plan, err := s.ReviewPlan(f.prof.ID, f.group.ID)
	mustNoErr(t, err)
	if len(plan.Assignments) != 1 {
		t.Fatalf("plan: %+v", plan)
	}
	e := plan.Assignments[0]
	if e.Students != 3 || e.Pending != 3 || e.WithAI != 2 || e.NotCompiling != 1 || e.AllPassing != 1 || e.FailingTests != 1 {
		t.Fatalf("solo cuenta la última entrega de cada estudiante: %+v", e)
	}
	if e.TotalMinutes != 4+9+6 || e.PerSubmission != 19.0/3 {
		t.Fatalf("minutos (IA 4 + IA 9 + 6 estimados para la que no compila): %+v", e)
	}

	_, err = s.SaveTeacherReview(f.prof.ID, ana.ID, "IA", &g, true, nil, nil)
	mustNoErr(t, err)
	_, err = s.AddReviewTime(f.prof.ID, ana.ID, 6*60000)
	mustNoErr(t, err)
	plan, err = s.ReviewPlan(f.prof.ID, f.group.ID)
	mustNoErr(t, err)
	e = plan.Assignments[0]
	if e.Returned != 1 || e.Pending != 2 || plan.Measured != 1 || *plan.MeasuredMinutes != 6 || *plan.EstimatedMinutes != 4 {
		t.Fatalf("ritmo real frente a la estimación: %+v %+v", plan, e)
	}
	if e.TotalMinutes != 9+6 {
		t.Fatalf("sin estimación de la IA se usa el ritmo real del profesor: %+v", e)
	}

	m, err := s.GroupMetrics(f.prof.ID, f.group.ID)
	mustNoErr(t, err)
	if m.ReviewEstimate.Count != 1 || m.OverEstimate != 1 || *m.Thresholds.BatchHours != DefaultBatchHours || *m.Thresholds.FeedbackHours != DefaultFeedbackHours {
		t.Fatalf("RNF-08 contra la estimación y umbral de lote acordado: %+v %d %v", m.ReviewEstimate, m.OverEstimate, *m.Thresholds.BatchHours)
	}
	rev, err := s.DiscardAIDraft(f.prof.ID, beto.ID)
	mustNoErr(t, err)
	if rev.AIMinutes != nil {
		t.Fatalf("descartar quita la estimación: %+v", rev.AIMinutes)
	}
}

func TestComprehensionGoal(t *testing.T) {
	f := newClass(t)
	s := f.s
	second, err := s.CreateAssignment(f.prof.ID, f.group.ID, "Taller 3", "Mayor de tres", "", nil, AssignmentPolicy{AllowLate: true}, nil)
	mustNoErr(t, err)
	rate := func(assignment string, understood string, n int) {
		for i := 0; i < n; i++ {
			mustNoErr(t, s.RateInstant(f.ana.ID, AIKindFormal, assignment, understood, ""))
		}
	}
	rate(f.assignment.ID, "si", 10)
	m, err := s.GroupMetrics(f.prof.ID, f.group.ID)
	mustNoErr(t, err)
	if c := m.Comprehension; c.Verdict != VerdictTooFew || c.ByAssignment[0].Enough || c.Goal.MinResponses != 20 {
		t.Fatalf("con menos de 20 respuestas no se juzga: %+v", c)
	}

	rate(f.assignment.ID, "si", 6)
	rate(f.assignment.ID, "parcial", 2)
	rate(f.assignment.ID, "no", 2)
	m, err = s.GroupMetrics(f.prof.ID, f.group.ID)
	mustNoErr(t, err)
	if c := m.Comprehension; c.Verdict != VerdictMeets || !c.ByAssignment[0].Meets || c.ByAssignment[0].Total != 20 || c.ByAssignment[0].Title != "Taller 2" {
		t.Fatalf("16 sí y 2 no de 20 cumple justo la meta (80 %% y 10 %%): %+v", c.ByAssignment)
	}

	rate(second.ID, "si", 15)
	rate(second.ID, "no", 5)
	rate("", "no", 3)
	m, err = s.GroupMetrics(f.prof.ID, f.group.ID)
	mustNoErr(t, err)
	c := m.Comprehension
	if c.Verdict != VerdictFails || len(c.ByAssignment) != 2 {
		t.Fatalf("un taller con 25 %% de «no» no cumple; la práctica libre sin clase no cuenta: %+v", c)
	}
	for _, a := range c.ByAssignment {
		if a.Title == "Taller 3" && (a.Meets || !a.Enough) {
			t.Fatalf("Taller 3: %+v", a)
		}
	}

	g := 4.0
	sub := f.submit(t, f.beto, KindOfficial, nil)
	_, err = s.SaveTeacherReview(f.prof.ID, sub.ID, "Bien", &g, true, nil, nil)
	mustNoErr(t, err)
	mustNoErr(t, s.RateReview(f.beto.ID, sub.ID, "si", ""))
	m, err = s.GroupMetrics(f.prof.ID, f.group.ID)
	mustNoErr(t, err)
	for _, a := range m.Comprehension.ByAssignment {
		if a.Title == "Taller 2" && a.Total != 21 {
			t.Fatalf("la valoración de una revisión cuenta en su taller: %+v", a)
		}
	}
}

func TestAssignmentAudit(t *testing.T) {
	f := newClass(t)
	s := f.s
	_, err := s.UpdateAssignment(f.prof.ID, f.assignment.ID, "Taller 2: suma", "Suma de 1 a n", "", nil, AssignmentPolicy{AllowLate: true}, nil, nil)
	mustNoErr(t, err)
	tests := []model.TestCase{{Name: "A", Expected: "1", Match: model.MatchContains}, {Name: "B", Expected: "2", Match: model.MatchContains, Hidden: true}}
	_, err = s.SetAssignmentTests(f.prof.ID, f.assignment.ID, tests)
	mustNoErr(t, err)
	_, err = s.SetAssignmentTests(f.prof.ID, f.assignment.ID, tests)
	mustNoErr(t, err)
	if err := s.DeleteAssignment(f.otro.ID, f.assignment.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor eliminó el taller: %v", err)
	}
	mustNoErr(t, s.DeleteAssignment(f.prof.ID, f.assignment.ID))
	log, err := s.GroupAudit(f.prof.ID, f.group.ID, 0)
	mustNoErr(t, err)
	var got []string
	for _, e := range log {
		got = append(got, e.Action+" "+e.Detail)
	}
	want := []string{
		"taller_eliminado «Taller 2: suma»",
		"casos_actualizados «Taller 2: suma»: 2 casos (1 oculto)",
		"taller_editado «Taller 2: suma»",
		"taller_creado «Taller 2»",
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("bitácora de talleres (guardar los mismos casos no se registra):\n%v", got)
	}
}
