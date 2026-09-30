package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds runtime configuration, sourced from environment variables.
type Config struct {
	Port         string
	LogDir       string
	ViewAPIKey   string
	MaxBodyBytes int64
	PageSize     int
}

// Load reads .env (when present) and builds the Config.
func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("config: no .env file loaded, using environment")
	}

	cfg := &Config{
		Port:         getEnv("PORT", "8080"),
		LogDir:       getEnv("LOG_DIR", "logs"),
		ViewAPIKey:   getEnv("VIEW_API_KEY", ""),
		MaxBodyBytes: int64(getEnvInt("MAX_BODY_BYTES", 1<<20)),
		PageSize:     getEnvInt("PAGE_SIZE", 20),
	}

	if cfg.ViewAPIKey == "" {
		log.Println("config: VIEW_API_KEY is empty, the log viewer will reject every request")
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		log.Printf("config: %s=%q is not a number, using %d", key, v, fallback)
		return fallback
	}
	return n
}
