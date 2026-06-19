package upload

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// FFprobePath derives the ffprobe binary path from the ffmpeg path.
// If ffmpeg path is "ffmpeg", ffprobe is "ffprobe".
// If ffmpeg path is "/usr/bin/ffmpeg", ffprobe is "/usr/bin/ffprobe".
func FFprobePath(ffmpegPath string) string {
	if strings.HasSuffix(ffmpegPath, "ffmpeg") {
		return strings.TrimSuffix(ffmpegPath, "ffmpeg") + "ffprobe"
	}
	return "ffprobe"
}

// VideoDimensions holds the width and height of a video.
type VideoDimensions struct {
	Width  int
	Height int
}

// ProbeVideoDimensions uses ffprobe to get the width and height of a video file.
// Uses FFMPEG_PATH env var (default: "ffmpeg") to find ffprobe (replaces ffmpeg with ffprobe).
func ProbeVideoDimensions(ffprobePath, filePath string) (*VideoDimensions, error) {
	cmd := exec.Command(ffprobePath,
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height",
		"-of", "csv=p=0",
		filePath,
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe: %w", err)
	}

	parts := strings.Split(strings.TrimSpace(string(out)), ",")
	if len(parts) != 2 {
		return nil, fmt.Errorf("unexpected ffprobe output: %s", string(out))
	}

	w, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return nil, fmt.Errorf("parse width: %w", err)
	}
	h, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return nil, fmt.Errorf("parse height: %w", err)
	}

	return &VideoDimensions{Width: w, Height: h}, nil
}

// ProbeVideoDuration uses ffprobe to get the duration of a video file in seconds.
func ProbeVideoDuration(ffprobePath, filePath string) (float64, error) {
	cmd := exec.Command(ffprobePath,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "csv=p=0",
		filePath,
	)
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe duration: %w", err)
	}

	duration, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		return 0, fmt.Errorf("parse duration: %w", err)
	}

	return duration, nil
}

// MinSide returns the smaller of width and height.
func (d *VideoDimensions) MinSide() int {
	if d.Width < d.Height {
		return d.Width
	}
	return d.Height
}

// MaxSide returns the larger of width and height.
func (d *VideoDimensions) MaxSide() int {
	if d.Width > d.Height {
		return d.Width
	}
	return d.Height
}

// AspectRatio returns the ratio of long side to short side.
func (d *VideoDimensions) AspectRatio() float64 {
	return float64(d.MaxSide()) / float64(d.MinSide())
}

const (
	MinShortSide      = 144        // minimum short side in pixels
	MaxAspectRatio    = 4.0        // maximum width/height or height/width ratio
	MinDuration       = 1.0        // minimum video duration in seconds
	NoTranscodeSide   = 360        // short side below this → no transcoding
)
