package handler

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/marlon/edu-trace/internal/auth"
	"github.com/marlon/edu-trace/internal/config"
	"github.com/marlon/edu-trace/internal/model"
)

// AuthHandler agrupa registro, login, logout y sesión actual.
type AuthHandler struct {
	store   *auth.Store
	cfg     *config.Config
	limiter *auth.LoginLimiter
}

func NewAuthHandler(store *auth.Store, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		store:   store,
		cfg:     cfg,
		limiter: auth.NewLoginLimiter(10, 15*time.Minute),
	}
}

type publicUser struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	Email    string        `json:"email"`
	Role     auth.Role     `json:"role"`
	Activity auth.Activity `json:"activity"`
}

func toPublic(u *auth.User) publicUser {
	return publicUser{ID: u.ID, Name: u.Name, Email: u.Email, Role: u.Role, Activity: u.Activity}
}

type registerRequest struct {
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	Password    string    `json:"password"`
	Role        auth.Role `json:"role"`
	TeacherCode string    `json:"teacher_code"`
}

type loginRequest struct {
	Email    string    `json:"email"`
	Password string    `json:"password"`
	Role     auth.Role `json:"role"`
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "cuerpo de la petición inválido"})
		return false
	}
	return true
}

func roleLabel(r auth.Role) string {
	if r == auth.RoleTeacher {
		return "profesor"
	}
	return "estudiante"
}

// Register maneja POST /api/auth/register.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	name := strings.TrimSpace(req.Name)
	if n := utf8.RuneCountInString(name); n < 2 || n > 80 {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "el nombre debe tener entre 2 y 80 caracteres"})
		return
	}
	if addr, err := mail.ParseAddress(req.Email); err != nil || addr.Address != strings.TrimSpace(req.Email) || len(req.Email) > 254 {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "email inválido"})
		return
	}
	if n := utf8.RuneCountInString(req.Password); n < 8 || n > 128 {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "la contraseña debe tener entre 8 y 128 caracteres"})
		return
	}

	switch req.Role {
	case auth.RoleStudent:
	case auth.RoleTeacher:
		// Las cuentas docentes requieren el código que define el servidor (TEACHER_CODE).
		ip := clientIP(r)
		if h.limiter.Blocked(ip) {
			writeJSON(w, http.StatusTooManyRequests, model.ErrorResponse{Error: "demasiados intentos, espera unos minutos"})
			return
		}
		if h.cfg.TeacherCode == "" ||
			subtle.ConstantTimeCompare([]byte(req.TeacherCode), []byte(h.cfg.TeacherCode)) != 1 {
			h.limiter.Fail(ip)
			writeJSON(w, http.StatusForbidden, model.ErrorResponse{Error: "código de profesor incorrecto"})
			return
		}
	default:
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "rol inválido"})
		return
	}

	u, err := h.store.CreateUser(name, req.Email, req.Password, req.Role)
	if errors.Is(err, auth.ErrEmailTaken) {
		writeJSON(w, http.StatusConflict, model.ErrorResponse{Error: err.Error()})
		return
	}
	if err != nil {
		slog.Error("register failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, model.ErrorResponse{Error: "no se pudo crear la cuenta"})
		return
	}

	slog.Info("user registered", "user_id", u.ID, "role", u.Role)
	h.startSession(w, u)
}

// Login maneja POST /api/auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	ip := clientIP(r)
	if h.limiter.Blocked(ip) {
		writeJSON(w, http.StatusTooManyRequests, model.ErrorResponse{Error: "demasiados intentos, espera unos minutos"})
		return
	}

	u, err := h.store.Authenticate(req.Email, req.Password)
	if err != nil {
		h.limiter.Fail(ip)
		writeJSON(w, http.StatusUnauthorized, model.ErrorResponse{Error: err.Error()})
		return
	}
	if req.Role != "" && req.Role != u.Role {
		writeJSON(w, http.StatusForbidden, model.ErrorResponse{
			Error: "esta cuenta es de " + roleLabel(u.Role) + ", no de " + roleLabel(req.Role),
		})
		return
	}

	h.startSession(w, u)
}

func (h *AuthHandler) startSession(w http.ResponseWriter, u *auth.User) {
	token, expires, err := h.store.CreateSession(u.ID)
	if err != nil {
		slog.Error("create session failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, model.ErrorResponse{Error: "no se pudo iniciar sesión"})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
	writeJSON(w, http.StatusOK, toPublic(u))
}

// Logout maneja POST /api/auth/logout.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(auth.CookieName); err == nil {
		if err := h.store.DeleteSession(c.Value); err != nil {
			slog.Error("delete session failed", "error", err)
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

// Me maneja GET /api/auth/me (requiere sesión).
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, toPublic(auth.UserFrom(r.Context())))
}
