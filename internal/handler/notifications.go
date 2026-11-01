package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/marlon/edu-trace/internal/auth"
	"github.com/marlon/edu-trace/internal/mail"
)

const (
	mailEvery = time.Minute
	mailDelay = 2 * time.Minute
	mailMaxAge = 24 * time.Hour
	mailBatch = 100
)

type NotificationsHandler struct {
	store *auth.Store
	mailOn bool
}

func NewNotificationsHandler(store *auth.Store, mailOn bool) *NotificationsHandler {
	return &NotificationsHandler{store: store, mailOn: mailOn}
}

type notificationsResponse struct {
	*auth.NotificationList
	EmailAvailable bool `json:"email_available"`
}

func (h *NotificationsHandler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.Notifications(auth.UserFrom(r.Context()).ID)
	if err != nil {
		writeStoreErr(w, err, "cargar tus avisos")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, notificationsResponse{NotificationList: list, EmailAvailable: h.mailOn})
}

func (h *NotificationsHandler) Read(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs []int64 `json:"ids"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if len(req.IDs) > 100 {
		req.IDs = req.IDs[:100]
	}
	if err := h.store.MarkNotificationsRead(auth.UserFrom(r.Context()).ID, req.IDs); err != nil {
		writeStoreErr(w, err, "marcar los avisos como leídos")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *NotificationsHandler) SetEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := h.store.SetEmailNotifications(auth.UserFrom(r.Context()).ID, req.Enabled); err != nil {
		writeStoreErr(w, err, "guardar tu preferencia de correo")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type Mailer struct {
	store *auth.Store
	cfg mail.Config
	publicURL string
}

func NewMailer(store *auth.Store, cfg mail.Config, publicURL string) *Mailer {
	return &Mailer{store: store, cfg: cfg, publicURL: publicURL}
}

func (m *Mailer) Run(ctx context.Context) {
	if !m.cfg.Enabled() {
		slog.Info("avisos por correo desactivados: falta SMTP_HOST o SMTP_FROM")
		return
	}
	ticker := time.NewTicker(mailEvery)
	defer ticker.Stop()
	for {
		m.sendPending(time.Now())
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (m *Mailer) sendPending(now time.Time) {
	pending, err := m.store.PendingEmails(now.Add(-mailDelay), now.Add(-mailMaxAge), mailBatch)
	if err != nil {
		slog.Error("avisos por correo pendientes", "error", err)
		return
	}
	for len(pending) > 0 {
		n := 1
		for n < len(pending) && pending[n].UserID == pending[0].UserID {
			n++
		}
		batch := pending[:n]
		pending = pending[n:]
		err := m.cfg.Send(m.message(batch))
		if err != nil && !mail.Permanent(err) {
			slog.Error("enviar aviso por correo", "user_id", batch[0].UserID, "error", err)
			return
		}
		if err != nil {
			slog.Warn("aviso por correo rechazado", "user_id", batch[0].UserID, "error", err)
		}
		ids := make([]int64, len(batch))
		for i, p := range batch {
			ids[i] = p.ID
		}
		if err := m.store.MarkEmailed(ids); err != nil {
			slog.Error("marcar avisos enviados", "error", err)
			return
		}
	}
}

func (m *Mailer) message(batch []auth.PendingEmail) mail.Message {
	var b strings.Builder
	fmt.Fprintf(&b, "Hola, %s:\n\n", batch[0].Name)
	for _, p := range batch {
		fmt.Fprintf(&b, "- %s\n  %s%s\n", p.Title, m.publicURL, p.Link)
	}
	fmt.Fprintf(&b, "\nRecibes este correo porque tienes activados los avisos de EduTrace. Puedes desactivarlos en «Mi cuenta»: %s/cuenta\n", m.publicURL)
	subject := batch[0].Title
	if len(batch) > 1 {
		subject = fmt.Sprintf("EduTrace: %d avisos nuevos", len(batch))
	}
	return mail.Message{To: batch[0].Email, Subject: subject, Body: b.String()}
}
