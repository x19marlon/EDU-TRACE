package auth

import (
	"errors"
	"os"
	"testing"
	"time"
)

func TestEvaluationMode(t *testing.T) {
	f := newClass(t)
	s := f.s
	mustNoErr(t, s.SetGroupAIMode(f.prof.ID, f.group.ID, AIModeImmediate))
	if b, _ := s.StudentAIBlock(f.ana.ID, ""); b != nil {
		t.Fatalf("en modo inmediato sin evaluaciones la IA debe estar disponible: %+v", b)
	}
	if _, err := s.CreateAssignment(f.prof.ID, f.group.ID, "Parcial", "Enunciado", "", nil, AssignmentPolicy{Evaluation: true}, nil); !errors.Is(err, ErrInvalidDeadline) {
		t.Fatalf("una evaluación sin fecha límite bloquearía la IA para siempre: %v", err)
	}
	due := time.Now().Add(2 * time.Hour)
	exam, err := s.CreateAssignment(f.prof.ID, f.group.ID, "Parcial 1", "Enunciado", "", &due, AssignmentPolicy{Evaluation: true}, nil)
	mustNoErr(t, err)
	if !exam.Evaluation {
		t.Fatal("el taller no quedó en modo evaluación")
	}
	for _, assignment := range []string{"", exam.ID, f.assignment.ID} {
		b, err := s.StudentAIBlock(f.ana.ID, assignment)
		mustNoErr(t, err)
		if b == nil || b.Code != AIBlockEvaluation || b.Evaluation.Title != "Parcial 1" {
			t.Fatalf("durante la evaluación la IA debe estar bloqueada (taller %q): %+v", assignment, b)
		}
	}
	groups := s.StudentGroups(f.ana.ID)
	if len(groups) != 1 || groups[0].Evaluation == nil || groups[0].Evaluation.Until.Sub(due).Abs() > time.Second {
		t.Fatalf("la clase debe mostrar la evaluación en curso: %+v", groups)
	}
	if b, _ := s.StudentAIBlock(f.prof.ID, ""); b != nil {
		t.Fatal("la evaluación solo afecta a los estudiantes de la clase")
	}
	past := time.Now().Add(-time.Minute)
	_, err = s.UpdateAssignment(f.prof.ID, exam.ID, "Parcial 1", "Enunciado", "", &past, AssignmentPolicy{Evaluation: true}, nil, nil)
	mustNoErr(t, err)
	if b, _ := s.StudentAIBlock(f.ana.ID, ""); b != nil {
		t.Fatalf("después de la fecha límite la IA vuelve: %+v", b)
	}
	lateUntil := time.Now().Add(time.Hour)
	_, err = s.UpdateAssignment(f.prof.ID, exam.ID, "Parcial 1", "Enunciado", "", &past, AssignmentPolicy{Evaluation: true, AllowLate: true, LateUntil: &lateUntil}, nil, nil)
	mustNoErr(t, err)
	if b, _ := s.StudentAIBlock(f.ana.ID, ""); b == nil {
		t.Fatal("mientras se aceptan entregas tardías la evaluación sigue en curso")
	}
	log, err := s.GroupAudit(f.prof.ID, f.group.ID, 0)
	mustNoErr(t, err)
	if log[0].Detail != "«Parcial 1» (modo evaluación: sin IA)" {
		t.Fatalf("la bitácora debe indicar el modo evaluación: %+v", log[0])
	}
}

func TestTeacherResetCode(t *testing.T) {
	f := newClass(t)
	s := f.s
	if _, _, err := s.IssueResetCode(f.otro.ID, f.group.ID, f.ana.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor generó un código: %v", err)
	}
	if _, _, err := s.IssueResetCode(f.prof.ID, f.group.ID, f.otro.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("generó un código para alguien que no está en la clase: %v", err)
	}
	personal, err := s.NewRecoveryCode(f.ana.ID)
	mustNoErr(t, err)
	code, expires, err := s.IssueResetCode(f.prof.ID, f.group.ID, f.ana.ID)
	mustNoErr(t, err)
	if d := time.Until(expires); d < ResetCodeTTL-time.Minute || d > ResetCodeTTL {
		t.Fatalf("vencimiento: %v", expires)
	}
	if _, _, err := s.RecoverAccount("ana@u.edu", "AAAA-BBBB-CCCC", "nueva-clave-1"); !errors.Is(err, ErrInvalidRecovery) {
		t.Fatalf("aceptó un código falso: %v", err)
	}
	u, next, err := s.RecoverAccount("ANA@u.edu", code, "nueva-clave-1")
	mustNoErr(t, err)
	if u.ID != f.ana.ID || next == "" {
		t.Fatalf("recuperación: %+v %q", u, next)
	}
	if _, err := s.Authenticate("ana@u.edu", "nueva-clave-1"); err != nil {
		t.Fatalf("la contraseña nueva no funciona: %v", err)
	}
	if _, _, err := s.RecoverAccount("ana@u.edu", code, "otra-clave-2"); !errors.Is(err, ErrInvalidRecovery) {
		t.Fatalf("el código del profesor sirve una sola vez: %v", err)
	}
	if _, _, err := s.RecoverAccount("ana@u.edu", personal, "otra-clave-2"); !errors.Is(err, ErrInvalidRecovery) {
		t.Fatalf("el código personal anterior debe cambiar tras recuperar: %v", err)
	}
	if _, _, err := s.RecoverAccount("ana@u.edu", next, "otra-clave-2"); err != nil {
		t.Fatalf("el código personal nuevo debe seguir sirviendo: %v", err)
	}

	old, _, err := s.IssueResetCode(f.prof.ID, f.group.ID, f.beto.ID)
	mustNoErr(t, err)
	_, err = s.db.Exec(`UPDATE users SET reset_expires_at = ? WHERE id = ?`, fmtTime(time.Now().Add(-time.Minute)), f.beto.ID)
	mustNoErr(t, err)
	if _, _, err := s.RecoverAccount("beto@u.edu", old, "nueva-clave-1"); !errors.Is(err, ErrInvalidRecovery) {
		t.Fatalf("aceptó un código vencido: %v", err)
	}

	list, err := s.Notifications(f.ana.ID)
	mustNoErr(t, err)
	if list.Unread != 1 || list.Items[0].Kind != NoticeResetCode {
		t.Fatalf("el estudiante debe enterarse del código: %+v", list)
	}
	log, err := s.GroupAudit(f.prof.ID, f.group.ID, 0)
	mustNoErr(t, err)
	if log[0].Action != AuditResetCode || log[0].Detail != "Generó un código para restablecer la contraseña de Beto" {
		t.Fatalf("bitácora: %+v", log[0])
	}
}

func TestNotifications(t *testing.T) {
	f := newClass(t)
	s := f.s
	sub := f.submit(t, f.ana, KindOfficial, nil)
	g := 4.0
	_, err := s.SaveTeacherReview(f.prof.ID, sub.ID, "Bien", &g, false, nil, nil)
	mustNoErr(t, err)
	if list, _ := s.Notifications(f.ana.ID); list.Unread != 0 {
		t.Fatalf("un borrador no debe avisar al estudiante: %+v", list)
	}
	_, err = s.SaveTeacherReview(f.prof.ID, sub.ID, "Bien", &g, true, nil, nil)
	mustNoErr(t, err)
	_, err = s.SaveTeacherReview(f.prof.ID, sub.ID, "Bien", &g, true, nil, nil)
	mustNoErr(t, err)
	g = 4.5
	_, err = s.SaveTeacherReview(f.prof.ID, sub.ID, "Muy bien", &g, true, nil, nil)
	mustNoErr(t, err)
	q, err := s.AskQuestion(f.ana.ID, sub.ID, "¿Por qué?")
	mustNoErr(t, err)
	_, err = s.AnswerQuestion(f.prof, q.ID, "Porque sí")
	mustNoErr(t, err)

	list, err := s.Notifications(f.ana.ID)
	mustNoErr(t, err)
	kinds := []string{NoticeQuestionAnswered, NoticeReviewUpdated, NoticeReviewReturned}
	if list.Unread != 3 || len(list.Items) != 3 || !list.Email {
		t.Fatalf("avisos del estudiante: %+v", list)
	}
	for i, k := range kinds {
		if list.Items[i].Kind != k || list.Items[i].Link != "/historial" {
			t.Fatalf("aviso %d: %+v", i, list.Items[i])
		}
	}
	if list.Items[2].Title != "Tu profesor devolvió la revisión de la entrega de «Taller 2»." {
		t.Fatalf("texto: %q", list.Items[2].Title)
	}
	teacher, err := s.Notifications(f.prof.ID)
	mustNoErr(t, err)
	if teacher.Unread != 1 || teacher.Items[0].Kind != NoticeQuestionAsked || teacher.Items[0].Link != "/docente" {
		t.Fatalf("el profesor debe enterarse de la pregunta: %+v", teacher)
	}

	now := time.Now().Add(time.Minute)
	pending, err := s.PendingEmails(now, now.Add(-time.Hour), 10)
	mustNoErr(t, err)
	if len(pending) != 4 || pending[0].Email == "" {
		t.Fatalf("correos pendientes: %+v", pending)
	}
	mustNoErr(t, s.MarkNotificationsRead(f.ana.ID, []int64{list.Items[0].ID, list.Items[0].ID + 1000}))
	mustNoErr(t, s.SetEmailNotifications(f.prof.ID, false))
	pending, err = s.PendingEmails(now, now.Add(-time.Hour), 10)
	mustNoErr(t, err)
	if len(pending) != 2 {
		t.Fatalf("no se envían avisos leídos ni a quien desactivó el correo: %+v", pending)
	}
	mustNoErr(t, s.MarkEmailed([]int64{pending[0].ID, pending[1].ID}))
	if pending, _ = s.PendingEmails(now, now.Add(-time.Hour), 10); len(pending) != 0 {
		t.Fatalf("un aviso se envía una sola vez: %+v", pending)
	}
	if pending, _ = s.PendingEmails(time.Now().Add(-time.Hour), time.Now().Add(-2*time.Hour), 10); len(pending) != 0 {
		t.Fatal("no se envían avisos fuera de la ventana")
	}
	mustNoErr(t, s.MarkNotificationsRead(f.ana.ID, nil))
	if list, _ = s.Notifications(f.ana.ID); list.Unread != 0 {
		t.Fatalf("marcar todo como leído: %+v", list)
	}
	if err := s.MarkNotificationsRead(f.beto.ID, nil); err != nil {
		t.Fatal(err)
	}
	if teacher, _ = s.Notifications(f.prof.ID); teacher.Unread != 1 {
		t.Fatal("un usuario no puede marcar los avisos de otro")
	}
}

func TestCloseCourseAndPurge(t *testing.T) {
	f := newClass(t)
	s := f.s
	sub := f.submit(t, f.ana, KindOfficial, nil)
	if _, err := s.CloseCourse(f.otro.ID, f.course.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor cerró la materia: %v", err)
	}
	purge, err := s.CloseCourse(f.prof.ID, f.course.ID)
	mustNoErr(t, err)
	if purge == nil || time.Until(*purge) < DefaultRetention-time.Hour {
		t.Fatalf("fecha de borrado: %v", purge)
	}
	again, err := s.CloseCourse(f.prof.ID, f.course.ID)
	mustNoErr(t, err)
	if !again.Equal(*purge) {
		t.Fatal("cerrar dos veces no debe mover la fecha de borrado")
	}
	courses := s.TeacherCourses(f.prof.ID)
	if courses[0].ClosedAt == nil || courses[0].PurgeAt == nil {
		t.Fatalf("la materia debe mostrarse cerrada: %+v", courses[0])
	}
	if groups := s.StudentGroups(f.ana.ID); groups[0].ClosedAt == nil {
		t.Fatalf("el estudiante debe ver la clase cerrada: %+v", groups[0])
	}
	if err := s.AddSubmission(&Submission{SubmissionMeta: SubmissionMeta{Kind: KindOfficial, GroupID: f.group.ID, StudentID: f.beto.ID}, Code: "x"}); !errors.Is(err, ErrCourseClosed) {
		t.Fatalf("una materia cerrada no recibe entregas: %v", err)
	}
	carla, err := s.CreateUser("Carla", "carla@u.edu", "password123", RoleStudent)
	mustNoErr(t, err)
	if _, err := s.JoinGroup(carla.ID, f.group.JoinCode); !errors.Is(err, ErrCourseClosed) {
		t.Fatalf("una materia cerrada no recibe inscripciones: %v", err)
	}
	if b, _ := s.StudentAIBlock(f.ana.ID, ""); b != nil {
		t.Fatalf("una clase supervisada ya cerrada no debe bloquear la práctica libre: %+v", b)
	}
	if list, _ := s.Notifications(f.ana.ID); list.Unread != 1 || list.Items[0].Kind != NoticeCourseClosed {
		t.Fatalf("el estudiante debe saber cuándo se borran sus datos: %+v", list)
	}

	report, err := s.Purge(time.Now())
	mustNoErr(t, err)
	if report.Courses != 0 || report.Accounts != 0 {
		t.Fatalf("no debe borrar antes de tiempo: %+v", report)
	}
	mustNoErr(t, s.ReopenCourse(f.prof.ID, f.course.ID))
	if err := s.ReopenCourse(f.prof.ID, f.course.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("reabrir una materia abierta: %v", err)
	}
	if report, _ = s.Purge(time.Now().Add(2 * DefaultRetention)); report.Courses != 0 || report.Accounts != 1 {
		t.Fatalf("una materia reabierta no se borra; sí la cuenta inactiva de Carla, que no tiene clases: %+v", report)
	}
	_, err = s.CloseCourse(f.prof.ID, f.course.ID)
	mustNoErr(t, err)
	dir := s.assignmentDir(f.assignment.ID)
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("la carpeta del taller debería existir: %v", err)
	}
	report, err = s.Purge(time.Now().Add(DefaultRetention + time.Hour))
	mustNoErr(t, err)
	if report.Courses != 1 {
		t.Fatalf("debe borrar la materia vencida: %+v", report)
	}
	if _, err := s.SubmissionForTeacher(f.prof.ID, sub.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("las entregas deben desaparecer con la materia: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("los archivos del taller deben borrarse: %v", err)
	}
	if report.Accounts != 2 {
		t.Fatalf("debe borrar las cuentas de estudiantes que quedaron sin clases e inactivas (Ana y Beto): %+v", report)
	}
	if _, err := s.Authenticate("elena@u.edu", "password123"); err != nil {
		t.Fatal("las cuentas de profesor nunca se borran solas")
	}
	if _, err := s.Authenticate("ana@u.edu", "password123"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("la cuenta inactiva debía borrarse: %v", err)
	}

	s.SetRetention(0)
	_, err = s.CreateUser("Dani", "dani@u.edu", "password123", RoleStudent)
	mustNoErr(t, err)
	if report, _ = s.Purge(time.Now().Add(10 * DefaultRetention)); !report.Empty() {
		t.Fatalf("con RETENTION_DAYS=0 no se borra nada: %+v", report)
	}
}

func TestQueueStalled(t *testing.T) {
	f := newClass(t)
	s := f.s
	now := time.Now()
	if stalled, _ := s.QueueStalled(now, time.Hour); stalled {
		t.Fatal("sin trabajos la cola no está atascada")
	}
	f.submit(t, f.ana, KindOfficial, nil)
	if stalled, _ := s.QueueStalled(now, time.Hour); stalled {
		t.Fatal("un trabajo recién llegado no es un atasco")
	}
	if stalled, _ := s.QueueStalled(now.Add(2*time.Hour), time.Hour); !stalled {
		t.Fatal("un trabajo que espera más de una hora sin que nada termine es un atasco")
	}
}

func TestLimiterTake(t *testing.T) {
	l := NewLoginLimiter(2, time.Hour)
	for i := 0; i < 2; i++ {
		if _, ok := l.Take("ana"); !ok {
			t.Fatalf("pedido %d rechazado", i+1)
		}
	}
	wait, ok := l.Take("ana")
	if ok || wait <= 0 || wait > time.Hour {
		t.Fatalf("el tercer pedido debía esperar: %v %v", wait, ok)
	}
	if _, ok := l.Take("beto"); !ok {
		t.Fatal("el límite es por persona")
	}
}

func TestConsentVersion(t *testing.T) {
	s := newTestStore(t)
	u, err := s.CreateUser("Ana", "ana@u.edu", "password123", RoleStudent)
	mustNoErr(t, err)
	if u, _ = s.Authenticate("ana@u.edu", "password123"); u.ConsentCurrent() {
		t.Fatal("sin autorización no puede estar al día")
	}
	_, err = s.RecordConsent(u.ID)
	mustNoErr(t, err)
	if u, _ = s.Authenticate("ana@u.edu", "password123"); !u.ConsentCurrent() {
		t.Fatalf("recién autorizada debe estar al día: %+v", u)
	}
	_, err = s.db.Exec(`UPDATE users SET consent_version = '2026-10' WHERE id = ?`, u.ID)
	mustNoErr(t, err)
	if u, _ = s.Authenticate("ana@u.edu", "password123"); u.ConsentCurrent() {
		t.Fatal("con una versión anterior de la política debe volver a autorizar")
	}
}
