package transcode

import (
	"os"
	"strconv"

	"videoshare/upload"
)

// Quality defines a video rendition for HLS.
type Quality struct {
	Name         string // "360p", "720p", "1080p"
	Width        int
	Height       int
	VideoBitrate string // kept for backward compat, empty string means use CRF mode
	MaxRate      string // e.g., "856k"
	BufSize      string // e.g., "1200k"
	AudioBitrate string // e.g., "96k"
	CRF          int    // H.264 CRF value (0-51, lower = better). 0 means use VideoBitrate instead
}

// DefaultQualities is the standard ABR ladder.
var DefaultQualities = []Quality{
	{Name: "360p", Width: 640, Height: 360, MaxRate: "800k", BufSize: "1200k", AudioBitrate: "64k", CRF: 23},
	{Name: "720p", Width: 1280, Height: 720, MaxRate: "1500k", BufSize: "2250k", AudioBitrate: "96k", CRF: 23},
	{Name: "1080p", Width: 1920, Height: 1080, MaxRate: "2000k", BufSize: "3000k", AudioBitrate: "96k", CRF: 23},
}

// FilterQualitiesByInput filters the quality ladder based on input video dimensions.
// A quality is included only if the input video is large enough to support it.
func FilterQualitiesByInput(qualities []Quality, dims *upload.VideoDimensions) []Quality {
	if dims == nil {
		return qualities
	}
	longSide := dims.MaxSide()
	shortSide := dims.MinSide()

	var filtered []Quality
	for _, q := range qualities {
		// Only include a quality if the input is large enough for it.
		// For 1080p: requires long_side >= 1920 OR short_side >= 1080
		// For 720p: requires long_side >= 1280 OR short_side >= 720
		// For 360p: requires long_side >= 640 OR short_side >= 360
		if longSide >= q.Width || shortSide >= q.Height {
			filtered = append(filtered, q)
		}
	}
	return filtered
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
