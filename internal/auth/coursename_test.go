package auth

import "testing"

func TestCourseNameRequiresBelongingToTheClass(t *testing.T) {
	f := newClass(t)
	s := f.s
	const name = "Programación I"
	carla, err := s.CreateUser("Carla", "carla@u.edu", "password123", RoleStudent)
	mustNoErr(t, err)
	mona, err := s.CreateUser("Mona", "mona@u.edu", "password123", RoleMonitor)
	mustNoErr(t, err)
	_, err = s.AddMonitor(f.prof.ID, f.group.ID, mona.Email)
	mustNoErr(t, err)

	tests := []struct {
		who               string
		user              *User
		group, assignment string
		want              string
	}{
		{"estudiante, por clase", f.ana, f.group.ID, "", name},
		{"estudiante, por taller", f.ana, "", f.assignment.ID, name},
		{"docente", f.prof, f.group.ID, "", name},
		{"monitor", mona, f.group.ID, "", name},
		{"otro docente", f.otro, f.group.ID, "", ""},
		{"estudiante de otra clase, por clase", carla, f.group.ID, "", ""},
		{"estudiante de otra clase, por taller", carla, "", f.assignment.ID, ""},
		{"sin contexto", f.ana, "", "", ""},
		{"clase inexistente", f.ana, "no-existe", "", ""},
	}
	for _, tt := range tests {
		if got := s.CourseName(tt.user, tt.group, tt.assignment); got != tt.want {
			t.Errorf("%s: %q; quería %q", tt.who, got, tt.want)
		}
	}
	if got := s.GroupCourseName(f.group.ID); got != name {
		t.Errorf("GroupCourseName = %q", got)
	}
}
