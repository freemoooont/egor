// Package config exposes a tiny env-loader used by cmd/api. The contract:
// keep parsing centralized so handlers/middleware never reach for os.Getenv.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config carries all runtime knobs the API binary needs.
type Config struct {
	ListenAddr        string
	DatabaseURL       string
	JWTSecret         []byte
	BcryptCost        int
	MinPasswordLength int
	AccessTokenTTL    time.Duration
	RefreshTokenTTL   time.Duration
	AIAPIKey          string
	AIModel           string
	LogFormat         string
	LogLevel          string
	CORSOrigins       []string
	OutboxTickSeconds int
}

// FromEnv reads environment variables and returns a populated Config.
// Defaults match the spec: 127.0.0.1:8080, JWT_SECRET min 32 bytes, bcrypt cost 10.
func FromEnv() (Config, error) {
	c := Config{
		ListenAddr:        firstNonEmpty(os.Getenv("LISTEN_ADDR"), "127.0.0.1:8080"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		AIAPIKey:          os.Getenv("AI_API_KEY"),
		AIModel:           firstNonEmpty(os.Getenv("AI_MODEL"), "gpt-4o-mini"),
		LogFormat:         firstNonEmpty(os.Getenv("LOG_FORMAT"), "json"),
		LogLevel:          firstNonEmpty(os.Getenv("LOG_LEVEL"), "info"),
		AccessTokenTTL:    15 * time.Minute,
		RefreshTokenTTL:   7 * 24 * time.Hour,
		BcryptCost:        intFromEnv("BCRYPT_COST", 10),
		MinPasswordLength: intFromEnv("MIN_PASSWORD_LENGTH", 8),
		OutboxTickSeconds: intFromEnv("OUTBOX_TICK_SECONDS", 30),
	}
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// Generate a deterministic-ish dev secret so `go run ./cmd/api` works
		// out of the box. Production MUST set JWT_SECRET explicitly (≥32 bytes).
		secret = "dev-jwt-secret-please-override-in-production-env-file-12345"
	}
	if len(secret) < 32 {
		return Config{}, errors.New("config: JWT_SECRET must be at least 32 bytes")
	}
	c.JWTSecret = []byte(secret)

	// CORS_ORIGINS comma-separated. Defaults to the dev frontend.
	rawOrigins := firstNonEmpty(os.Getenv("CORS_ORIGINS"), "http://localhost:5173")
	for _, o := range strings.Split(rawOrigins, ",") {
		if v := strings.TrimSpace(o); v != "" {
			c.CORSOrigins = append(c.CORSOrigins, v)
		}
	}

	if c.DatabaseURL == "" {
		c.DatabaseURL = "postgres://micocards:micocards@127.0.0.1:55432/micocards?sslmode=disable"
	}

	return c, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func intFromEnv(key string, def int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return v
}

// String redacts the JWT secret for logs.
func (c Config) String() string {
	return fmt.Sprintf(
		"Config{ListenAddr=%s, DatabaseURL=%s, JWTSecret=<%d bytes>, BcryptCost=%d, AIConfigured=%t, LogFormat=%s, CORSOrigins=%v}",
		c.ListenAddr, redactURL(c.DatabaseURL), len(c.JWTSecret), c.BcryptCost, c.AIAPIKey != "", c.LogFormat, c.CORSOrigins,
	)
}

func redactURL(u string) string {
	at := strings.LastIndex(u, "@")
	if at <= 0 {
		return u
	}
	scheme := strings.Index(u, "://")
	if scheme < 0 {
		return u
	}
	return u[:scheme+3] + "***" + u[at:]
}
