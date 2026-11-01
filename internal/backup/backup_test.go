package backup

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"filippo.io/age"
	"github.com/marlon/edu-trace/internal/auth"
)

func seed(t *testing.T, dir string) *auth.Store {
	t.Helper()
	s, err := auth.OpenStore(dir, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	prof, err := s.CreateUser("Elena", "elena@u.edu", "password123", auth.RoleTeacher)
	if err != nil {
		t.Fatal(err)
	}
	course, err := s.CreateCourse(prof.ID, "Programación I")
	if err != nil {
		t.Fatal(err)
	}
	group, err := s.CreateGroup(prof.ID, course.ID, "Grupo 1")
	if err != nil {
		t.Fatal(err)
	}
	files := []auth.FileUpload{{Name: "enunciado.txt", Body: strings.NewReader("suma de 1 a n")}}
	if _, err := s.CreateAssignment(prof.ID, group.ID, "Taller 1", "Suma", "", nil, auth.AssignmentPolicy{AllowLate: true}, files); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestBackupAndRestore(t *testing.T) {
	data := filepath.Join(t.TempDir(), "data")
	store := seed(t, data)
	dest := filepath.Join(t.TempDir(), "respaldos")
	path, _, err := Create(data, dest, time.Date(2026, 11, 1, 2, 30, 0, 0, time.UTC), nil)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "edutrace-20261101-023000.tar.gz" {
		t.Fatalf("nombre: %s", path)
	}
	if _, err := store.CreateUser("Tarde", "tarde@u.edu", "password123", auth.RoleStudent); err != nil {
		t.Fatal(err)
	}
	store.Close()

	s, err := Verify(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.Users != 1 || s.Courses != 1 || s.Files != 1 || s.MissingFiles != 0 {
		t.Fatalf("resumen: %+v", s)
	}

	restored := filepath.Join(t.TempDir(), "restaurado")
	if err := Extract(path, restored, nil); err != nil {
		t.Fatal(err)
	}
	if err := Extract(path, restored, nil); err == nil {
		t.Fatal("no debe extraer encima de una carpeta existente")
	}
	back, err := auth.OpenStore(restored, time.Hour)
	if err != nil {
		t.Fatalf("EduTrace no abre la base restaurada: %v", err)
	}
	defer back.Close()
	if _, err := back.Authenticate("elena@u.edu", "password123"); err != nil {
		t.Fatalf("el profesor no puede entrar con la base restaurada: %v", err)
	}
	if _, err := back.Authenticate("tarde@u.edu", "password123"); err == nil {
		t.Fatal("el respaldo no debe tener lo que se escribió después")
	}
	courses := back.TeacherCourses(mustUser(t, back).ID)
	if len(courses) != 1 || len(courses[0].Groups) != 1 {
		t.Fatalf("materias restauradas: %+v", courses)
	}
}

func mustUser(t *testing.T, s *auth.Store) *auth.User {
	t.Helper()
	u, err := s.Authenticate("elena@u.edu", "password123")
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestExtractRejectsUnsafePaths(t *testing.T) {
	for _, name := range []string{"../fuera.db", "/etc/passwd", "assignments/../../fuera", "otro.txt"} {
		path := filepath.Join(t.TempDir(), "malo.tar.gz")
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		gz := gzip.NewWriter(f)
		tw := tar.NewWriter(gz)
		tw.WriteHeader(&tar.Header{Name: name, Mode: 0o600, Size: 1, Typeflag: tar.TypeReg})
		tw.Write([]byte("x"))
		tw.Close()
		gz.Close()
		f.Close()
		if err := Extract(path, filepath.Join(t.TempDir(), "x"), nil); err == nil {
			t.Errorf("aceptó %q", name)
		}
	}
}

func TestPrune(t *testing.T) {
	dir := t.TempDir()
	for _, day := range []string{"20261028", "20261029", "20261030", "20261031"} {
		os.WriteFile(filepath.Join(dir, "edutrace-"+day+"-023000.tar.gz"), []byte("x"), 0o600)
	}
	os.WriteFile(filepath.Join(dir, "otro.txt"), []byte("x"), 0o600)
	removed, err := Prune(dir, 2)
	if err != nil {
		t.Fatal(err)
	}
	left, _ := List(dir)
	if len(removed) != 2 || len(left) != 2 || !strings.Contains(left[0], "20261030") {
		t.Fatalf("borrados %v, quedan %v", removed, left)
	}
	if _, err := os.Stat(filepath.Join(dir, "otro.txt")); err != nil {
		t.Fatal("no debe tocar archivos que no son respaldos")
	}
}

func TestEncryptedBackup(t *testing.T) {
	data := filepath.Join(t.TempDir(), "data")
	seed(t, data).Close()
	id, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	recipients, err := ParseRecipients(id.Recipient().String())
	if err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	path, summary, err := Create(data, dest, time.Now(), recipients)
	if err != nil {
		t.Fatal(err)
	}
	if !Encrypted(path) || summary.Users != 1 {
		t.Fatalf("respaldo cifrado: %s %+v", path, summary)
	}
	raw, _ := os.ReadFile(path)
	if strings.Contains(string(raw), "elena@u.edu") || !strings.HasPrefix(string(raw), "age-encryption.org/v1") {
		t.Fatal("el archivo debe estar cifrado")
	}
	if _, err := Verify(path, nil); !errors.Is(err, ErrNeedIdentity) {
		t.Fatalf("sin la clave privada no se puede abrir: %v", err)
	}
	other, _ := age.GenerateX25519Identity()
	if _, err := Verify(path, []age.Identity{other}); err == nil {
		t.Fatal("con otra clave no se puede abrir")
	}
	if s, err := Verify(path, []age.Identity{id}); err != nil || s.Users != 1 || s.Files != 1 {
		t.Fatalf("con la clave correcta se verifica: %+v %v", s, err)
	}
	if list, _ := List(dest); len(list) != 1 {
		t.Fatalf("los respaldos cifrados cuentan para la rotación: %v", list)
	}
}
