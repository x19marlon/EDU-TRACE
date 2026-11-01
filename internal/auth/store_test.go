package auth

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func mustNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := OpenStore(t.TempDir(), time.Hour)
	mustNoErr(t, err)
	t.Cleanup(func() { s.Close() })
	return s
}

func TestStoreFlow(t *testing.T) {
	s := newTestStore(t)

	// Usuarios y sesiones
	prof, err := s.CreateUser("Elena Torres", "Elena@U.edu ", "password123", RoleTeacher)
	mustNoErr(t, err)
	otro, err := s.CreateUser("Otro Profe", "otro@u.edu", "password123", RoleTeacher)
	mustNoErr(t, err)
	ana, err := s.CreateUser("Ana Alumna", "ana@u.edu", "password123", RoleStudent)
	mustNoErr(t, err)
	beto, err := s.CreateUser("Beto", "beto@u.edu", "password123", RoleStudent)
	mustNoErr(t, err)
	if _, err := s.CreateUser("X", "ELENA@u.edu", "password123", RoleStudent); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("email repetido: %v", err)
	}
	if _, err := s.Authenticate("elena@u.edu", "mala"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("contraseña mala: %v", err)
	}
	if _, err := s.Authenticate("nadie@u.edu", "password123"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("email inexistente: %v", err)
	}
	u, err := s.Authenticate(" ELENA@u.edu", "password123")
	mustNoErr(t, err)
	if u.ID != prof.ID || u.Role != RoleTeacher {
		t.Fatalf("login: %+v", u)
	}
	token, _, err := s.CreateSession(prof.ID)
	mustNoErr(t, err)
	if u, ok := s.UserForSession(token); !ok || u.ID != prof.ID {
		t.Fatal("sesión no encontrada")
	}
	mustNoErr(t, s.DeleteSession(token))
	if _, ok := s.UserForSession(token); ok {
		t.Fatal("la sesión debería estar borrada")
	}

	// Materias y clases
	c, err := s.CreateCourse(prof.ID, "Programación I")
	mustNoErr(t, err)
	if _, err := s.CreateGroup(otro.ID, c.ID, "Intruso"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor crea clase: %v", err)
	}
	g, err := s.CreateGroup(prof.ID, c.ID, "Grupo 1")
	mustNoErr(t, err)
	if len(g.JoinCode) != 6 {
		t.Fatalf("código: %q", g.JoinCode)
	}

	// Inscripción
	if _, err := s.JoinGroup(ana.ID, "ZZZZZZ"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("código inválido: %v", err)
	}
	lc := strings.ToLower(g.JoinCode)
	sg, err := s.JoinGroup(ana.ID, lc[:3]+"-"+lc[3:])
	mustNoErr(t, err)
	if sg.ID != g.ID || sg.CourseName != "Programación I" || sg.TeacherName != "Elena Torres" {
		t.Fatalf("clase del estudiante: %+v", sg)
	}
	_, err = s.JoinGroup(ana.ID, g.JoinCode) // idempotente
	mustNoErr(t, err)
	if gs := s.StudentGroups(ana.ID); len(gs) != 1 {
		t.Fatalf("clases de Ana: %d", len(gs))
	}
	if !s.IsEnrolled(ana.ID, g.ID) || s.IsEnrolled(beto.ID, g.ID) {
		t.Fatal("IsEnrolled")
	}

	// Envíos e intentos
	attempt := &Submission{SubmissionMeta: SubmissionMeta{Kind: KindAttempt, GroupID: g.ID, StudentID: ana.ID},
		Code: "int main(){}\n", Stdin: "1"}
	mustNoErr(t, s.AddSubmission(attempt))
	dup := &Submission{SubmissionMeta: SubmissionMeta{Kind: KindAttempt, GroupID: g.ID, StudentID: ana.ID},
		Code: "int main(){}\n", Stdin: "1"}
	if err := s.AddSubmission(dup); !errors.Is(err, ErrDuplicateAttempt) {
		t.Fatalf("intento repetido: %v", err)
	}
	if err := s.AddSubmission(&Submission{SubmissionMeta: SubmissionMeta{GroupID: g.ID, StudentID: beto.ID},
		Code: "x"}); !errors.Is(err, ErrNotEnrolled) {
		t.Fatalf("no inscrito: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	official := &Submission{SubmissionMeta: SubmissionMeta{GroupID: g.ID, StudentID: ana.ID, Success: true},
		Code: "int main(){ return 0; }\n", ProgramOutput: "ok", AIFeedback: "## ¿Cómo voy?\nbien"}
	mustNoErr(t, s.AddSubmission(official))
	if official.Kind != KindOfficial || !official.HasAIFeedback || official.Lines != 1 {
		t.Fatalf("entrega oficial: %+v", official.SubmissionMeta)
	}

	if list, _ := s.GroupSubmissions(prof.ID, g.ID, "", ""); len(list) != 1 || list[0].ID != official.ID {
		t.Fatalf("oficiales: %+v", list)
	}
	if list, _ := s.GroupSubmissions(prof.ID, g.ID, ana.ID, KindAttempt); len(list) != 1 {
		t.Fatalf("intentos: %d", len(list))
	}
	all, _ := s.GroupSubmissions(prof.ID, g.ID, "", "all")
	if len(all) != 2 || all[0].ID != official.ID { // más reciente primero
		t.Fatalf("todo: %+v", all)
	}
	if _, err := s.GroupSubmissions(otro.ID, g.ID, "", "all"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor lista envíos: %v", err)
	}
	full, err := s.SubmissionForTeacher(prof.ID, official.ID)
	mustNoErr(t, err)
	if full.Code != official.Code || full.ProgramOutput != "ok" || full.AssignmentID != "" {
		t.Fatalf("envío completo: %+v", full)
	}
	if _, err := s.SubmissionForTeacher(otro.ID, official.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor ve envío: %v", err)
	}

	// Resúmenes del profesor
	courses := s.TeacherCourses(prof.ID)
	if len(courses) != 1 || len(courses[0].Groups) != 1 ||
		courses[0].Groups[0].StudentCount != 1 || courses[0].Groups[0].SubmissionCount != 1 {
		t.Fatalf("materias: %+v", courses)
	}
	if len(s.TeacherCourses(otro.ID)) != 0 {
		t.Fatal("el otro profesor no tiene materias")
	}
	s.RecordCompile(ana.ID)
	s.RecordFeedback(ana.ID)
	detail, err := s.GroupForTeacher(prof.ID, g.ID)
	mustNoErr(t, err)
	st := detail.Students[0]
	if len(detail.Students) != 1 || st.SubmissionCount != 1 || st.AttemptCount != 1 ||
		st.LastSubmissionAt == nil || st.LastAttemptAt == nil || st.Activity.Compiles != 1 ||
		st.Activity.Feedbacks != 1 || st.Activity.LastActiveAt.IsZero() {
		t.Fatalf("detalle de clase: %+v", detail)
	}
	name, process, err := s.StudentProcess(prof.ID, g.ID, ana.ID, 20)
	mustNoErr(t, err)
	if name != "Ana Alumna" || len(process) != 2 || process[0].ID != attempt.ID { // del más antiguo al más reciente
		t.Fatalf("proceso: %s %+v", name, process)
	}
	if _, _, err := s.StudentProcess(prof.ID, g.ID, beto.ID, 20); !errors.Is(err, ErrNotFound) {
		t.Fatalf("proceso de no inscrito: %v", err)
	}

	// Revisiones
	if _, err := s.ReviewForTeacher(prof.ID, attempt.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("revisar un intento: %v", err)
	}
	if _, err := s.ReviewForTeacher(otro.ID, official.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor revisa: %v", err)
	}
	r, err := s.ReviewForTeacher(prof.ID, official.ID)
	mustNoErr(t, err)
	if r.Status != ReviewDraft || r.Feedback != "" {
		t.Fatalf("revisión vacía: %+v", r)
	}
	g40 := 4.0
	r, err = s.SaveAIDraft(prof.ID, official.ID, AIDraft{Feedback: "borrador IA", Grade: &g40, Reason: "razón", Confidence: "media"})
	mustNoErr(t, err)
	if r.Feedback != "borrador IA" || r.Grade == nil || *r.Grade != 4 || r.AIAt == nil || r.GradeSource() != "" {
		t.Fatalf("propuesta IA: %+v", r)
	}
	g35 := 3.5
	r, err = s.SaveTeacherReview(prof.ID, official.ID, "mi texto", &g35, false, nil, nil)
	mustNoErr(t, err)
	if r.Status != ReviewDraft || *r.AIGrade != 4 || r.GradeSource() != "profesor" {
		t.Fatalf("guardar sin devolver: %+v", r)
	}
	if subs := s.StudentSubmissions(ana.ID, 20); len(subs) != 1 || subs[0].Review != nil {
		t.Fatalf("el estudiante no debe ver un borrador: %+v", subs)
	}
	states := s.ReviewStates([]string{official.ID, attempt.ID})
	if len(states) != 1 || states[official.ID].Status != ReviewDraft || *states[official.ID].Grade != 3.5 {
		t.Fatalf("estados: %+v", states)
	}
	r, err = s.SaveTeacherReview(prof.ID, official.ID, "mi texto final", &g35, true, nil, nil)
	mustNoErr(t, err)
	if r.Status != ReviewReturned || r.ReturnedAt == nil {
		t.Fatalf("devolver: %+v", r)
	}
	subs := s.StudentSubmissions(ana.ID, 20)
	if len(subs) != 1 || subs[0].Review == nil || subs[0].Review.Feedback != "mi texto final" ||
		*subs[0].Review.Grade != 3.5 || !subs[0].Review.AIAssisted || subs[0].CourseName != "Programación I" {
		t.Fatalf("el estudiante ve su revisión: %+v", subs)
	}
	raw, _ := json.Marshal(subs)
	if strings.Contains(string(raw), "ai_grade") || strings.Contains(string(raw), "razón") {
		t.Fatalf("el estudiante no debe recibir la propuesta de la IA: %s", raw)
	}

	// Talleres
	due := time.Now().Add(48 * time.Hour)
	a, err := s.CreateAssignment(prof.ID, g.ID, "Taller 1", "Enunciado", "Compila (1.0)", &due, AssignmentPolicy{AllowLate: true},
		[]FileUpload{{Name: "../datos.txt", ContentType: "text/plain", Body: strings.NewReader("1 2 3")}})
	mustNoErr(t, err)
	if _, err := s.CreateAssignment(otro.ID, g.ID, "X", "Y", "", nil, AssignmentPolicy{AllowLate: true}, nil); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor crea taller: %v", err)
	}
	list, err := s.GroupAssignments(prof.ID, g.ID)
	mustNoErr(t, err)
	if len(list) != 1 || len(list[0].Files) != 1 || list[0].Files[0].Name != "datos.txt" || list[0].DueAt == nil {
		t.Fatalf("talleres: %+v", list)
	}
	if sa := s.StudentAssignments(ana.ID); len(sa) != 1 || sa[0].GroupName != "Grupo 1" || len(sa[0].Files) != 1 {
		t.Fatalf("talleres de Ana: %+v", sa)
	}
	if sa := s.StudentAssignments(beto.ID); len(sa) != 0 {
		t.Fatal("Beto no debe ver talleres")
	}
	if st, err := s.AssignmentStatement(ana, a.ID); err != nil || st != "Taller 1\nEnunciado" {
		t.Fatalf("enunciado (estudiante): %q %v", st, err)
	}
	if _, err := s.AssignmentStatement(prof, a.ID); err != nil {
		t.Fatalf("enunciado (profesor): %v", err)
	}
	if _, err := s.AssignmentStatement(beto, a.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("enunciado (no inscrito): %v", err)
	}
	if _, err := s.AssignmentStatement(otro, a.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("enunciado (otro profesor): %v", err)
	}
	path, f, err := s.AssignmentFilePath(ana, a.ID, a.Files[0].ID)
	mustNoErr(t, err)
	if body, _ := os.ReadFile(path); string(body) != "1 2 3" || f.Size != 5 {
		t.Fatalf("archivo: %q %+v", body, f)
	}
	if _, _, err := s.AssignmentFilePath(beto, a.ID, a.Files[0].ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("archivo (no inscrito): %v", err)
	}
	if st, rubric, err := s.AssignmentForGrading(prof.ID, a.ID); err != nil || rubric != "Compila (1.0)" || st == "" {
		t.Fatalf("para calificar: %q %v", rubric, err)
	}
	if !s.AssignmentInGroup(a.ID, g.ID) || s.AssignmentInGroup(a.ID, "otra") {
		t.Fatal("AssignmentInGroup")
	}
	withTaller := &Submission{SubmissionMeta: SubmissionMeta{GroupID: g.ID, AssignmentID: a.ID, StudentID: ana.ID}, Code: "y"}
	mustNoErr(t, s.AddSubmission(withTaller))
	if full, _ := s.SubmissionForTeacher(prof.ID, withTaller.ID); full.AssignmentID != a.ID {
		t.Fatalf("envío con taller: %+v", full.SubmissionMeta)
	}
}

func TestImportLegacyJSON(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().UTC()
	hash, _ := HashPassword("password123")
	grade := 4.5
	legacy := legacyData{
		Users: map[string]*User{
			"t1": {ID: "t1", Name: "Profe", Email: "p@u.edu", Role: RoleTeacher, PasswordHash: hash, CreatedAt: now},
			"s1": {ID: "s1", Name: "Ana", Email: "a@u.edu", Role: RoleStudent, PasswordHash: hash, CreatedAt: now,
				Activity: Activity{Compiles: 3, LastActiveAt: now}},
		},
		Sessions: map[string]*legacySession{
			hashToken("vigente"): {UserID: "s1", ExpiresAt: now.Add(time.Hour)},
			hashToken("vencida"): {UserID: "s1", ExpiresAt: now.Add(-time.Hour)},
		},
		Courses: map[string]*Course{"c1": {ID: "c1", TeacherID: "t1", Name: "Prog I", CreatedAt: now}},
		Groups: map[string]*Group{"g1": {ID: "g1", CourseID: "c1", Name: "G1", JoinCode: "ABC234", StudentIDs: []string{"s1"}, CreatedAt: now}},
		Assignments: map[string]*Assignment{"a1": {ID: "a1", GroupID: "g1", Title: "T", Statement: "E", CreatedAt: now, Files: []AssignmentFile{{ID: "f1", Name: "x.txt", Size: 1}}}},
		Submissions: map[string]*SubmissionMeta{
			"sub1": {ID: "sub1", Kind: "", GroupID: "g1", AssignmentID: "a1", StudentID: "s1", CreatedAt: now, Success: true, Lines: 1},
			"sub2": {ID: "sub2", Kind: KindAttempt, GroupID: "g1", StudentID: "s1", CreatedAt: now.Add(-time.Minute), Lines: 1},
			"lost": {ID: "lost", GroupID: "g1", StudentID: "s1", CreatedAt: now}, // sin archivo: se omite
		},
		Reviews: map[string]*Review{"sub1": {SubmissionID: "sub1", TeacherID: "t1", Feedback: "bien", Grade: &grade,
			Status: ReviewReturned, ReturnedAt: &now, UpdatedAt: now}},
	}
	raw, _ := json.Marshal(legacy)
	mustNoErr(t, os.WriteFile(filepath.Join(dir, "edutrace.json"), raw, 0o600))
	mustNoErr(t, os.MkdirAll(filepath.Join(dir, "submissions"), 0o700))
	for _, id := range []string{"sub1", "sub2"} {
		body, _ := json.Marshal(Submission{SubmissionMeta: *legacy.Submissions[id], Code: "codigo " + id})
		mustNoErr(t, os.WriteFile(filepath.Join(dir, "submissions", id+".json"), body, 0o600))
	}

	s, err := OpenStore(dir, time.Hour)
	mustNoErr(t, err)
	if _, err := os.Stat(filepath.Join(dir, "edutrace.json.migrated")); err != nil {
		t.Fatal("el JSON debía renombrarse a .migrated")
	}
	if _, err := s.Authenticate("a@u.edu", "password123"); err != nil {
		t.Fatalf("login tras migrar: %v", err)
	}
	if u, ok := s.UserForSession("vigente"); !ok || u.ID != "s1" || u.Activity.Compiles != 3 {
		t.Fatal("la sesión vigente debía conservarse")
	}
	if _, ok := s.UserForSession("vencida"); ok {
		t.Fatal("la sesión vencida no debía migrarse")
	}
	all, err := s.GroupSubmissions("t1", "g1", "", "all")
	mustNoErr(t, err)
	if len(all) != 2 || all[0].ID != "sub1" || all[0].Kind != KindOfficial || all[0].AssignmentID != "a1" {
		t.Fatalf("envíos migrados: %+v", all)
	}
	full, err := s.SubmissionForTeacher("t1", "sub2")
	mustNoErr(t, err)
	if full.Code != "codigo sub2" || full.Kind != KindAttempt {
		t.Fatalf("código migrado: %+v", full)
	}
	if subs := s.StudentSubmissions("s1", 10); len(subs) != 1 || subs[0].Review == nil || *subs[0].Review.Grade != 4.5 {
		t.Fatalf("revisión migrada: %+v", subs)
	}
	if sg := s.StudentGroups("s1"); len(sg) != 1 {
		t.Fatal("inscripción migrada")
	}
	if list, _ := s.GroupAssignments("t1", "g1"); len(list) != 1 || len(list[0].Files) != 1 {
		t.Fatalf("talleres migrados: %+v", list)
	}
	s.Close()

	// Abrir de nuevo no vuelve a migrar ni falla.
	s2, err := OpenStore(dir, time.Hour)
	mustNoErr(t, err)
	defer s2.Close()
	if all, _ := s2.GroupSubmissions("t1", "g1", "", "all"); len(all) != 2 {
		t.Fatalf("segunda apertura: %d envíos", len(all))
	}
}

func TestConcurrentWrites(t *testing.T) {
	s := newTestStore(t)
	prof, err := s.CreateUser("Profe", "p@u.edu", "password123", RoleTeacher)
	mustNoErr(t, err)
	c, err := s.CreateCourse(prof.ID, "Prog")
	mustNoErr(t, err)
	g, err := s.CreateGroup(prof.ID, c.ID, "G")
	mustNoErr(t, err)

	const n = 20
	students := make([]*User, n)
	for i := range students {
		students[i], err = s.CreateUser("E", "e"+strings.Repeat("x", i)+"@u.edu", "password123", RoleStudent)
		mustNoErr(t, err)
		_, err = s.JoinGroup(students[i].ID, g.JoinCode)
		mustNoErr(t, err)
	}
	errs := make(chan error, 2*n)
	for i := 0; i < n; i++ {
		go func(u *User, i int) {
			errs <- s.AddSubmission(&Submission{SubmissionMeta: SubmissionMeta{GroupID: g.ID, StudentID: u.ID},
				Code: strings.Repeat("x", i+1)})
		}(students[i], i)
		go func(u *User) { s.RecordCompile(u.ID); errs <- nil }(students[i])
	}
	for i := 0; i < 2*n; i++ {
		mustNoErr(t, <-errs)
	}
	if list, _ := s.GroupSubmissions(prof.ID, g.ID, "", ""); len(list) != n {
		t.Fatalf("envíos: %d", len(list))
	}
}
