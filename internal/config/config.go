package config

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"os"
)

// Config holds all application configuration.
type Config struct {
	Addr           string
	DataDir        string
	AdminUsername  string
	AdminPassword  string
	SessionKey     string
	CsrfKey        string
	CookieSecure   bool
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	cfg := &Config{
		Addr:    getEnv("PORT", ":8080"),
		DataDir: getEnv("DATA_DIR", "./data"),
		CsrfKey: getEnv("CSRF_KEY", ""),
		CookieSecure: false,
	}

	// AdminUsername — default to "admin".
	cfg.AdminUsername = getEnv("ADMIN_USERNAME", "admin")

	// AdminPassword — fail fast: generate a random one if not set.
	if pw := os.Getenv("ADMIN_PASSWORD"); pw != "" {
		cfg.AdminPassword = pw
	} else {
		cfg.AdminPassword = generateRandomPassword(16)
		slog.Warn("ADMIN_PASSWORD not set, generated random password", "password", cfg.AdminPassword)
	}

	// SessionKey — fail fast: generate a random key if not set.
	if sk := os.Getenv("SESSION_KEY"); sk != "" {
		cfg.SessionKey = sk
	} else {
		cfg.SessionKey = generateRandomKey()
		slog.Warn("SESSION_KEY not set, generated random key (sessions will be invalidated on restart)")
	}

	// CsrfKey — fail fast: generate a random key if not set.
	if cfg.CsrfKey == "" {
		cfg.CsrfKey = generateRandomKey()
		slog.Warn("CSRF_KEY not set, generated random key (sessions will be invalidated on restart)")
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// generateRandomKey creates a 32-byte hex-encoded random key.
func generateRandomKey() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		panic("failed to generate random key: " + err.Error())
	}
	return hex.EncodeToString(buf)
}

// generateRandomPassword creates a random alphanumeric string of the given length.
func generateRandomPassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	buf := make([]byte, length)
	randBuf := make([]byte, length)
	if _, err := rand.Read(randBuf); err != nil {
		panic("failed to generate random password: " + err.Error())
	}
	for i, b := range randBuf {
		buf[i] = charset[int(b)%len(charset)]
	}
	return string(buf)
}
