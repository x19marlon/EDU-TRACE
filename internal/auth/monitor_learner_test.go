package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMonitorStudiesInOtherSubjects(t *testing.T) {
	f := newClass(t)
	s := f.s
	mona, err := s.CreateUser("Mona", "mona@u.edu", "password123", RoleMonitor)
	mustNoErr(t, err)
	mario, err := s.CreateUser("Mario", "mario@u.edu", "password123", RoleMonitor)
	mustNoErr(t, err)
	carla, err := s.CreateUser("Carla", "carla@u.edu", "password123", RoleStudent)
	mustNoErr(t, err)
	hermana, err := s.CreateGroup(f.prof.ID, f.course.ID, "Grupo 2")
	mustNoErr(t, err)
	otraMateria, err := s.CreateCourse(f.prof.ID, "Programación II")
	mustNoErr(t, err)
	acompanada, err := s.CreateGroup(f.prof.ID, otraMateria.ID, "Grupo A")
	mustNoErr(t, err)
	acompanadaDos, err := s.CreateGroup(f.prof.ID, otraMateria.ID, "Grupo B")
	mustNoErr(t, err)

	_, err = s.JoinGroup(mona.ID, f.group.JoinCode)
	mustNoErr(t, err)
	_, err = s.AddMonitor(f.prof.ID, acompanada.ID, "mona@u.edu")
	mustNoErr(t, err)
	_, err = s.AddMonitor(f.prof.ID, f.group.ID, "mario@u.edu")
	mustNoErr(t, err)
	_, err = s.JoinGroup(carla.ID, acompanada.JoinCode)
	mustNoErr(t, err)

	for _, g := range []*Group{f.group, hermana} {
		if _, err := s.AddMonitor(f.prof.ID, g.ID, "mona@u.edu"); !errors.Is(err, ErrStudentOfCourse) {
			t.Fatalf("quedó de monitor en la materia donde estudia (%s): %v", g.Name, err)
		}
	}
	_, err = s.AddMonitor(f.prof.ID, acompanadaDos.ID, "mona@u.edu")
	mustNoErr(t, err)
	for _, g := range []*Group{acompanada, acompanadaDos} {
		if _, err := s.JoinGroup(mona.ID, g.JoinCode); !errors.Is(err, ErrMonitorOfCourse) {
			t.Fatalf("se inscribió en la materia que acompaña (%s): %v", g.Name, err)
		}
		if s.IsEnrolled(mona.ID, g.ID) {
			t.Fatalf("quedó inscrita en la materia que acompaña (%s)", g.Name)
		}
	}

	if _, err := s.AssignmentStatement(mona, f.assignment.ID); err != nil {
		t.Fatalf("no ve el taller de su clase: %v", err)
	}
	if b, err := s.StudentAIBlock(mona.ID, f.assignment.ID); err != nil || b == nil || b.Code != AIModeSupervised {
		t.Fatalf("el modo supervisado también aplica a su propio trabajo: %+v, %v", b, err)
	}

	own := f.submit(t, mona, KindOfficial, nil)
	if _, err := s.SubmissionForMonitor(mona.ID, own.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("abrió su entrega como monitor: %v", err)
	}
	if _, err := s.AddMonitorFeedback(mona.ID, own.ID, "Muy bien"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("se retroalimentó a sí misma: %v", err)
	}
	if _, err := s.AddAnnotation(mona, own.ID, AnnotationInput{Kind: AnnotationNote, Comment: "Muy bien"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("marcó su propio código: %v", err)
	}
	if mine := s.StudentSubmissions(mona.ID, 5); len(mine) != 1 || mine[0].ID != own.ID {
		t.Fatalf("su historial: %+v", mine)
	}

	_, err = s.AddAnnotation(mario, own.ID, AnnotationInput{Kind: AnnotationNote, Comment: "Revisa el ciclo"})
	mustNoErr(t, err)
	if list := s.AnnotationsFor(mona, own.ID); len(list) != 0 {
		t.Fatalf("vio una marca sin enviar: %+v", list)
	}
	_, _, err = s.SendMonitorAnnotations(mario, own.ID)
	mustNoErr(t, err)
	if list := s.AnnotationsFor(mona, own.ID); len(list) != 1 || list[0].Comment != "Revisa el ciclo" {
		t.Fatalf("no ve la marca que le enviaron: %+v", list)
	}

	classmate := f.submit(t, f.ana, KindOfficial, nil)
	if _, err := s.SubmissionForMonitor(mona.ID, classmate.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("abrió la entrega de un compañero: %v", err)
	}
	if list := s.AnnotationsFor(mona, classmate.ID); len(list) != 0 {
		t.Fatalf("vio marcas de un compañero: %+v", list)
	}

	accompanied := &Submission{SubmissionMeta: SubmissionMeta{GroupID: acompanada.ID, StudentID: carla.ID}, Code: "int main(){}\n"}
	mustNoErr(t, s.AddSubmission(accompanied))
	if _, err := s.SubmissionForMonitor(mona.ID, accompanied.ID); err != nil {
		t.Fatalf("perdió el acceso a la clase que acompaña: %v", err)
	}
	_, err = s.AddMonitorFeedback(mona.ID, accompanied.ID, "Vas bien, revisa la entrada")
	mustNoErr(t, err)
	_, err = s.AddAnnotation(mona, accompanied.ID, AnnotationInput{Kind: AnnotationNote, Comment: "Borrador"})
	mustNoErr(t, err)
	if list := s.AnnotationsFor(mona, accompanied.ID); len(list) != 1 {
		t.Fatalf("no ve sus marcas como monitor: %+v", list)
	}

	exported, err := s.ExportData(mona)
	mustNoErr(t, err)
	if len(exported.Submissions) != 1 || len(exported.MonitorFeedback) != 1 {
		t.Fatalf("su exportación debe traer sus dos papeles: %d entregas, %d comentarios", len(exported.Submissions), len(exported.MonitorFeedback))
	}
}

func TestRequireLearner(t *testing.T) {
	f := newClass(t)
	mona, err := f.s.CreateUser("Mona", "mona@u.edu", "password123", RoleMonitor)
	mustNoErr(t, err)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	for _, c := range []struct {
		user *User
		want int
	}{{f.ana, http.StatusNoContent}, {mona, http.StatusNoContent}, {f.prof, http.StatusForbidden}} {
		token, _, err := f.s.CreateSession(c.user.ID)
		mustNoErr(t, err)
		r := httptest.NewRequest("GET", "/", nil)
		r.AddCookie(&http.Cookie{Name: CookieName, Value: token})
		rec := httptest.NewRecorder()
		RequireLearner(f.s, next).ServeHTTP(rec, r)
		if rec.Code != c.want {
			t.Fatalf("%s: %d, se esperaba %d", c.user.Name, rec.Code, c.want)
		}
	}
}
