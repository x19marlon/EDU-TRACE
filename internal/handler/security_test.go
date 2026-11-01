package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/marlon/edu-trace/internal/auth"
)

func TestSecurityHeaders(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("{}")) })
	for _, https := range []bool{false, true} {
		rec := httptest.NewRecorder()
		securityHeaders(ok, https).ServeHTTP(rec, httptest.NewRequest("GET", "/api/health", nil))
		h := rec.Header()
		if h.Get("X-Content-Type-Options") != "nosniff" || h.Get("X-Frame-Options") != "DENY" ||
			h.Get("Cache-Control") != "no-store" || h.Get("Content-Security-Policy") == "" {
			t.Fatalf("cabeceras: %v", h)
		}
		if (h.Get("Strict-Transport-Security") != "") != https {
			t.Fatalf("HSTS solo con HTTPS (https=%v): %v", https, h)
		}
	}
}

func TestProxyMiddleware(t *testing.T) {
	var seen string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { seen = clientIP(r) })
	cases := []struct {
		trust bool
		remote, fwd, want string
	}{
		{true, "127.0.0.1:5000", "6.6.6.6, 200.1.2.3", "200.1.2.3"},
		{true, "[::1]:5000", "200.1.2.3", "200.1.2.3"},
		{true, "10.0.0.9:5000", "200.1.2.3", "10.0.0.9"},
		{true, "127.0.0.1:5000", "basura", "127.0.0.1"},
		{false, "127.0.0.1:5000", "200.1.2.3", "127.0.0.1"},
	}
	for _, c := range cases {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = c.remote
		r.Header.Set("X-Forwarded-For", c.fwd)
		proxyMiddleware(next, c.trust).ServeHTTP(httptest.NewRecorder(), r)
		if seen != c.want {
			t.Errorf("%+v: IP %q", c, seen)
		}
	}
}

func TestAIRequestLimit(t *testing.T) {
	store, err := auth.OpenStore(t.TempDir(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	student, err := store.CreateUser("Ana", "ana@u.edu", "password123", auth.RoleStudent)
	if err != nil {
		t.Fatal(err)
	}
	limit := auth.NewLoginLimiter(2, time.Hour)
	for i := 0; i < 2; i++ {
		if !aiAllowed(httptest.NewRecorder(), store, limit, student, "") {
			t.Fatalf("pedido %d rechazado", i+1)
		}
	}
	rec := httptest.NewRecorder()
	if aiAllowed(rec, store, limit, student, "") {
		t.Fatal("el tercer pedido de la hora debía rechazarse")
	}
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("respuesta: %d %v %s", rec.Code, rec.Header(), rec.Body)
	}
	teacher, err := store.CreateUser("Elena", "elena@u.edu", "password123", auth.RoleTeacher)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		if !aiAllowed(httptest.NewRecorder(), store, limit, teacher, "") {
			t.Fatal("el límite es solo para estudiantes")
		}
	}
}
