package config

import (
	"os"
)

// Config holds all application configuration.
type Config struct {
	Addr          string
	DataDir       string
	AdminUsername string
	CookieSecure  bool
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	cfg := &Config{
		Addr:         getEnv("PORT", ":8080"),
		DataDir:      getEnv("DATA_DIR", "./data"),
		CookieSecure: false,
	}

	cfg.AdminUsername = getEnv("ADMIN_USERNAME", "admin")

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
