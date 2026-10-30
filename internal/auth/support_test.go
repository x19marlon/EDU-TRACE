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
	_, err = s.SaveTeacherReview(f.prof.ID, sub.ID, "Bien", &g, true, nil)
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
	second, err := s.CreateAssignment(f.prof.ID, f.group.ID, "Taller 3", "Mayor de tres", "", nil, LatePolicy{AllowLate: true}, nil)
	mustNoErr(t, err)
	g1, g2, g3 := 3.0, 4.5, 5.0
	old := f.submit(t, f.ana, KindOfficial, nil)
	_, err = s.SaveTeacherReview(f.prof.ID, old.ID, "Primera", &g1, true, nil)
	mustNoErr(t, err)
	last := f.submit(t, f.ana, KindOfficial, nil)
	_, err = s.SaveTeacherReview(f.prof.ID, last.ID, "Segunda", &g2, true, nil)
	mustNoErr(t, err)
	other := &Submission{SubmissionMeta: SubmissionMeta{Kind: KindOfficial, GroupID: f.group.ID, AssignmentID: second.ID, StudentID: f.ana.ID, Success: true}, Code: "y"}
	mustNoErr(t, s.AddSubmission(other))
	_, err = s.SaveTeacherReview(f.prof.ID, other.ID, "=HYPERLINK(\"x\")", &g3, true, nil)
	mustNoErr(t, err)
	draft := f.submit(t, f.beto, KindOfficial, nil)
	_, err = s.SaveTeacherReview(f.prof.ID, draft.ID, "Borrador", &g3, false, nil)
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
