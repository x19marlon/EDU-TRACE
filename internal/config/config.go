package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration.
type Config struct {
	Host              string
	Port              string
	GppPath           string
	Sandbox           string // "bwrap" (aislamiento real) o "none" (solo desarrollo)
	DataDir           string
	SessionTTL        time.Duration
	TeacherCode       string // código para registrar cuentas docentes; vacío = deshabilitado
	CookieSecure      bool
	CompileTimeout    time.Duration
	RunTimeout        time.Duration
	MaxOutputSize     int
	OllamaBaseURL     string
	OllamaModel       string
	OllamaTimeout     int
	OllamaContextSize int
	AllowedOrigins    string
}

// Load reads configuration from environment variables with defaults.
func Load() *Config {
	return &Config{
		Host:              envOrDefault("HOST", "127.0.0.1"),
		Port:              envOrDefault("PORT", "8080"),
		GppPath:           envOrDefault("GPP_PATH", "g++"),
		Sandbox:           envOrDefault("SANDBOX", "bwrap"),
		DataDir:           envOrDefault("DATA_DIR", "data"),
		SessionTTL:        time.Duration(intOrDefault("SESSION_TTL_HOURS", 72)) * time.Hour,
		TeacherCode:       os.Getenv("TEACHER_CODE"),
		CookieSecure:      os.Getenv("COOKIE_SECURE") == "true",
		CompileTimeout:    durationOrDefault("COMPILE_TIMEOUT_SECS", 10*time.Second),
		RunTimeout:        durationOrDefault("RUN_TIMEOUT_SECS", 5*time.Second),
		MaxOutputSize:     intOrDefault("MAX_OUTPUT_SIZE", 10240),
		OllamaBaseURL:     envOrDefault("OLLAMA_BASE_URL", "http://localhost:11434"),
		OllamaModel:       envOrDefault("OLLAMA_MODEL", "llama3.2:latest"),
		OllamaTimeout:     intOrDefault("OLLAMA_TIMEOUT", 180),
		OllamaContextSize: intOrDefault("OLLAMA_CONTEXT_SIZE", 8192),
		AllowedOrigins:    envOrDefault("ALLOWED_ORIGINS", "http://localhost:3000"),
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func intOrDefault(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func durationOrDefault(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return time.Duration(n) * time.Second
		}
	}
	return fallback
}
