package handler

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/marlon/edu-trace/internal/ai"
	"github.com/marlon/edu-trace/internal/compiler"
	"github.com/marlon/edu-trace/internal/config"
)

// NewRouter builds the HTTP router with all middleware and routes.
func NewRouter(cfg *config.Config, comp *compiler.Compiler, ollama *ai.OllamaClient) http.Handler {
	mux := http.NewServeMux()

	// --- Health check ---
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// --- Compile endpoint ---
	mux.Handle("POST /api/compile", NewCompileHandler(comp))

	// --- Feedback endpoint (Ollama) ---
	if ollama != nil {
		mux.Handle("POST /api/feedback", NewFeedbackHandler(ollama, cfg.OllamaModel))
		slog.Info("feedback endpoint enabled", "model", cfg.OllamaModel)
	} else {
		mux.HandleFunc("POST /api/feedback", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"error": "Ollama no está configurado. La retroalimentación con IA no está disponible.",
			})
		})
		slog.Warn("feedback endpoint disabled — Ollama client not available")
	}

	// Apply middleware.
	var handler http.Handler = mux
	handler = corsMiddleware(handler, cfg.AllowedOrigins)
	handler = loggingMiddleware(handler)

	return handler
}

// corsMiddleware adds CORS headers for the allowed origins.
func corsMiddleware(next http.Handler, allowedOrigins string) http.Handler {
	origins := strings.Split(allowedOrigins, ",")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		for _, o := range origins {
			if strings.TrimSpace(o) == origin {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				break
			}
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs each request with slog.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start).String(),
			"remote", r.RemoteAddr,
		)
	})
}
