package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Role identifica el tipo de cuenta.
type Role string

const (
	RoleStudent Role = "student"
	RoleTeacher Role = "teacher"
)

var (
	ErrEmailTaken         = errors.New("ya existe una cuenta con ese email")
	ErrInvalidCredentials = errors.New("email o contraseña incorrectos")
)

// Activity acumula el uso de la plataforma por usuario (visible para docentes).
type Activity struct {
	Compiles     int       `json:"compiles"`
	Feedbacks    int       `json:"feedbacks"`
	LastActiveAt time.Time `json:"last_active_at,omitempty"`
}

// User es una cuenta registrada.
type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Role         Role      `json:"role"`
	PasswordHash string    `json:"password_hash"`
	CreatedAt    time.Time `json:"created_at"`
	Activity     Activity  `json:"activity"`
}

type session struct {
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

// storeData es lo que se persiste en disco.
type storeData struct {
	Users       map[string]*User           `json:"users"`
	Sessions    map[string]*session        `json:"sessions"` // clave: SHA-256 del token, nunca el token en claro
	Courses     map[string]*Course         `json:"courses"`
	Groups      map[string]*Group          `json:"groups"`
	Submissions map[string]*SubmissionMeta `json:"submissions"` // el código va en dir/submissions/<id>.json
}

// Store guarda usuarios y sesiones en un archivo JSON. Pensado para un curso
// (decenas o cientos de cuentas), no para miles de usuarios concurrentes.
type Store struct {
	mu   sync.Mutex
	dir  string
	path string
	ttl  time.Duration
	data storeData
}

// OpenStore carga (o crea) el almacén en dir/edutrace.json.
func OpenStore(dir string, ttl time.Duration) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("crear directorio de datos: %w", err)
	}
	s := &Store{
		dir:  dir,
		path: filepath.Join(dir, "edutrace.json"),
		ttl:  ttl,
	}
	if err := os.MkdirAll(filepath.Join(dir, "submissions"), 0o700); err != nil {
		return nil, fmt.Errorf("crear directorio de envíos: %w", err)
	}
	raw, err := os.ReadFile(s.path)
	switch {
	case errors.Is(err, os.ErrNotExist):
	case err != nil:
		return nil, err
	default:
		if err := json.Unmarshal(raw, &s.data); err != nil {
			return nil, fmt.Errorf("leer %s: %w", s.path, err)
		}
	}
	if s.data.Users == nil {
		s.data.Users = map[string]*User{}
	}
	if s.data.Sessions == nil {
		s.data.Sessions = map[string]*session{}
	}
	if s.data.Courses == nil {
		s.data.Courses = map[string]*Course{}
	}
	if s.data.Groups == nil {
		s.data.Groups = map[string]*Group{}
	}
	if s.data.Submissions == nil {
		s.data.Submissions = map[string]*SubmissionMeta{}
	}
	return s, nil
}

// saveLocked escribe el archivo de forma atómica. Requiere s.mu.
func (s *Store) saveLocked() error {
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (s *Store) findByEmailLocked(email string) *User {
	for _, u := range s.data.Users {
		if u.Email == email {
			return u
		}
	}
	return nil
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// CreateUser registra una cuenta nueva. La contraseña se hashea antes de tomar el lock.
func (s *Store) CreateUser(name, email, password string, role Role) (*User, error) {
	email = normalizeEmail(email)
	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}
	id, err := randomHex(12)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.findByEmailLocked(email) != nil {
		return nil, ErrEmailTaken
	}
	u := &User{
		ID:           id,
		Name:         strings.TrimSpace(name),
		Email:        email,
		Role:         role,
		PasswordHash: hash,
		CreatedAt:    time.Now().UTC(),
	}
	s.data.Users[id] = u
	if err := s.saveLocked(); err != nil {
		delete(s.data.Users, id)
		return nil, err
	}
	cp := *u
	return &cp, nil
}

// Authenticate verifica email y contraseña.
func (s *Store) Authenticate(email, password string) (*User, error) {
	email = normalizeEmail(email)

	s.mu.Lock()
	u := s.findByEmailLocked(email)
	var cp User
	if u != nil {
		cp = *u
	}
	s.mu.Unlock()

	if u == nil {
		CheckPassword(password, dummyHash) // mismo coste que un login real
		return nil, ErrInvalidCredentials
	}
	if !CheckPassword(password, cp.PasswordHash) {
		return nil, ErrInvalidCredentials
	}
	return &cp, nil
}

// CreateSession devuelve un token opaco nuevo para el usuario.
func (s *Store) CreateSession(userID string) (string, time.Time, error) {
	token, err := randomHex(32)
	if err != nil {
		return "", time.Time{}, err
	}
	expires := time.Now().Add(s.ttl).UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for k, sess := range s.data.Sessions {
		if now.After(sess.ExpiresAt) {
			delete(s.data.Sessions, k)
		}
	}
	s.data.Sessions[hashToken(token)] = &session{UserID: userID, ExpiresAt: expires}
	if err := s.saveLocked(); err != nil {
		return "", time.Time{}, err
	}
	return token, expires, nil
}

// UserForSession resuelve el usuario de un token válido y no expirado.
func (s *Store) UserForSession(token string) (*User, bool) {
	if token == "" {
		return nil, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.data.Sessions[hashToken(token)]
	if !ok || time.Now().After(sess.ExpiresAt) {
		return nil, false
	}
	u, ok := s.data.Users[sess.UserID]
	if !ok {
		return nil, false
	}
	cp := *u
	return &cp, true
}

// DeleteSession invalida un token (logout).
func (s *Store) DeleteSession(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data.Sessions, hashToken(token))
	return s.saveLocked()
}

// RecordCompile / RecordFeedback actualizan las estadísticas de actividad.
func (s *Store) RecordCompile(userID string) {
	s.recordActivity(userID, func(a *Activity) { a.Compiles++ })
}
func (s *Store) RecordFeedback(userID string) {
	s.recordActivity(userID, func(a *Activity) { a.Feedbacks++ })
}

func (s *Store) recordActivity(userID string, f func(*Activity)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.data.Users[userID]
	if !ok {
		return
	}
	f(&u.Activity)
	u.Activity.LastActiveAt = time.Now().UTC()
	_ = s.saveLocked()
}
