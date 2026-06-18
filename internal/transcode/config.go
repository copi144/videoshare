package transcode

import (
	"os"
	"strconv"
)

// Quality defines a video rendition for HLS.
type Quality struct {
	Name         string // "360p", "720p", "1080p"
	Width        int
	Height       int
	VideoBitrate string // e.g., "800k"
	MaxRate      string // e.g., "856k"
	BufSize      string // e.g., "1200k"
	AudioBitrate string // e.g., "96k"
}

// DefaultQualities is the standard ABR ladder.
var DefaultQualities = []Quality{
	{Name: "360p", Width: 640, Height: 360, VideoBitrate: "800k", MaxRate: "856k", BufSize: "1200k", AudioBitrate: "96k"},
	{Name: "720p", Width: 1280, Height: 720, VideoBitrate: "2800k", MaxRate: "2996k", BufSize: "4200k", AudioBitrate: "128k"},
	{Name: "1080p", Width: 1920, Height: 1080, VideoBitrate: "5000k", MaxRate: "5350k", BufSize: "7500k", AudioBitrate: "192k"},
}

// TranscodeConfig holds transcoding configuration.
type TranscodeConfig struct {
	FFmpegPath string
	Workers    int
}

// LoadTranscodeConfig reads transcoding config from environment.
func LoadTranscodeConfig() *TranscodeConfig {
	return &TranscodeConfig{
		FFmpegPath: getEnv("FFMPEG_PATH", "ffmpeg"),
		Workers:    getEnvInt("TRANSCODE_WORKERS", 1),
	}
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
