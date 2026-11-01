package config

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration.
type Config struct {
	Host string
	Port string
	GppPath string
	Sandbox string // "bwrap" (aislamiento real) o "none" (solo desarrollo)
	DataDir string
	SessionTTL time.Duration
	TeacherCode string // código para registrar cuentas docentes; vacío = deshabilitado
	CookieSecure bool
	CompileTimeout time.Duration
	RunTimeout time.Duration
	MaxOutputSize int
	MaxSandboxes int
	OllamaBaseURL string
	OllamaModel string
	OllamaTimeout int
	OllamaContextSize int
	OllamaThink string // "auto" (según el modelo), "true" o "false": razonamiento antes de responder
	OllamaKeepAlive string // cuánto tiempo mantiene Ollama el modelo cargado en la GPU
	AllowedOrigins string
	TrustProxy bool
	AIRequestsPerHour int
	RetentionDays int
	PublicURL string
	SMTPHost string
	SMTPPort int
	SMTPUser string
	SMTPPassword string
	SMTPFrom string
	DiskMinFreePercent int
}

// Load reads configuration from environment variables with defaults.
func Load() *Config {
	return &Config{
		Host: envOrDefault("HOST", "127.0.0.1"),
		Port: envOrDefault("PORT", "8080"),
		GppPath: envOrDefault("GPP_PATH", "g++"),
		Sandbox: envOrDefault("SANDBOX", "bwrap"),
		DataDir: envOrDefault("DATA_DIR", "data"),
		SessionTTL: time.Duration(intOrDefault("SESSION_TTL_HOURS", 72)) * time.Hour,
		TeacherCode: os.Getenv("TEACHER_CODE"),
		CookieSecure: os.Getenv("COOKIE_SECURE") == "true",
		CompileTimeout: durationOrDefault("COMPILE_TIMEOUT_SECS", 10*time.Second),
		RunTimeout: durationOrDefault("RUN_TIMEOUT_SECS", 5*time.Second),
		MaxOutputSize: intOrDefault("MAX_OUTPUT_SIZE", 10240),
		MaxSandboxes: intOrDefault("MAX_SANDBOXES", runtime.NumCPU()),
		OllamaBaseURL: envOrDefault("OLLAMA_BASE_URL", "http://localhost:11434"),
		OllamaModel: envOrDefault("OLLAMA_MODEL", "llama3.1:8b"),
		OllamaTimeout: intOrDefault("OLLAMA_TIMEOUT", 180),
		OllamaContextSize: intOrDefault("OLLAMA_CONTEXT_SIZE", 6144),
		OllamaThink: envOrDefault("OLLAMA_THINK", "auto"),
		OllamaKeepAlive: envOrDefault("OLLAMA_KEEP_ALIVE", "30m"),
		AllowedOrigins: envOrDefault("ALLOWED_ORIGINS", "http://localhost:3000"),
		TrustProxy: os.Getenv("TRUST_PROXY") == "true",
		AIRequestsPerHour: intOrDefault("AI_REQUESTS_PER_HOUR", 20),
		RetentionDays: intOrDefault("RETENTION_DAYS", 180),
		PublicURL: strings.TrimRight(envOrDefault("PUBLIC_URL", firstOrigin(os.Getenv("ALLOWED_ORIGINS"))), "/"),
		SMTPHost: os.Getenv("SMTP_HOST"),
		SMTPPort: intOrDefault("SMTP_PORT", 587),
		SMTPUser: os.Getenv("SMTP_USER"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),
		SMTPFrom: os.Getenv("SMTP_FROM"),
		DiskMinFreePercent: intOrDefault("DISK_MIN_FREE_PERCENT", 10),
	}
}

func firstOrigin(origins string) string {
	first, _, _ := strings.Cut(origins, ",")
	if first = strings.TrimSpace(first); first != "" {
		return first
	}
	return "http://localhost:3000"
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
