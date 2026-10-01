package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// CookieName es la cookie HttpOnly que guarda el token de sesión.
const CookieName = "edutrace_session"

type ctxKey struct{}

// UserFrom devuelve el usuario autenticado de la petición (nil si no hay).
func UserFrom(ctx context.Context) *User {
	u, _ := ctx.Value(ctxKey{}).(*User)
	return u
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// RequireAuth rechaza con 401 las peticiones sin una sesión válida.
func RequireAuth(store *Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(CookieName)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "no has iniciado sesión")
			return
		}
		u, ok := store.UserForSession(c.Value)
		if !ok {
			writeErr(w, http.StatusUnauthorized, "la sesión expiró, vuelve a iniciar sesión")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, u)))
	})
}

// RequireRole exige sesión válida y el rol indicado (403 si no coincide).
func RequireRole(store *Store, role Role, next http.Handler) http.Handler {
	return RequireAuth(store, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if UserFrom(r.Context()).Role != role {
			writeErr(w, http.StatusForbidden, "no tienes permiso para esta sección")
			return
		}
		next.ServeHTTP(w, r)
	}))
}

// SingleFlight permite una sola operación en curso por usuario (p. ej. una
// compilación o una retroalimentación a la vez), para que nadie acapare CPU/GPU.
type SingleFlight struct {
	mu     sync.Mutex
	active map[string]bool
}

func NewSingleFlight() *SingleFlight {
	return &SingleFlight{active: map[string]bool{}}
}

// Wrap devuelve 429 si el usuario ya tiene una petición en curso.
func (sf *SingleFlight) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := UserFrom(r.Context()).ID
		sf.mu.Lock()
		if sf.active[id] {
			sf.mu.Unlock()
			writeErr(w, http.StatusTooManyRequests, "ya tienes una petición en curso, espera a que termine")
			return
		}
		sf.active[id] = true
		sf.mu.Unlock()

		defer func() {
			sf.mu.Lock()
			delete(sf.active, id)
			sf.mu.Unlock()
		}()
		next.ServeHTTP(w, r)
	})
}

// LoginLimiter bloquea temporalmente una IP tras demasiados intentos fallidos.
type LoginLimiter struct {
	mu       sync.Mutex
	failures map[string][]time.Time
	max      int
	window   time.Duration
}

func NewLoginLimiter(max int, window time.Duration) *LoginLimiter {
	return &LoginLimiter{failures: map[string][]time.Time{}, max: max, window: window}
}

func (l *LoginLimiter) recentLocked(key string) []time.Time {
	cutoff := time.Now().Add(-l.window)
	kept := l.failures[key][:0]
	for _, t := range l.failures[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) == 0 {
		delete(l.failures, key)
		return nil
	}
	l.failures[key] = kept
	return kept
}

// Blocked indica si la clave (IP) superó el máximo de fallos en la ventana.
func (l *LoginLimiter) Blocked(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.recentLocked(key)) >= l.max
}

// Fail registra un intento fallido.
func (l *LoginLimiter) Fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.failures[key] = append(l.recentLocked(key), time.Now())
}
