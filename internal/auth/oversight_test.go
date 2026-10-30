package auth

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/marlon/edu-trace/internal/model"
)

type classFixture struct {
	s *Store
	prof, otro, ana, beto *User
	course *Course
	group *Group
	assignment *Assignment
}

func newClass(t *testing.T) *classFixture {
	t.Helper()
	s := newTestStore(t)
	f := &classFixture{s: s}
	var err error
	f.prof, err = s.CreateUser("Elena", "elena@u.edu", "password123", RoleTeacher)
	mustNoErr(t, err)
	f.otro, err = s.CreateUser("Otro", "otro@u.edu", "password123", RoleTeacher)
	mustNoErr(t, err)
	f.ana, err = s.CreateUser("Ana", "ana@u.edu", "password123", RoleStudent)
	mustNoErr(t, err)
	f.beto, err = s.CreateUser("Beto", "beto@u.edu", "password123", RoleStudent)
	mustNoErr(t, err)
	f.course, err = s.CreateCourse(f.prof.ID, "Programación I")
	mustNoErr(t, err)
	f.group, err = s.CreateGroup(f.prof.ID, f.course.ID, "Grupo 1")
	mustNoErr(t, err)
	for _, u := range []*User{f.ana, f.beto} {
		_, err := s.JoinGroup(u.ID, f.group.JoinCode)
		mustNoErr(t, err)
	}
	f.assignment, err = s.CreateAssignment(f.prof.ID, f.group.ID, "Taller 2", "Suma de 1 a n", "", nil, LatePolicy{AllowLate: true}, nil)
	mustNoErr(t, err)
	return f
}

func (f *classFixture) submit(t *testing.T, u *User, kind string, results []model.TestResult) *Submission {
	t.Helper()
	sub := &Submission{SubmissionMeta: SubmissionMeta{Kind: kind, GroupID: f.group.ID, AssignmentID: f.assignment.ID, StudentID: u.ID, Success: true},
		Code: "int main(){} //" + u.ID + time.Now().String(), Tests: results}
	mustNoErr(t, f.s.AddSubmission(sub))
	return sub
}

func TestAssignmentTestsAndResults(t *testing.T) {
	f := newClass(t)
	s := f.s
	if _, err := NormalizeTests([]model.TestCase{{Stdin: "1", Expected: "  "}}); !errors.Is(err, ErrInvalidTests) {
		t.Fatalf("aceptó un caso sin salida esperada: %v", err)
	}
	if _, err := NormalizeTests([]model.TestCase{{Expected: "x", Match: "parecida"}}); !errors.Is(err, ErrInvalidTests) {
		t.Fatalf("aceptó una comparación inválida: %v", err)
	}
	tests, err := NormalizeTests([]model.TestCase{
		{Stdin: "5", Expected: "Suma: 15"},
		{Name: "Secreto", Stdin: "100\r\n", Expected: "Suma: 5050", Match: model.MatchExact, Hidden: true},
	})
	mustNoErr(t, err)
	if tests[0].Name != "Caso 1" || tests[0].Match != model.MatchContains || tests[1].Stdin != "100\n" {
		t.Fatalf("normalización: %+v", tests)
	}
	if _, err := s.SetAssignmentTests(f.otro.ID, f.assignment.ID, tests); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor cambió los casos: %v", err)
	}
	saved, err := s.SetAssignmentTests(f.prof.ID, f.assignment.ID, tests)
	mustNoErr(t, err)
	again, err := s.SetAssignmentTests(f.prof.ID, f.assignment.ID, tests)
	mustNoErr(t, err)
	if again[0].ID != saved[0].ID {
		t.Fatal("guardar los mismos casos no debe reemplazarlos")
	}

	results := []model.TestResult{
		{TestID: saved[0].ID, Passed: true, Output: "Suma: 15"},
		{TestID: saved[1].ID, Passed: false, Output: "Suma: 4950"},
		{TestID: "inexistente", Passed: true},
	}
	sub := f.submit(t, f.ana, KindOfficial, results)
	if sub.TestsTotal != 3 || sub.TestsPassed != 2 {
		t.Fatalf("conteo al enviar: %+v", sub.SubmissionMeta)
	}
	metas, err := s.GroupSubmissions(f.prof.ID, f.group.ID, "", "official")
	mustNoErr(t, err)
	if metas[0].TestsTotal != 2 || metas[0].TestsPassed != 1 {
		t.Fatalf("conteo guardado (sin el caso inexistente): %+v", metas[0])
	}
	full, err := s.SubmissionForTeacher(f.prof.ID, sub.ID)
	mustNoErr(t, err)
	if len(full.Tests) != 2 || full.Tests[1].Expected != "Suma: 5050" || full.Tests[1].Output != "Suma: 4950" {
		t.Fatalf("el profesor ve todo: %+v", full.Tests)
	}
	own, err := s.OwnSubmission(f.ana.ID, sub.ID)
	mustNoErr(t, err)
	if own.Tests[1].Expected != "" || own.Tests[1].Stdin != "" || !own.Tests[1].Hidden || own.Tests[0].Expected != "Suma: 15" {
		t.Fatalf("el estudiante no debe ver el caso oculto: %+v", own.Tests)
	}
	student := s.StudentAssignments(f.ana.ID)
	if len(student[0].Tests) != 2 || student[0].Tests[1].Expected != "" || student[0].Tests[0].Stdin != "5" {
		t.Fatalf("talleres del estudiante: %+v", student[0].Tests)
	}
	teacher, err := s.GroupAssignments(f.prof.ID, f.group.ID)
	mustNoErr(t, err)
	if teacher[0].Tests[1].Expected != "Suma: 5050" {
		t.Fatalf("talleres del profesor: %+v", teacher[0].Tests)
	}

	mustNoErr(t, s.SaveSubmissionTests(f.prof.ID, sub.ID, []model.TestResult{{TestID: saved[0].ID, Passed: true}, {TestID: saved[1].ID, Passed: true}}))
	if err := s.SaveSubmissionTests(f.otro.ID, sub.ID, nil); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor re-ejecutó: %v", err)
	}
	changed, err := s.SetAssignmentTests(f.prof.ID, f.assignment.ID, tests[:1])
	mustNoErr(t, err)
	if len(changed) != 1 || changed[0].ID == saved[0].ID {
		t.Fatalf("cambiar los casos debe reemplazarlos: %+v", changed)
	}
	if left, _ := s.SubmissionTests(sub.ID); len(left) != 0 {
		t.Fatalf("quedaron resultados de casos que ya no existen: %+v", left)
	}
}

func TestDraftQueueJobs(t *testing.T) {
	f := newClass(t)
	s := f.s
	first := f.submit(t, f.ana, KindOfficial, nil)
	f.submit(t, f.ana, KindAttempt, nil)
	second := f.submit(t, f.beto, KindOfficial, nil)
	if states := s.JobStates([]string{first.ID, second.ID}); states[first.ID] != JobPending || len(states) != 2 {
		t.Fatalf("solo las entregas oficiales entran a la cola: %+v", states)
	}

	job, err := s.ClaimJob(time.Minute)
	mustNoErr(t, err)
	if job == nil || job.SubmissionID != first.ID || job.TeacherID != f.prof.ID || job.Attempts != 1 {
		t.Fatalf("primer trabajo: %+v", job)
	}
	s.FinishJob(job.SubmissionID, JobFailed, "Ollama no responde", false)
	job, err = s.ClaimJob(time.Minute)
	mustNoErr(t, err)
	if job.SubmissionID != second.ID {
		t.Fatalf("un trabajo fallido no se reintenta antes de tiempo: %+v", job)
	}
	s.FinishJob(job.SubmissionID, JobDone, "", false)
	if job, _ := s.ClaimJob(time.Minute); job != nil {
		t.Fatalf("no quedan trabajos: %+v", job)
	}
	job, err = s.ClaimJob(-time.Second)
	mustNoErr(t, err)
	if job == nil || job.SubmissionID != first.ID || job.Attempts != 2 {
		t.Fatalf("reintento: %+v", job)
	}
	s.FinishJob(job.SubmissionID, JobFailed, "tu código es demasiado largo", true)
	if job, _ := s.ClaimJob(-time.Second); job != nil {
		t.Fatalf("un error definitivo no se reintenta: %+v", job)
	}
	q, err := s.TeacherQueue(f.prof.ID)
	mustNoErr(t, err)
	if len(q.Failed) != 1 || q.Failed[0].Failed != 1 || !strings.Contains(q.Failed[0].LastError, "largo") {
		t.Fatalf("aviso de fallos al profesor: %+v", q)
	}
	if n, err := s.RetryFailedJobs(f.otro.ID, f.group.ID); !errors.Is(err, ErrNotFound) || n != 0 {
		t.Fatalf("otro profesor reintentó: %v", err)
	}
	n, err := s.RetryFailedJobs(f.prof.ID, f.group.ID)
	mustNoErr(t, err)
	if n != 1 {
		t.Fatalf("reintentados: %d", n)
	}

	needed, err := s.NeedsAIDraft(first.ID)
	mustNoErr(t, err)
	if !needed {
		t.Fatal("sin revisión necesita propuesta")
	}
	g := 4.0
	_, err = s.SaveAIDraft(f.prof.ID, first.ID, "propuesta", &g, "razón", "alta", nil)
	mustNoErr(t, err)
	if needed, _ := s.NeedsAIDraft(first.ID); needed {
		t.Fatal("ya tiene propuesta: no debe generarse otra")
	}
	mustNoErr(t, s.ResetRunningJobs())
	state, err := s.JobFor(f.prof.ID, second.ID)
	mustNoErr(t, err)
	if state == nil || state.Status != JobDone {
		t.Fatalf("estado del trabajo: %+v", state)
	}
}

func TestDiscardOriginAndAudit(t *testing.T) {
	f := newClass(t)
	s := f.s
	sub := f.submit(t, f.ana, KindOfficial, nil)
	if _, err := s.DiscardAIDraft(f.prof.ID, sub.ID); !errors.Is(err, ErrNoAIDraft) {
		t.Fatalf("descartó sin propuesta: %v", err)
	}
	ai := 3.5
	_, err := s.SaveAutoAIDraft(f.prof.ID, sub.ID, "## ¿Qué sigue?\nPrueba con 0.", &ai, "razón", "media", nil)
	mustNoErr(t, err)
	rev, err := s.DiscardAIDraft(f.prof.ID, sub.ID)
	mustNoErr(t, err)
	if rev.AIAt != nil || rev.AIFeedback != "" || rev.Feedback != "" || rev.Grade != nil || rev.AIDiscardedAt == nil {
		t.Fatalf("descartar debe quitar la propuesta y su precarga: %+v", rev)
	}
	if needed, _ := s.NeedsAIDraft(sub.ID); needed {
		t.Fatal("una propuesta descartada no se vuelve a generar sola")
	}

	_, err = s.SaveAIDraft(f.prof.ID, sub.ID, "Texto IA", &ai, "razón", "alta", nil)
	mustNoErr(t, err)
	rev, err = s.ReviewForTeacher(f.prof.ID, sub.ID)
	mustNoErr(t, err)
	if rev.AIDiscardedAt != nil || rev.Feedback != "Texto IA" {
		t.Fatalf("pedir otra propuesta reactiva la IA: %+v", rev)
	}
	mine := 4.0
	_, err = s.SaveTeacherReview(f.prof.ID, sub.ID, "Texto IA con mi aporte", &mine, false, nil)
	mustNoErr(t, err)
	added, err := s.AddReviewTime(f.prof.ID, sub.ID, 90_000)
	mustNoErr(t, err)
	if added != 90_000 {
		t.Fatalf("tiempo sumado: %d", added)
	}
	if added, _ := s.AddReviewTime(f.prof.ID, sub.ID, 10*time.Hour.Milliseconds()); added != maxReviewMs {
		t.Fatalf("el tiempo de una sesión se limita: %d", added)
	}
	rev, err = s.SaveTeacherReview(f.prof.ID, sub.ID, "Texto IA con mi aporte", &mine, true, nil)
	mustNoErr(t, err)
	if rev.Origin != OriginAIEdited || rev.ReviewMs != 90_000+maxReviewMs {
		t.Fatalf("origen y tiempo: %+v", rev)
	}
	if _, err := s.DiscardAIDraft(f.prof.ID, sub.ID); !errors.Is(err, ErrAlreadyReturned) {
		t.Fatalf("descartó después de devolver: %v", err)
	}
	late := f.submit(t, f.beto, KindOfficial, nil)
	_, err = s.SaveTeacherReview(f.prof.ID, late.ID, "Revisada sin IA", &mine, true, nil)
	mustNoErr(t, err)
	if _, err := s.SaveAutoAIDraft(f.prof.ID, late.ID, "tarde", &ai, "", "alta", nil); !errors.Is(err, ErrAlreadyReturned) {
		t.Fatalf("la propuesta automática no debe llegar a una revisión ya devuelta: %v", err)
	}
	if rev, _ := s.ReviewForTeacher(f.prof.ID, late.ID); rev.AIAt != nil {
		t.Fatalf("quedó una propuesta en una revisión devuelta: %+v", rev)
	}
	seen := s.StudentSubmissions(f.ana.ID, 5)
	if seen[0].Review == nil || seen[0].Review.Origin != OriginAIEdited || seen[0].Review.TeacherName != "Elena" || !seen[0].Review.AIAssisted {
		t.Fatalf("el estudiante debe ver quién revisó y el origen: %+v", seen[0].Review)
	}

	log, err := s.SubmissionAudit(f.prof.ID, sub.ID)
	mustNoErr(t, err)
	actions := []string{}
	for _, e := range log {
		actions = append(actions, e.Action)
	}
	want := []string{AuditApproved, AuditDraftSaved, AuditAIDraft, AuditAIDiscarded, AuditAIDraft}
	if strings.Join(actions, ",") != strings.Join(want, ",") {
		t.Fatalf("bitácora: %v", actions)
	}
	if log[4].ActorRole != SystemActorRole || log[4].ActorName != SystemActorName || log[2].ActorName != "Elena" {
		t.Fatalf("autores: %+v", log)
	}
	if !strings.Contains(log[0].Detail, "la IA proponía 3.5") || !strings.Contains(log[0].Detail, "editado") || log[0].StudentName != "Ana" {
		t.Fatalf("detalle de la aprobación: %+v", log[0])
	}
	if _, err := s.SubmissionAudit(f.otro.ID, sub.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor vio la bitácora: %v", err)
	}
	group, err := s.GroupAudit(f.prof.ID, f.group.ID, 0)
	mustNoErr(t, err)
	if len(group) != len(log)+1 || group[0].StudentName != "Beto" || group[1].AssignmentTitle != "Taller 2" {
		t.Fatalf("bitácora de la clase: %+v", group)
	}
}

func TestAIModeGate(t *testing.T) {
	f := newClass(t)
	s := f.s
	if f.group.AIMode != AIModeSupervised {
		t.Fatalf("las clases nuevas deben empezar supervisadas: %q", f.group.AIMode)
	}
	if ok, _ := s.StudentAIAllowed(f.ana.ID, f.assignment.ID); ok {
		t.Fatal("en modo supervisado el estudiante no recibe IA directa")
	}
	if ok, _ := s.StudentAIAllowed(f.ana.ID, ""); ok {
		t.Fatal("tampoco en práctica libre si alguna de sus clases es supervisada")
	}
	loner, err := s.CreateUser("Sola", "sola@u.edu", "password123", RoleStudent)
	mustNoErr(t, err)
	if ok, _ := s.StudentAIAllowed(loner.ID, ""); !ok {
		t.Fatal("sin clases puede practicar con IA")
	}
	if err := s.SetGroupAIMode(f.prof.ID, f.group.ID, "libre"); !errors.Is(err, ErrInvalidAIMode) {
		t.Fatalf("aceptó un modo inválido: %v", err)
	}
	if err := s.SetGroupAIMode(f.otro.ID, f.group.ID, AIModeImmediate); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor cambió el modo: %v", err)
	}
	mustNoErr(t, s.SetGroupAIMode(f.prof.ID, f.group.ID, AIModeImmediate))
	if ok, _ := s.StudentAIAllowed(f.ana.ID, f.assignment.ID); !ok {
		t.Fatal("en modo inmediato el estudiante recibe IA")
	}
	if groups := s.StudentGroups(f.ana.ID); groups[0].AIMode != AIModeImmediate {
		t.Fatalf("el estudiante debe conocer el modo: %+v", groups)
	}
	log, err := s.GroupAudit(f.prof.ID, f.group.ID, 10)
	mustNoErr(t, err)
	if len(log) != 1 || log[0].Action != AuditAIMode || !strings.Contains(log[0].Detail, "supervisada → inmediata") {
		t.Fatalf("el cambio de modo queda en la bitácora: %+v", log)
	}
}

func TestGroupMetrics(t *testing.T) {
	f := newClass(t)
	s := f.s
	if _, err := s.GroupMetrics(f.otro.ID, f.group.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor vio las métricas: %v", err)
	}
	ten, day := 10.0, 24.0
	if err := s.SetCourseThresholds(f.prof.ID, f.course.ID, Thresholds{ReviewMinutes: &ten, FeedbackHours: &day, BatchHours: &day}); err != nil {
		t.Fatal(err)
	}
	zero := 0.0
	if err := s.SetCourseThresholds(f.prof.ID, f.course.ID, Thresholds{ReviewMinutes: &zero}); !errors.Is(err, ErrInvalidThreshold) {
		t.Fatalf("aceptó un umbral de 0: %v", err)
	}
	if err := s.SetCourseThresholds(f.otro.ID, f.course.ID, Thresholds{}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor cambió los umbrales: %v", err)
	}

	quick := f.submit(t, f.ana, KindOfficial, nil)
	slow := f.submit(t, f.beto, KindOfficial, nil)
	f.submit(t, f.beto, KindOfficial, nil)
	g := 4.0
	_, err := s.SaveAIDraft(f.prof.ID, quick.ID, "IA", &g, "", "alta", nil)
	mustNoErr(t, err)
	_, err = s.SaveTeacherReview(f.prof.ID, quick.ID, "IA", &g, true, nil)
	mustNoErr(t, err)
	_, err = s.AddReviewTime(f.prof.ID, quick.ID, 5*60*1000)
	mustNoErr(t, err)
	_, err = s.SaveTeacherReview(f.prof.ID, slow.ID, "Escrita por mí", &g, true, nil)
	mustNoErr(t, err)
	_, err = s.AddReviewTime(f.prof.ID, slow.ID, 15*60*1000)
	mustNoErr(t, err)
	_, err = s.db.Exec(`UPDATE submissions SET created_at = ? WHERE id = ?`, fmtTime(time.Now().Add(-48*time.Hour)), slow.ID)
	mustNoErr(t, err)
	job, err := s.ClaimJob(time.Minute)
	mustNoErr(t, err)
	s.FinishJob(job.SubmissionID, JobDone, "", false)
	yes, no := true, false
	s.RecordAIRequest(AIRequest{Kind: AIKindFormal, GroupID: f.group.ID, Role: RoleStudent, FirstMs: 900, TotalMs: 30000, OK: true, SectionsOK: &yes})
	s.RecordAIRequest(AIRequest{Kind: AIKindInformal, GroupID: f.group.ID, Role: RoleStudent, FirstMs: 1100, TotalMs: 50000, OK: true, SectionsOK: &no})
	s.RecordAIRequest(AIRequest{Kind: AIKindFormal, GroupID: f.group.ID, Role: RoleStudent, TotalMs: 100, OK: false, Error: "sin conexión"})
	s.RecordAIRequest(AIRequest{Kind: AIKindChat, GroupID: f.group.ID, Role: RoleStudent, TotalMs: 100, OK: true})
	s.RecordAIRequest(AIRequest{Kind: AIKindFormal, GroupID: "no-existe", Role: RoleStudent, TotalMs: 100, OK: true})

	m, err := s.GroupMetrics(f.prof.ID, f.group.ID)
	mustNoErr(t, err)
	if m.ReviewTime.Count != 2 || m.ReviewTime.Over != 1 || m.ReviewTime.AvgMs != 10*60*1000 || m.ReviewTime.P90Ms != 15*60*1000 {
		t.Fatalf("tiempo de revisión (RNF-08): %+v", m.ReviewTime)
	}
	if m.Turnaround.Count != 2 || m.Turnaround.Over != 1 || m.PendingReviews != 1 || m.PendingOver != 0 {
		t.Fatalf("entrega → retroalimentación (RNF-11): %+v pendientes %d", m.Turnaround, m.PendingReviews)
	}
	if m.Queue.Done != 1 || m.Queue.Pending != 2 || len(m.Batches) != 1 || m.Batches[0].Submissions != 3 || m.Batches[0].Finished {
		t.Fatalf("cola y lote (RNF-10): %+v %+v", m.Queue, m.Batches)
	}
	if m.Instant.Requests != 3 || m.Instant.Failures != 1 || m.Instant.Chats != 1 || m.Instant.SectionsChecked != 2 || m.Instant.SectionsOK != 1 || m.Instant.FirstToken.AvgMs != 1000 {
		t.Fatalf("IA inmediata: %+v", m.Instant)
	}
	if m.Decisions.AI != 1 || m.Decisions.Teacher != 1 || *m.Thresholds.ReviewMinutes != 10 {
		t.Fatalf("decisiones y umbrales: %+v %+v", m.Decisions, m.Thresholds)
	}
}

func TestPersonalData(t *testing.T) {
	f := newClass(t)
	s := f.s
	if f.ana.ConsentAt != nil {
		t.Fatal("una cuenta nueva aún no tiene autorización registrada")
	}
	_, err := s.RecordConsent(f.ana.ID)
	mustNoErr(t, err)
	u, err := s.Authenticate("ana@u.edu", "password123")
	mustNoErr(t, err)
	if u.ConsentAt == nil {
		t.Fatal("la autorización no quedó guardada")
	}
	mustNoErr(t, s.UpdateName(f.ana.ID, "  Ana María "))
	sub := f.submit(t, f.ana, KindOfficial, nil)
	g := 4.0
	_, err = s.SaveTeacherReview(f.prof.ID, sub.ID, "Bien", &g, true, nil)
	mustNoErr(t, err)

	data, err := s.ExportData(u)
	mustNoErr(t, err)
	if data.Profile.Name != "Ana María" || data.Profile.ConsentVersion != PrivacyVersion || len(data.Classes) != 1 ||
		len(data.Submissions) != 1 || data.Submissions[0].Code == "" || data.Submissions[0].Review == nil {
		t.Fatalf("exportación: %+v", data)
	}

	if err := s.DeleteAccount(f.prof.ID, "password123"); !errors.Is(err, ErrOwnsCourses) {
		t.Fatalf("un profesor con materias no puede borrar su cuenta: %v", err)
	}
	if err := s.DeleteAccount(f.ana.ID, "otra-clave"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("borró sin la contraseña correcta: %v", err)
	}
	mustNoErr(t, s.DeleteAccount(f.ana.ID, "password123"))
	var n int
	mustNoErr(t, s.db.QueryRow(`SELECT (SELECT COUNT(*) FROM submissions WHERE student_id = ?) + (SELECT COUNT(*) FROM audit_log WHERE submission_id = ?)
		+ (SELECT COUNT(*) FROM ai_jobs WHERE submission_id = ?)`, f.ana.ID, sub.ID, sub.ID).Scan(&n))
	if n != 0 {
		t.Fatalf("quedaron %d registros de la cuenta borrada", n)
	}
	if _, err := s.Authenticate("ana@u.edu", "password123"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("la cuenta sigue existiendo: %v", err)
	}
	mustNoErr(t, s.DeleteCourse(f.prof.ID, f.course.ID))
	mustNoErr(t, s.DeleteAccount(f.prof.ID, "password123"))
}
