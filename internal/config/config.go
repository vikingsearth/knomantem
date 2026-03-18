// Package config loads application configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration.
type Config struct {
	// DatabaseURL is the PostgreSQL connection string.
	DatabaseURL string

	// JWT settings.
	JWTSecret     string
	JWTExpiry     time.Duration
	RefreshExpiry time.Duration

	// Bleve full-text search index path.
	BleveIndexPath string

	// HTTP server settings.
	Port string

	// Logging.
	LogLevel string

	// CORS origins (comma-separated or "*").
	CORSOrigins string

	// Freshness worker interval.
	FreshnessInterval time.Duration

	// Rate limiter: requests per second and burst size.
	RateLimitRPS   float64
	RateLimitBurst int
}

// Load reads configuration from environment variables.
// A .env file in the working directory is loaded first (if it exists) so that
// local development does not require exporting variables.
func Load() (*Config, error) {
	// Best-effort load of .env — ignore error if file is absent.
	_ = godotenv.Load()

	cfg := &Config{}

	cfg.DatabaseURL = requireEnv("DATABASE_URL")
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("config: DATABASE_URL is required")
	}

	cfg.JWTSecret = requireEnv("JWT_SECRET")
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("config: JWT_SECRET is required")
	}

	cfg.JWTExpiry = parseDuration(getEnv("JWT_EXPIRY", "15m"))
	cfg.RefreshExpiry = parseDuration(getEnv("REFRESH_EXPIRY", "168h"))
	cfg.BleveIndexPath = getEnv("BLEVE_INDEX_PATH", "./data/search.bleve")
	cfg.Port = getEnv("PORT", "8080")
	cfg.LogLevel = getEnv("LOG_LEVEL", "info")
	cfg.CORSOrigins = getEnv("CORS_ORIGINS", "*")
	cfg.FreshnessInterval = parseDuration(getEnv("FRESHNESS_INTERVAL", "6h"))

	cfg.RateLimitRPS = parseFloat(getEnv("RATE_LIMIT_RPS", "10"), 10)
	cfg.RateLimitBurst = parseInt(getEnv("RATE_LIMIT_BURST", "30"), 30)

	return cfg, nil
}

func requireEnv(key string) string {
	return os.Getenv(key)
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0
	}
	return d
}

func parseFloat(s string, def float64) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return f
}

func parseInt(s string, def int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
