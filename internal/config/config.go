package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration.
type Config struct {
	Port              string
	GppPath           string
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
		Port:              envOrDefault("PORT", "8080"),
		GppPath:           envOrDefault("GPP_PATH", "g++"),
		CompileTimeout:    durationOrDefault("COMPILE_TIMEOUT_SECS", 10*time.Second),
		RunTimeout:        durationOrDefault("RUN_TIMEOUT_SECS", 5*time.Second),
		MaxOutputSize:     intOrDefault("MAX_OUTPUT_SIZE", 10240),
		OllamaBaseURL:     envOrDefault("OLLAMA_BASE_URL", "http://localhost:11434"),
		OllamaModel:       envOrDefault("OLLAMA_MODEL", "llama3.2:latest"),
		OllamaTimeout:     intOrDefault("OLLAMA_TIMEOUT", 180),
		OllamaContextSize: intOrDefault("OLLAMA_CONTEXT_SIZE", 4096),
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
