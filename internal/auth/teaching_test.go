package auth

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/marlon/edu-trace/internal/model"
)

func TestCoTeachers(t *testing.T) {
	f := newClass(t)
	s := f.s
	if _, err := s.AddCoTeacher(f.otro.ID, f.course.ID, "elena@u.edu"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("solo el dueño agrega profesores: %v", err)
	}
	if _, err := s.AddCoTeacher(f.prof.ID, f.course.ID, "ana@u.edu"); !errors.Is(err, ErrTeacherNotFound) {
		t.Fatalf("una estudiante no puede ser profesora: %v", err)
	}
	co, err := s.AddCoTeacher(f.prof.ID, f.course.ID, "OTRO@u.edu")
	mustNoErr(t, err)
	if _, err := s.AddCoTeacher(f.prof.ID, f.course.ID, "otro@u.edu"); !errors.Is(err, ErrAlreadyTeacher) {
		t.Fatalf("agregar dos veces: %v", err)
	}
	courses := s.TeacherCourses(f.otro.ID)
	if len(courses) != 1 || courses[0].Owner || len(courses[0].Teachers) != 2 || !courses[0].Teachers[0].Owner {
		t.Fatalf("el coprofesor debe ver la materia compartida: %+v", courses)
	}
	sub := f.submit(t, f.ana, KindOfficial, nil)
	g := 4.0
	if _, err := s.SaveTeacherReview(co.ID, sub.ID, "Bien hecho", &g, true, nil, nil); err != nil {
		t.Fatalf("el coprofesor revisa y devuelve: %v", err)
	}
	if _, err := s.GroupMetrics(co.ID, f.group.ID); err != nil {
		t.Fatalf("el coprofesor ve los indicadores: %v", err)
	}
	q, err := s.AskQuestion(f.ana.ID, sub.ID, "¿Por qué?")
	mustNoErr(t, err)
	for _, id := range []string{f.prof.ID, co.ID} {
		if list, _ := s.Notifications(id); list.Unread == 0 {
			t.Fatalf("todos los profesores de la materia se enteran de la pregunta (%s)", id)
		}
	}
	if _, err := s.AnswerQuestion(f.otro, q.ID, "Porque sí"); err != nil {
		t.Fatalf("el coprofesor responde: %v", err)
	}
	if err := s.DeleteAccount(co.ID, "password123"); !errors.Is(err, ErrOwnsCourses) {
		t.Fatalf("un coprofesor debe salir de la materia antes de borrar su cuenta: %v", err)
	}
	if err := s.DeleteCourse(co.ID, f.course.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("solo el dueño borra la materia: %v", err)
	}
	mustNoErr(t, s.RemoveCoTeacher(co.ID, f.course.ID, co.ID))
	if rev, _ := s.ReviewForTeacher(f.prof.ID, sub.ID); rev.TeacherID != f.prof.ID {
		t.Fatalf("las revisiones del coprofesor pasan al dueño: %+v", rev.TeacherID)
	}
	if len(s.TeacherCourses(co.ID)) != 0 {
		t.Fatal("tras salir ya no ve la materia")
	}
	if _, err := s.GroupMetrics(co.ID, f.group.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("tras salir ya no ve la clase: %v", err)
	}
}

func TestCriteriaAndClone(t *testing.T) {
	f := newClass(t)
	s := f.s
	if _, err := NormalizeCriteria([]Criterion{{Name: "", Points: 1}}); !errors.Is(err, ErrInvalidCriteria) {
		t.Fatalf("criterio sin nombre: %v", err)
	}
	criteria, err := NormalizeCriteria([]Criterion{{Name: "Compila", Points: 1}, {Name: "Resultado correcto", Points: 3}})
	mustNoErr(t, err)
	saved, err := s.SetAssignmentCriteria(f.prof.ID, f.assignment.ID, criteria)
	mustNoErr(t, err)
	_, err = s.SetAssignmentTests(f.prof.ID, f.assignment.ID, []model.TestCase{{Name: "uno", Stdin: "1", Expected: "1", Match: model.MatchContains}})
	mustNoErr(t, err)
	sub := f.submit(t, f.ana, KindOfficial, nil)
	g := 3.8
	if _, err := s.SaveTeacherReview(f.prof.ID, sub.ID, "Bien", &g, false, nil, map[string]float64{saved[0].ID: 1, saved[1].ID: 4}); !errors.Is(err, ErrInvalidScore) {
		t.Fatalf("un puntaje mayor al máximo: %v", err)
	}
	rev, err := s.SaveTeacherReview(f.prof.ID, sub.ID, "Bien", &g, true, nil, map[string]float64{saved[0].ID: 1, saved[1].ID: 2})
	mustNoErr(t, err)
	if got := CriteriaGrade(rev.Criteria, 5); got == nil || *got != 3.8 {
		t.Fatalf("nota calculada: %v %+v", got, rev.Criteria)
	}
	history, err := s.studentSubmissions(f.ana.ID, KindOfficial, 10)
	mustNoErr(t, err)
	if len(history[0].Review.Criteria) != 2 || *history[0].Review.Criteria[1].Score != 2 {
		t.Fatalf("el estudiante ve los puntajes: %+v", history[0].Review)
	}
	_, rubric, err := s.AssignmentForGrading(f.prof.ID, f.assignment.ID)
	mustNoErr(t, err)
	if !strings.Contains(rubric, "Resultado correcto (3.0 puntos)") {
		t.Fatalf("la IA debe recibir los criterios: %q", rubric)
	}
	shrunk, err := s.SetAssignmentCriteria(f.prof.ID, f.assignment.ID, []Criterion{{Name: "Compila y funciona", Points: 0.5}})
	mustNoErr(t, err)
	if rev, _ = s.ReviewForTeacher(f.prof.ID, sub.ID); len(rev.Criteria) != 1 || rev.Criteria[0].ID != shrunk[0].ID || *rev.Criteria[0].Score != 0.5 {
		t.Fatalf("al cambiar los criterios se conserva el puntaje ajustado al nuevo máximo: %+v", rev.Criteria)
	}

	files := []FileUpload{{Name: "datos.txt", Body: strings.NewReader("1 2 3")}}
	due := time.Now().Add(time.Hour)
	withFile, err := s.CreateAssignment(f.prof.ID, f.group.ID, "Parcial", "Enunciado", "", &due, AssignmentPolicy{Evaluation: true}, files)
	mustNoErr(t, err)
	_, err = s.CreateRAE(f.prof.ID, f.course.ID, "RAE 1", "Resuelve problemas")
	mustNoErr(t, err)
	co, err := s.AddCoTeacher(f.prof.ID, f.course.ID, "otro@u.edu")
	mustNoErr(t, err)
	newID, err := s.CloneCourse(co.ID, f.course.ID, "Programación I (2027-1)", true)
	mustNoErr(t, err)
	courses := s.TeacherCourses(co.ID)
	var clone, old *CourseSummary
	for i := range courses {
		switch courses[i].ID {
		case newID:
			clone = &courses[i]
		case f.course.ID:
			old = &courses[i]
		}
	}
	if clone == nil || !clone.Owner || len(clone.Teachers) != 2 || len(clone.RAEs) != 1 || len(clone.Groups) != 1 || clone.Groups[0].StudentCount != 0 {
		t.Fatalf("copia: %+v", clone)
	}
	if old == nil || old.ClosedAt == nil {
		t.Fatalf("la materia anterior debe quedar cerrada: %+v", old)
	}
	if clone.Groups[0].JoinCode == f.group.JoinCode {
		t.Fatal("la clase copiada necesita un código nuevo")
	}
	list, err := s.groupAssignments(clone.Groups[0].ID, true)
	mustNoErr(t, err)
	if len(list) != 2 {
		t.Fatalf("talleres copiados: %+v", list)
	}
	for _, a := range list {
		if a.DueAt != nil {
			t.Fatalf("las fechas límite quedan vacías: %+v", a)
		}
		switch a.Title {
		case "Taller 2":
			if len(a.Tests) != 1 || len(a.Criteria) != 1 {
				t.Fatalf("casos y criterios copiados: %+v", a)
			}
		case "Parcial":
			if !a.Evaluation || len(a.Files) != 1 {
				t.Fatalf("archivos y modo evaluación copiados: %+v", a)
			}
			body, err := os.ReadFile(filepath.Join(s.assignmentDir(a.ID), a.Files[0].ID))
			if err != nil || string(body) != "1 2 3" {
				t.Fatalf("el archivo debe copiarse en disco: %q %v", body, err)
			}
			if a.Files[0].ID == withFile.Files[0].ID {
				t.Fatal("el archivo copiado necesita un id nuevo")
			}
		}
	}
	if _, err := s.CloneCourse(f.otro.ID, "no-existe", "", false); !errors.Is(err, ErrNotFound) {
		t.Fatalf("copiar una materia ajena: %v", err)
	}
}

func TestAlertsReportAndProgress(t *testing.T) {
	f := newClass(t)
	s := f.s
	r1, err := s.CreateRAE(f.prof.ID, f.course.ID, "RAE 1", "Uno")
	mustNoErr(t, err)
	r2, err := s.CreateRAE(f.prof.ID, f.course.ID, "RAE 2", "Dos")
	mustNoErr(t, err)
	past := time.Now().Add(-48 * time.Hour)
	_, err = s.CreateAssignment(f.prof.ID, f.group.ID, "Taller vencido", "x", "", &past, AssignmentPolicy{AllowLate: true}, nil)
	mustNoErr(t, err)
	for _, grade := range []float64{2.0, 2.5} {
		sub := f.submit(t, f.beto, KindOfficial, nil)
		g := grade
		_, err := s.SaveTeacherReview(f.prof.ID, sub.ID, "Revisa", &g, true, map[string]string{r1.ID: "bajo", r2.ID: "bajo"}, nil)
		mustNoErr(t, err)
		_, err = s.db.Exec(`UPDATE submissions SET assignment_id = NULL WHERE id = ?`, sub.ID)
		mustNoErr(t, err)
	}
	for i := 0; i < 9; i++ {
		mustNoErr(t, s.AddSubmission(&Submission{SubmissionMeta: SubmissionMeta{Kind: KindAttempt, GroupID: f.group.ID, StudentID: f.beto.ID}, Code: "int main( {" + strings.Repeat(" ", i)}))
	}
	_, err = s.db.Exec(`UPDATE group_students SET joined_at = ? WHERE student_id = ?`, fmtTime(time.Now().Add(-30*24*time.Hour)), f.ana.ID)
	mustNoErr(t, err)

	alerts, err := s.StudentAlerts(f.prof.ID, f.group.ID, time.Now())
	mustNoErr(t, err)
	if len(alerts) != 2 || alerts[0].Name != "Beto" {
		t.Fatalf("alertas: %+v", alerts)
	}
	joined := strings.Join(alerts[0].Reasons, " | ")
	for _, want := range []string{"No entregó 1 taller", "Sus últimas 9 compilaciones", "Nivel bajo en RAE 1, RAE 2", "Promedio de 2.3"} {
		if !strings.Contains(joined, want) {
			t.Errorf("a Beto le falta la razón %q: %s", want, joined)
		}
	}
	if !strings.Contains(strings.Join(alerts[1].Reasons, " "), "No ha compilado ni enviado nada") {
		t.Errorf("Ana lleva 30 días sin actividad: %+v", alerts[1])
	}
	if _, err := s.StudentAlerts(f.otro.ID, f.group.ID, time.Now()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro profesor no ve las alertas: %v", err)
	}

	rep, err := s.StudentReport(f.prof.ID, f.group.ID, f.beto.ID, time.Now())
	mustNoErr(t, err)
	if rep.Name != "Beto" || rep.Attempts != 9 || rep.Officials != 2 || len(rep.RAEs) != 2 || rep.RAEs[0].Level != "bajo" || len(rep.Alerts) == 0 || len(rep.Assignments) != 2 {
		t.Fatalf("reporte: %+v", rep)
	}
	ana1 := f.submit(t, f.ana, KindOfficial, nil)
	g45 := 4.5
	_, err = s.SaveTeacherReview(f.prof.ID, ana1.ID, "Bien", &g45, true, nil, nil)
	mustNoErr(t, err)
	f.submit(t, f.ana, KindOfficial, nil)
	anaRep, err := s.StudentReport(f.prof.ID, f.group.ID, f.ana.ID, time.Now())
	mustNoErr(t, err)
	var taller *ReportAssignment
	for i := range anaRep.Assignments {
		if anaRep.Assignments[i].Title == "Taller 2" {
			taller = &anaRep.Assignments[i]
		}
	}
	if taller == nil || taller.Submissions != 2 || taller.Grade == nil || *taller.Grade != 4.5 || anaRep.Average == nil || *anaRep.Average != 4.5 {
		t.Fatalf("una entrega nueva sin revisar no debe borrar la nota ya devuelta: %+v %+v", taller, anaRep.Average)
	}
	if _, err := s.StudentReport(f.prof.ID, f.group.ID, f.otro.ID, time.Now()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("reporte de alguien que no está en la clase: %v", err)
	}

	progress, err := s.StudentProgress(f.beto.ID)
	mustNoErr(t, err)
	if len(progress) != 1 || len(progress[0].Points) != 2 || progress[0].Average == nil || *progress[0].Average != 2.3 || progress[0].RAEs[1].Latest != "bajo" {
		t.Fatalf("progreso: %+v", progress)
	}

	mustNoErr(t, s.RecordPractice(f.ana.ID, "suma-1-n", 1, 3))
	mustNoErr(t, s.RecordPractice(f.ana.ID, "suma-1-n", 3, 3))
	mustNoErr(t, s.RecordPractice(f.ana.ID, "suma-1-n", 0, 3))
	results, err := s.PracticeResults(f.ana.ID)
	mustNoErr(t, err)
	if r := results["suma-1-n"]; r.Attempts != 3 || r.BestPassed != 3 || r.SolvedAt == nil {
		t.Fatalf("resultado de la práctica: %+v", r)
	}
}
