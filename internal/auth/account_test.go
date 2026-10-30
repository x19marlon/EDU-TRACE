package auth

import (
	"errors"
	"strings"
	"testing"
)

func TestPasswordAndRecovery(t *testing.T) {
	s := newTestStore(t)
	ana, err := s.CreateUser("Ana", "ana@u.edu", "password123", RoleStudent)
	mustNoErr(t, err)
	beto, err := s.CreateUser("Beto", "beto@u.edu", "password123", RoleStudent)
	mustNoErr(t, err)

	if err := s.ChangePassword(ana.ID, "mala", "nueva12345"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("cambió la contraseña con la actual equivocada: %v", err)
	}
	keep, _, err := s.CreateSession(ana.ID)
	mustNoErr(t, err)
	other, _, err := s.CreateSession(ana.ID)
	mustNoErr(t, err)
	mustNoErr(t, s.ChangePassword(ana.ID, "password123", "nueva12345"))
	mustNoErr(t, s.DeleteOtherSessions(ana.ID, keep))
	if _, ok := s.UserForSession(keep); !ok {
		t.Fatal("se cerró la sesión actual")
	}
	if _, ok := s.UserForSession(other); ok {
		t.Fatal("la otra sesión sigue abierta")
	}
	if _, err := s.Authenticate("ana@u.edu", "password123"); err == nil {
		t.Fatal("la contraseña vieja sigue sirviendo")
	}
	_, err = s.Authenticate("ana@u.edu", "nueva12345")
	mustNoErr(t, err)

	if _, _, err := s.RecoverAccount("ana@u.edu", "AAAA-AAAA-AAAA", "otra12345"); !errors.Is(err, ErrInvalidRecovery) {
		t.Fatalf("recuperó una cuenta sin código: %v", err)
	}
	code, err := s.NewRecoveryCode(ana.ID)
	mustNoErr(t, err)
	if len(code) != 14 || strings.Count(code, "-") != 2 {
		t.Fatalf("formato del código: %q", code)
	}
	if u, _ := s.Authenticate("ana@u.edu", "nueva12345"); u == nil || u.RecoveryHash == "" {
		t.Fatal("no quedó guardado el código")
	}
	if _, _, err := s.RecoverAccount("ana@u.edu", "ZZZZ-ZZZZ-ZZZZ", "otra12345"); !errors.Is(err, ErrInvalidRecovery) {
		t.Fatalf("aceptó un código equivocado: %v", err)
	}
	if _, _, err := s.RecoverAccount("nadie@u.edu", code, "otra12345"); !errors.Is(err, ErrInvalidRecovery) {
		t.Fatalf("email inexistente: %v", err)
	}
	if _, _, err := s.RecoverAccount("beto@u.edu", code, "otra12345"); !errors.Is(err, ErrInvalidRecovery) {
		t.Fatalf("el código de Ana sirvió para Beto: %v", err)
	}

	typed := strings.ToLower(strings.ReplaceAll(code, "-", " "))
	u, next, err := s.RecoverAccount(" ANA@u.edu ", typed, "recuperada1")
	mustNoErr(t, err)
	if u.ID != ana.ID || next == "" || next == code {
		t.Fatalf("recuperación: %+v nuevo código %q", u, next)
	}
	if _, ok := s.UserForSession(keep); ok {
		t.Fatal("al recuperar la cuenta deben cerrarse todas las sesiones")
	}
	_, err = s.Authenticate("ana@u.edu", "recuperada1")
	mustNoErr(t, err)
	if _, _, err := s.RecoverAccount("ana@u.edu", code, "otra12345"); !errors.Is(err, ErrInvalidRecovery) {
		t.Fatalf("el código usado sigue sirviendo: %v", err)
	}
	_, _, err = s.RecoverAccount("ana@u.edu", next, "otra12345")
	mustNoErr(t, err)
	if _, err := s.Authenticate("beto@u.edu", "password123"); err != nil {
		t.Fatalf("la cuenta de Beto cambió: %v", err)
	}
	_ = beto
}
