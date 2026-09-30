package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/marlon/edu-trace/internal/ai"
	"github.com/marlon/edu-trace/internal/compiler"
	"github.com/marlon/edu-trace/internal/config"
	"github.com/marlon/edu-trace/internal/handler"
)

func main() {
	cfg := config.Load()

	comp := compiler.New(cfg)

	// Initialize Ollama client (non-fatal if unavailable).
	var ollama *ai.OllamaClient
	ollamaClient, err := ai.NewOllamaClient(cfg)
	if err != nil {
		slog.Warn("Ollama not available — feedback disabled", "error", err)
	} else {
		ollama = ollamaClient
	}

	router := handler.NewRouter(cfg, comp, ollama)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 200 * time.Second, // Long for Ollama responses
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown.
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		slog.Info("shutting down", "signal", sig.String())

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			slog.Error("shutdown error", "error", err)
		}
	}()

	slog.Info("EDU-TRACE server starting",
		"port", cfg.Port,
		"g++", cfg.GppPath,
		"ollama_url", cfg.OllamaBaseURL,
		"ollama_model", cfg.OllamaModel,
		"ollama_enabled", ollama != nil,
	)

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped")
}
