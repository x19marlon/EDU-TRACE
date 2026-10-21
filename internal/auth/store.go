package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite" // driver SQLite en Go puro (sin CGO)
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

//go:embed schema.sql
var schemaSQL string

// Store guarda los datos en SQLite (dir/edutrace.db). Los adjuntos de los talleres
// siguen en disco, en dir/assignments/.
type Store struct {
	db  *sql.DB
	dir string
	ttl time.Duration
}

// OpenStore abre (o crea) la base en dir/edutrace.db. Si encuentra un edutrace.json
// del formato anterior, lo importa una sola vez.
func OpenStore(dir string, ttl time.Duration) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("crear directorio de datos: %w", err)
	}
	// foreign_keys: SQLite no valida las referencias si no se activa en cada conexión.
	// WAL + busy_timeout: lecturas concurrentes y esperas en vez de "database is locked".
	// _txlock=immediate: cada transacción toma el bloqueo de escritura al empezar, así
	// dos escrituras simultáneas esperan su turno en lugar de fallar a mitad de camino.
	dsn := "file:" + filepath.Join(dir, "edutrace.db") +
		"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_txlock=immediate"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("crear esquema: %w", err)
	}
	s := &Store{db: db, dir: dir, ttl: ttl}
	if err := s.importLegacyJSON(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrar edutrace.json: %w", err)
	}
	return s, nil
}

// Close cierra la base de datos.
func (s *Store) Close() error { return s.db.Close() }

// ---------- Ayudas para SQL ----------

// Ancho fijo: así el orden alfabético del TEXT coincide con el cronológico
// (RFC3339Nano recorta ceros y desordenaría ORDER BY created_at).
const timeLayout = "2006-01-02T15:04:05.000000000Z"

func fmtTime(t time.Time) string { return t.UTC().Format(timeLayout) }

func parseTime(s string) time.Time {
	t, _ := time.Parse(timeLayout, s)
	return t
}

// fmtTimePtr devuelve NULL para una fecha ausente o en cero.
func fmtTimePtr(t *time.Time) any {
	if t == nil || t.IsZero() {
		return nil
	}
	return fmtTime(*t)
}

func timePtr(ns sql.NullString) *time.Time {
	if !ns.Valid {
		return nil
	}
	t := parseTime(ns.String)
	return &t
}

func floatPtr(n sql.NullFloat64) *float64 {
	if !n.Valid {
		return nil
	}
	v := n.Float64
	return &v
}

// floatArg convierte *float64 en un valor para SQL (NULL si es nil).
func floatArg(f *float64) any {
	if f == nil {
		return nil
	}
	return *f
}

func isUnique(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// scanner lo cumplen *sql.Row y *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

// querier lo cumplen *sql.DB y *sql.Tx: así una consulta sirve dentro y fuera de una transacción.
type querier interface {
	QueryRow(query string, args ...any) *sql.Row
	Query(query string, args ...any) (*sql.Rows, error)
	Exec(query string, args ...any) (sql.Result, error)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
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

// ---------- Usuarios y sesiones ----------

const userCols = `u.id, u.name, u.email, u.role, u.password_hash, u.created_at, u.compiles, u.feedbacks, u.last_active_at`

func scanUser(sc scanner) (*User, error) {
	var u User
	var created string
	var lastActive sql.NullString
	if err := sc.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.PasswordHash, &created,
		&u.Activity.Compiles, &u.Activity.Feedbacks, &lastActive); err != nil {
		return nil, err
	}
	u.CreatedAt = parseTime(created)
	if t := timePtr(lastActive); t != nil {
		u.Activity.LastActiveAt = *t
	}
	return &u, nil
}

// CreateUser registra una cuenta nueva. El email único lo garantiza la base (UNIQUE).
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
	u := &User{
		ID:           id,
		Name:         strings.TrimSpace(name),
		Email:        email,
		Role:         role,
		PasswordHash: hash,
		CreatedAt:    time.Now().UTC(),
	}
	_, err = s.db.Exec(`INSERT INTO users (id, name, email, role, password_hash, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`, u.ID, u.Name, u.Email, u.Role, u.PasswordHash, fmtTime(u.CreatedAt))
	if isUnique(err) {
		return nil, ErrEmailTaken
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

// Authenticate verifica email y contraseña.
func (s *Store) Authenticate(email, password string) (*User, error) {
	u, err := scanUser(s.db.QueryRow(`SELECT `+userCols+` FROM users u WHERE u.email = ?`, normalizeEmail(email)))
	if errors.Is(err, sql.ErrNoRows) {
		CheckPassword(password, dummyHash) // mismo coste que un login real
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if !CheckPassword(password, u.PasswordHash) {
		return nil, ErrInvalidCredentials
	}
	return u, nil
}

// CreateSession devuelve un token opaco nuevo para el usuario (en la base solo se guarda su hash).
func (s *Store) CreateSession(userID string) (string, time.Time, error) {
	token, err := randomHex(32)
	if err != nil {
		return "", time.Time{}, err
	}
	now := time.Now().UTC()
	expires := now.Add(s.ttl)

	// Limpieza de sesiones vencidas: no es crítica, por eso se ignora su error.
	_, _ = s.db.Exec(`DELETE FROM sessions WHERE expires_at < ?`, fmtTime(now))
	if _, err := s.db.Exec(`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES (?, ?, ?)`,
		hashToken(token), userID, fmtTime(expires)); err != nil {
		return "", time.Time{}, err
	}
	return token, expires, nil
}

// UserForSession resuelve el usuario de un token válido y no expirado.
func (s *Store) UserForSession(token string) (*User, bool) {
	if token == "" {
		return nil, false
	}
	u, err := scanUser(s.db.QueryRow(`SELECT `+userCols+`
		FROM sessions se JOIN users u ON u.id = se.user_id
		WHERE se.token_hash = ? AND se.expires_at > ?`, hashToken(token), fmtTime(time.Now())))
	if err != nil {
		return nil, false
	}
	return u, true
}

// DeleteSession invalida un token (logout).
func (s *Store) DeleteSession(token string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE token_hash = ?`, hashToken(token))
	return err
}

// RecordCompile / RecordFeedback actualizan las estadísticas de actividad.
func (s *Store) RecordCompile(userID string) {
	_, _ = s.db.Exec(`UPDATE users SET compiles = compiles + 1, last_active_at = ? WHERE id = ?`,
		fmtTime(time.Now()), userID)
}

func (s *Store) RecordFeedback(userID string) {
	_, _ = s.db.Exec(`UPDATE users SET feedbacks = feedbacks + 1, last_active_at = ? WHERE id = ?`,
		fmtTime(time.Now()), userID)
}
