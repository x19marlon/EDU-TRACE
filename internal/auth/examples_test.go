package auth

import (
	"errors"
	"sort"
	"strings"
	"testing"
)

func exampleIDs(t *testing.T, s *Store, teacherID, groupID string) string {
	t.Helper()
	ids, err := s.GroupExampleIDs(teacherID, groupID)
	mustNoErr(t, err)
	sort.Strings(ids)
	return strings.Join(ids, ",")
}

func TestGroupExamplesSelection(t *testing.T) {
	f := newClass(t)
	s := f.s

	if got := exampleIDs(t, s, f.prof.ID, f.group.ID); got != "" {
		t.Fatalf("una clase nueva no debe tener ejemplos: %q", got)
	}
	mustNoErr(t, s.SetGroupExamples(f.prof.ID, f.group.ID, []string{"ciclo-for", "if-else", "ciclo-for"}, true))
	mustNoErr(t, s.SetGroupExamples(f.prof.ID, f.group.ID, []string{"if-else"}, true))
	if got := exampleIDs(t, s, f.prof.ID, f.group.ID); got != "ciclo-for,if-else" {
		t.Fatalf("agregar debe ser idempotente: %q", got)
	}
	mustNoErr(t, s.SetGroupExamples(f.prof.ID, f.group.ID, []string{"if-else", "no-activo"}, false))
	if got := exampleIDs(t, s, f.prof.ID, f.group.ID); got != "ciclo-for" {
		t.Fatalf("quitar: %q", got)
	}

	if _, err := s.GroupExampleIDs(f.otro.ID, f.group.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro docente no debe ver los ejemplos: %v", err)
	}
	if err := s.SetGroupExamples(f.otro.ID, f.group.ID, []string{"if-else"}, true); !errors.Is(err, ErrNotFound) {
		t.Fatalf("otro docente no debe cambiar los ejemplos: %v", err)
	}
	if got := exampleIDs(t, s, f.prof.ID, f.group.ID); got != "ciclo-for" {
		t.Fatalf("el intento de otro docente cambió la clase: %q", got)
	}
}

func TestLearnerSeesOnlyExamplesOfTheirClasses(t *testing.T) {
	f := newClass(t)
	s := f.s
	mustNoErr(t, s.SetGroupExamples(f.prof.ID, f.group.ID, []string{"ciclo-for"}, true))

	otherCourse, err := s.CreateCourse(f.otro.ID, "Otra materia")
	mustNoErr(t, err)
	otherGroup, err := s.CreateGroup(f.otro.ID, otherCourse.ID, "Grupo ajeno")
	mustNoErr(t, err)
	mustNoErr(t, s.SetGroupExamples(f.otro.ID, otherGroup.ID, []string{"if-else"}, true))

	groups, err := s.LearnerExamples(f.ana.ID)
	mustNoErr(t, err)
	if len(groups) != 1 || groups[0].ID != f.group.ID || strings.Join(groups[0].ExampleIDs, ",") != "ciclo-for" {
		t.Fatalf("clases de Ana: %+v", groups)
	}

	// Una clase sin ejemplos también aparece, para poder avisarlo al estudiante.
	mustNoErr(t, s.SetGroupExamples(f.prof.ID, f.group.ID, []string{"ciclo-for"}, false))
	groups, err = s.LearnerExamples(f.ana.ID)
	mustNoErr(t, err)
	if len(groups) != 1 || len(groups[0].ExampleIDs) != 0 {
		t.Fatalf("clase sin ejemplos: %+v", groups)
	}

	carla, err := s.CreateUser("Carla", "carla@u.edu", "password123", RoleStudent)
	mustNoErr(t, err)
	groups, err = s.LearnerExamples(carla.ID)
	mustNoErr(t, err)
	if len(groups) != 0 {
		t.Fatalf("sin clases no debe ver nada: %+v", groups)
	}
}

func TestMonitorSeesExamplesOfTheClassTheyAccompany(t *testing.T) {
	f := newClass(t)
	s := f.s
	mustNoErr(t, s.SetGroupExamples(f.prof.ID, f.group.ID, []string{"ciclo-for"}, true))
	monitor, err := s.CreateUser("Mona", "mona@u.edu", "password123", RoleMonitor)
	mustNoErr(t, err)
	_, err = s.AddMonitor(f.prof.ID, f.group.ID, monitor.Email)
	mustNoErr(t, err)
	groups, err := s.LearnerExamples(monitor.ID)
	mustNoErr(t, err)
	if len(groups) != 1 || strings.Join(groups[0].ExampleIDs, ",") != "ciclo-for" {
		t.Fatalf("clases del monitor: %+v", groups)
	}
}

func TestCloneCourseKeepsExamples(t *testing.T) {
	f := newClass(t)
	s := f.s
	mustNoErr(t, s.SetGroupExamples(f.prof.ID, f.group.ID, []string{"ciclo-for", "if-else"}, true))
	newID, err := s.CloneCourse(f.prof.ID, f.course.ID, "Programación I (copia)", false)
	mustNoErr(t, err)
	var newGroup string
	mustNoErr(t, s.db.QueryRow(`SELECT id FROM class_groups WHERE course_id = ?`, newID).Scan(&newGroup))
	if got := exampleIDs(t, s, f.prof.ID, newGroup); got != "ciclo-for,if-else" {
		t.Fatalf("ejemplos de la copia: %q", got)
	}
}

func TestDeletingGroupRemovesItsExamples(t *testing.T) {
	f := newClass(t)
	s := f.s
	mustNoErr(t, s.SetGroupExamples(f.prof.ID, f.group.ID, []string{"ciclo-for"}, true))
	mustNoErr(t, s.DeleteGroup(f.prof.ID, f.group.ID))
	var n int
	mustNoErr(t, s.db.QueryRow(`SELECT COUNT(*) FROM group_examples WHERE group_id = ?`, f.group.ID).Scan(&n))
	if n != 0 {
		t.Fatalf("quedaron %d ejemplos de una clase borrada", n)
	}
}
