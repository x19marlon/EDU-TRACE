package auth

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestClassAdmin(t *testing.T) {
	s := newTestStore(t)
	prof, err := s.CreateUser("Elena", "elena@u.edu", "password123", RoleTeacher)
	mustNoErr(t, err)
	otro, err := s.CreateUser("Otro", "otro@u.edu", "password123", RoleTeacher)
	mustNoErr(t, err)
	ana, err := s.CreateUser("Ana", "ana@u.edu", "password123", RoleStudent)
	mustNoErr(t, err)

	c, err := s.CreateCourse(prof.ID, "Programación I")
	mustNoErr(t, err)
	g, err := s.CreateGroup(prof.ID, c.ID, "Grupo 1")
	mustNoErr(t, err)

	mustNoErr(t, s.RenameCourse(prof.ID, c.ID, "Programación 1"))
	if err := s.RenameCourse(otro.ID, c.ID, "X"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor renombró: %v", err)
	}
	mustNoErr(t, s.RenameGroup(prof.ID, g.ID, "Grupo A"))
	if err := s.RenameGroup(otro.ID, g.ID, "X"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor renombró la clase: %v", err)
	}
	courses := s.TeacherCourses(prof.ID)
	if courses[0].Name != "Programación 1" || courses[0].Groups[0].Name != "Grupo A" {
		t.Fatalf("nombres: %+v", courses)
	}

	code, err := s.RegenerateJoinCode(prof.ID, g.ID)
	mustNoErr(t, err)
	if code == g.JoinCode {
		t.Fatal("el código no cambió")
	}
	if _, err := s.JoinGroup(ana.ID, g.JoinCode); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("el código viejo sigue sirviendo: %v", err)
	}
	_, err = s.JoinGroup(ana.ID, code)
	mustNoErr(t, err)

	a, err := s.CreateAssignment(prof.ID, g.ID, "Taller 1", "Suma", "", nil, AssignmentPolicy{AllowLate: true}, []FileUpload{
		{Name: "a.txt", Body: strings.NewReader("uno")},
		{Name: "b.txt", Body: strings.NewReader("dos")},
	})
	mustNoErr(t, err)
	due := time.Now().Add(48 * time.Hour).UTC().Truncate(time.Second)
	up, err := s.UpdateAssignment(prof.ID, a.ID, "Taller 1b", "Suma de pares", "Compila (1.0)", &due, AssignmentPolicy{AllowLate: true},
		[]string{a.Files[0].ID, "no-existe"}, []FileUpload{{Name: "c.txt", Body: strings.NewReader("tres")}})
	mustNoErr(t, err)
	if up.Title != "Taller 1b" || up.Rubric != "Compila (1.0)" || up.DueAt == nil || !up.DueAt.Equal(due) {
		t.Fatalf("taller editado: %+v", up)
	}
	if len(up.Files) != 2 || up.Files[0].Name != "b.txt" || up.Files[1].Name != "c.txt" {
		t.Fatalf("archivos: %+v", up.Files)
	}
	if _, err := os.Stat(filepath.Join(s.assignmentDir(a.ID), a.Files[0].ID)); !os.IsNotExist(err) {
		t.Fatalf("el archivo quitado sigue en disco: %v", err)
	}
	if _, err := s.UpdateAssignment(otro.ID, a.ID, "X", "Y", "", nil, AssignmentPolicy{AllowLate: true}, nil, nil); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor editó el taller: %v", err)
	}
	many := make([]FileUpload, 9)
	for i := range many {
		many[i] = FileUpload{Name: "f.txt", Body: strings.NewReader("x")}
	}
	if _, err := s.UpdateAssignment(prof.ID, a.ID, "T", "S", "", nil, AssignmentPolicy{AllowLate: true}, nil, many); !errors.Is(err, ErrTooManyFiles) {
		t.Fatalf("más de 10 archivos: %v", err)
	}
	entries, _ := os.ReadDir(s.assignmentDir(a.ID))
	if len(entries) != 2 {
		t.Fatalf("quedaron archivos de un intento fallido: %d", len(entries))
	}

	sub := &Submission{SubmissionMeta: SubmissionMeta{GroupID: g.ID, AssignmentID: a.ID, StudentID: ana.ID, Success: true}, Code: "int main(){}"}
	mustNoErr(t, s.AddSubmission(sub))
	mustNoErr(t, s.DeleteAssignment(prof.ID, a.ID))
	if _, err := os.Stat(s.assignmentDir(a.ID)); !os.IsNotExist(err) {
		t.Fatal("la carpeta del taller sigue en disco")
	}
	got, err := s.SubmissionForTeacher(prof.ID, sub.ID)
	mustNoErr(t, err)
	if got.AssignmentID != "" {
		t.Fatalf("la entrega sigue apuntando al taller borrado: %q", got.AssignmentID)
	}

	if err := s.RemoveStudent(otro.ID, g.ID, ana.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor quitó al estudiante: %v", err)
	}
	mustNoErr(t, s.RemoveStudent(prof.ID, g.ID, ana.ID))
	if s.IsEnrolled(ana.ID, g.ID) {
		t.Fatal("sigue inscrita")
	}
	if _, err := s.SubmissionForTeacher(prof.ID, sub.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("sus entregas siguen en la clase: %v", err)
	}

	a2, err := s.CreateAssignment(prof.ID, g.ID, "Taller 2", "Resta", "", nil, AssignmentPolicy{AllowLate: true}, []FileUpload{{Name: "x.txt", Body: strings.NewReader("x")}})
	mustNoErr(t, err)
	g2, err := s.CreateGroup(prof.ID, c.ID, "Grupo B")
	mustNoErr(t, err)
	if err := s.DeleteGroup(otro.ID, g2.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor borró la clase: %v", err)
	}
	mustNoErr(t, s.DeleteGroup(prof.ID, g2.ID))
	if err := s.DeleteCourse(otro.ID, c.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor borró la materia: %v", err)
	}
	mustNoErr(t, s.DeleteCourse(prof.ID, c.ID))
	if len(s.TeacherCourses(prof.ID)) != 0 {
		t.Fatal("la materia sigue")
	}
	if _, err := os.Stat(s.assignmentDir(a2.ID)); !os.IsNotExist(err) {
		t.Fatal("la carpeta del taller de la materia borrada sigue en disco")
	}
	var n int
	mustNoErr(t, s.db.QueryRow(`SELECT (SELECT COUNT(*) FROM class_groups) + (SELECT COUNT(*) FROM assignments) + (SELECT COUNT(*) FROM assignment_files)`).Scan(&n))
	if n != 0 {
		t.Fatalf("quedaron %d filas huérfanas", n)
	}
}

func TestMonitorRole(t *testing.T) {
	s := newTestStore(t)
	prof, err := s.CreateUser("Elena", "elena@u.edu", "password123", RoleTeacher)
	mustNoErr(t, err)
	otro, err := s.CreateUser("Otro", "otro@u.edu", "password123", RoleTeacher)
	mustNoErr(t, err)
	ana, err := s.CreateUser("Ana", "ana@u.edu", "password123", RoleStudent)
	mustNoErr(t, err)
	moni, err := s.CreateUser("Moni", "moni@u.edu", "password123", RoleMonitor)
	mustNoErr(t, err)
	if u, err := s.Authenticate("moni@u.edu", "password123"); err != nil || u.Role != RoleMonitor {
		t.Fatalf("login monitor: %v %+v", err, u)
	}

	c, err := s.CreateCourse(prof.ID, "Programación I")
	mustNoErr(t, err)
	g, err := s.CreateGroup(prof.ID, c.ID, "Grupo 1")
	mustNoErr(t, err)
	_, err = s.JoinGroup(ana.ID, g.JoinCode)
	mustNoErr(t, err)
	a, err := s.CreateAssignment(prof.ID, g.ID, "Taller", "Enunciado", "", nil, AssignmentPolicy{AllowLate: true}, nil)
	mustNoErr(t, err)
	official := &Submission{SubmissionMeta: SubmissionMeta{GroupID: g.ID, AssignmentID: a.ID, StudentID: ana.ID, Success: true}, Code: "int main(){}"}
	mustNoErr(t, s.AddSubmission(official))
	attempt := &Submission{SubmissionMeta: SubmissionMeta{Kind: KindAttempt, GroupID: g.ID, StudentID: ana.ID}, Code: "int main(){"}
	mustNoErr(t, s.AddSubmission(attempt))

	if _, err := s.GroupForMonitor(moni.ID, g.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ve la clase sin estar asignado: %v", err)
	}
	if _, err := s.AddMonitor(prof.ID, g.ID, "ana@u.edu"); !errors.Is(err, ErrMonitorNotFound) {
		t.Fatalf("se agregó un estudiante como monitor: %v", err)
	}
	if _, err := s.AddMonitor(otro.ID, g.ID, "moni@u.edu"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor agregó monitor: %v", err)
	}
	m, err := s.AddMonitor(prof.ID, g.ID, " MONI@u.edu ")
	mustNoErr(t, err)
	if m.ID != moni.ID {
		t.Fatalf("monitor: %+v", m)
	}
	_, err = s.AddMonitor(prof.ID, g.ID, "moni@u.edu")
	mustNoErr(t, err)
	detail, err := s.GroupForTeacher(prof.ID, g.ID)
	mustNoErr(t, err)
	if len(detail.Monitors) != 1 || detail.Monitors[0].Email != "moni@u.edu" {
		t.Fatalf("monitores en la clase: %+v", detail.Monitors)
	}

	groups := s.MonitorGroups(moni.ID)
	if len(groups) != 1 || groups[0].StudentCount != 1 || groups[0].SubmissionCount != 1 || groups[0].TeacherName != "Elena" {
		t.Fatalf("clases del monitor: %+v", groups)
	}
	mg, err := s.GroupForMonitor(moni.ID, g.ID)
	mustNoErr(t, err)
	if mg.JoinCode != "" || len(mg.Students) != 1 || mg.Students[0].Email != "" {
		t.Fatalf("el monitor ve datos de más: %+v", mg)
	}
	subs, err := s.MonitorGroupSubmissions(moni.ID, g.ID, "", "")
	mustNoErr(t, err)
	if len(subs) != 1 || subs[0].ID != official.ID {
		t.Fatalf("entregas del monitor: %+v", subs)
	}
	list, err := s.MonitorGroupAssignments(moni.ID, g.ID)
	mustNoErr(t, err)
	if len(list) != 1 {
		t.Fatalf("talleres del monitor: %+v", list)
	}
	if _, err := s.AssignmentStatement(moni, a.ID); err != nil {
		t.Fatalf("el monitor no ve el enunciado: %v", err)
	}
	if _, err := s.SubmissionForMonitor(moni.ID, official.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := s.AddMonitorFeedback(moni.ID, attempt.ID, "Bien"); !errors.Is(err, ErrAttemptFeedback) {
		t.Fatalf("retroalimentó un intento: %v", err)
	}
	f1, err := s.AddMonitorFeedback(moni.ID, official.ID, "  Revisa la línea 3  ")
	mustNoErr(t, err)
	if f1.Body != "Revisa la línea 3" || f1.MonitorName != "Moni" {
		t.Fatalf("retroalimentación: %+v", f1)
	}
	f2, err := s.AddMonitorFeedback(moni.ID, official.ID, "Buen uso del for")
	mustNoErr(t, err)

	mine := s.StudentSubmissions(ana.ID, 10)
	if len(mine) != 1 || len(mine[0].MonitorFeedback) != 2 || mine[0].MonitorFeedback[0].ID != f1.ID {
		t.Fatalf("el estudiante no ve la retroalimentación del monitor: %+v", mine)
	}
	if mine[0].Review != nil {
		t.Fatal("la retroalimentación del monitor no es una calificación")
	}
	if n := s.MonitorFeedbackCounts([]string{official.ID, attempt.ID}); n[official.ID] != 2 || n[attempt.ID] != 0 {
		t.Fatalf("conteos: %+v", n)
	}

	if err := s.DeleteMonitorFeedback(ana, f1.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("el estudiante borró: %v", err)
	}
	if err := s.DeleteMonitorFeedback(otro, f1.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor borró: %v", err)
	}
	mustNoErr(t, s.DeleteMonitorFeedback(moni, f1.ID))
	mustNoErr(t, s.DeleteMonitorFeedback(prof, f2.ID))
	if len(s.MonitorFeedbackFor(official.ID)) != 0 {
		t.Fatal("la retroalimentación sigue")
	}

	mustNoErr(t, s.RemoveMonitor(prof.ID, g.ID, moni.ID))
	if _, err := s.SubmissionForMonitor(moni.ID, official.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("sigue viendo entregas después de quitarlo: %v", err)
	}
	if _, err := s.AddMonitorFeedback(moni.ID, official.ID, "Hola"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("retroalimentó después de quitarlo: %v", err)
	}
	if _, err := s.AssignmentStatement(moni, a.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("sigue viendo el enunciado: %v", err)
	}
}

func TestUpgradeAddsMonitorRole(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(dir, "edutrace.db")+"?_pragma=foreign_keys(1)")
	mustNoErr(t, err)
	old := strings.Replace(schemaSQL, "CHECK (role IN ('student', 'teacher', 'monitor'))", "CHECK (role IN ('student', 'teacher'))", 1)
	if old == schemaSQL {
		t.Fatal("no se pudo armar el esquema viejo")
	}
	_, err = db.Exec(old)
	mustNoErr(t, err)
	_, err = db.Exec(`INSERT INTO users (id, name, email, role, password_hash, created_at) VALUES ('p1', 'Profe', 'p@u.edu', 'teacher', 'x', '2026-01-01T00:00:00.000000000Z')`)
	mustNoErr(t, err)
	_, err = db.Exec(`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ('h', 'p1', '2999-01-01T00:00:00.000000000Z')`)
	mustNoErr(t, err)
	_, err = db.Exec(`INSERT INTO courses (id, teacher_id, name, created_at) VALUES ('c1', 'p1', 'Prog', '2026-01-01T00:00:00.000000000Z')`)
	mustNoErr(t, err)
	if _, err := db.Exec(`INSERT INTO users (id, name, email, role, password_hash, created_at) VALUES ('m', 'M', 'm@u.edu', 'monitor', 'x', '2026-01-01T00:00:00.000000000Z')`); err == nil {
		t.Fatal("el esquema viejo aceptó monitor")
	}
	db.Close()

	s, err := OpenStore(dir, time.Hour)
	mustNoErr(t, err)
	defer s.Close()
	_, err = s.CreateUser("Moni", "moni@u.edu", "password123", RoleMonitor)
	mustNoErr(t, err)
	if len(s.TeacherCourses("p1")) != 1 {
		t.Fatal("se perdió la materia")
	}
	var sessions int
	mustNoErr(t, s.db.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&sessions))
	if sessions != 1 {
		t.Fatal("se perdió la sesión")
	}
	if _, err := s.db.Exec(`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ('z', 'nadie', '2999-01-01')`); err == nil {
		t.Fatal("las claves foráneas quedaron desactivadas")
	}
	if _, err := s.db.Exec(`DELETE FROM users WHERE id = 'p1'`); err == nil {
		t.Fatal("se pudo borrar un profesor con materias: las referencias a users se rompieron")
	}
	s.Close()
	s2, err := OpenStore(dir, time.Hour)
	mustNoErr(t, err)
	s2.Close()
}
