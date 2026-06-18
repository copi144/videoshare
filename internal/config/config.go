package config

import (
	"os"
	"strconv"
)

// Config holds all application configuration.
type Config struct {
	Addr             string
	DataDir          string
	AdminUsername    string
	CookieSecure     bool
	FFmpegPath       string
	TranscodeWorkers int
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	cfg := &Config{
		Addr:             getEnv("PORT", ":8080"),
		DataDir:          getEnv("DATA_DIR", "./data"),
		CookieSecure:     false,
		FFmpegPath:       getEnv("FFMPEG_PATH", "ffmpeg"),
		TranscodeWorkers: getEnvInt("TRANSCODE_WORKERS", 1),
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

func getEnvInt(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return n
}
