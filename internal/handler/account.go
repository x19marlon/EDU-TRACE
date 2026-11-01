package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/marlon/edu-trace/internal/auth"
	"github.com/marlon/edu-trace/internal/model"
)

func validNewPassword(w http.ResponseWriter, password string) bool {
	if n := utf8.RuneCountInString(password); n < 8 || n > 128 {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "la nueva contraseña debe tener entre 8 y 128 caracteres"})
		return false
	}
	return true
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Current string `json:"current_password"`
		New string `json:"new_password"`
	}
	if !decodeJSON(w, r, &req) || !validNewPassword(w, req.New) {
		return
	}
	if req.New == req.Current {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "la nueva contraseña debe ser distinta de la actual"})
		return
	}
	ip := clientIP(r)
	if h.limiter.Blocked(ip) {
		writeJSON(w, http.StatusTooManyRequests, model.ErrorResponse{Error: "demasiados intentos, espera unos minutos"})
		return
	}
	user := auth.UserFrom(r.Context())
	err := h.store.ChangePassword(user.ID, req.Current, req.New)
	if errors.Is(err, auth.ErrInvalidCredentials) {
		h.limiter.Fail(ip)
		writeJSON(w, http.StatusForbidden, model.ErrorResponse{Error: "la contraseña actual no es correcta"})
		return
	}
	if err != nil {
		writeStoreErr(w, err, "cambiar la contraseña")
		return
	}
	if c, err := r.Cookie(auth.CookieName); err == nil {
		if err := h.store.DeleteOtherSessions(user.ID, c.Value); err != nil {
			slog.Error("delete other sessions failed", "user_id", user.ID, "error", err)
		}
	}
	slog.Info("password changed", "user_id", user.ID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) RecoveryCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	ip := clientIP(r)
	if h.limiter.Blocked(ip) {
		writeJSON(w, http.StatusTooManyRequests, model.ErrorResponse{Error: "demasiados intentos, espera unos minutos"})
		return
	}
	user := auth.UserFrom(r.Context())
	if _, err := h.store.Authenticate(user.Email, req.Password); err != nil {
		h.limiter.Fail(ip)
		writeJSON(w, http.StatusForbidden, model.ErrorResponse{Error: "la contraseña no es correcta"})
		return
	}
	code, err := h.store.NewRecoveryCode(user.ID)
	if err != nil {
		writeStoreErr(w, err, "generar el código de recuperación")
		return
	}
	slog.Info("recovery code generated", "user_id", user.ID)
	writeJSON(w, http.StatusOK, map[string]string{"recovery_code": code})
}

func (h *AuthHandler) Recover(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
		Code string `json:"recovery_code"`
		New string `json:"new_password"`
	}
	if !decodeJSON(w, r, &req) || !validNewPassword(w, req.New) {
		return
	}
	ip := clientIP(r)
	if h.limiter.Blocked(ip) {
		writeJSON(w, http.StatusTooManyRequests, model.ErrorResponse{Error: "demasiados intentos, espera unos minutos"})
		return
	}
	u, code, err := h.store.RecoverAccount(req.Email, req.Code, req.New)
	if errors.Is(err, auth.ErrInvalidRecovery) {
		h.limiter.Fail(ip)
		writeJSON(w, http.StatusUnauthorized, model.ErrorResponse{Error: err.Error()})
		return
	}
	if err != nil {
		slog.Error("account recovery failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, model.ErrorResponse{Error: "no se pudo recuperar la cuenta"})
		return
	}
	slog.Info("account recovered", "user_id", u.ID)
	h.startSession(w, u, code)
}

func (h *AuthHandler) Consent(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFrom(r.Context())
	at, err := h.store.RecordConsent(user.ID)
	if err != nil {
		writeStoreErr(w, err, "registrar tu autorización")
		return
	}
	user.ConsentAt, user.ConsentVersion = &at, auth.PrivacyVersion
	slog.Info("privacy consent recorded", "user_id", user.ID, "version", auth.PrivacyVersion)
	writeJSON(w, http.StatusOK, toPublic(user))
}

func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	name := strings.TrimSpace(req.Name)
	if n := utf8.RuneCountInString(name); n < 2 || n > 80 {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "el nombre debe tener entre 2 y 80 caracteres"})
		return
	}
	user := auth.UserFrom(r.Context())
	if err := h.store.UpdateName(user.ID, name); err != nil {
		writeStoreErr(w, err, "actualizar tu nombre")
		return
	}
	user.Name = name
	writeJSON(w, http.StatusOK, toPublic(user))
}

func (h *AuthHandler) Export(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFrom(r.Context())
	data, err := h.store.ExportData(user)
	if err != nil {
		writeStoreErr(w, err, "exportar tus datos")
		return
	}
	slog.Info("personal data exported", "user_id", user.ID)
	w.Header().Set("Content-Disposition", `attachment; filename="edutrace-mis-datos.json"`)
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, data)
}

func (h *AuthHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	ip := clientIP(r)
	if h.limiter.Blocked(ip) {
		writeJSON(w, http.StatusTooManyRequests, model.ErrorResponse{Error: "demasiados intentos, espera unos minutos"})
		return
	}
	user := auth.UserFrom(r.Context())
	err := h.store.DeleteAccount(user.ID, req.Password)
	switch {
	case errors.Is(err, auth.ErrInvalidCredentials):
		h.limiter.Fail(ip)
		writeJSON(w, http.StatusForbidden, model.ErrorResponse{Error: "la contraseña no es correcta"})
		return
	case errors.Is(err, auth.ErrOwnsCourses):
		writeJSON(w, http.StatusConflict, model.ErrorResponse{Error: err.Error()})
		return
	case err != nil:
		writeStoreErr(w, err, "eliminar tu cuenta")
		return
	}
	slog.Info("account deleted", "user_id", user.ID, "role", user.Role)
	h.Logout(w, r)
}
