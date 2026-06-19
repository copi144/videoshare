package transcode

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// BuildHLSCommand generates an FFmpeg command to produce HLS with multiple renditions.
// inputPath: path to the original video file
// outputDir: directory where HLS output should be written (e.g., data/video/{xx}/{yy}/{hash}/hls)
func BuildHLSCommand(cfg *TranscodeConfig, inputPath, outputDir string, qualities []Quality) *exec.Cmd {
	// Ensure output directory exists.
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		// exec.Cmd.Run will fail later anyway; we let FFmpeg surface the error.
	}

	// Create numbered subdirectories for each quality rendition.
	for i := range qualities {
		if err := os.MkdirAll(filepath.Join(outputDir, fmt.Sprintf("%d", i)), 0o755); err != nil {
			// Let FFmpeg surface the error.
		}
	}

	// Build the filter_complex string:
	//   split=N[v1][v2]...[vN];[v1]scale=w=W1:h=H1[v1out];[v2]scale=w=W2:h=H2[v2out];...
	var filterParts []string
	splitLabels := make([]string, len(qualities))
	for i := range qualities {
		splitLabels[i] = fmt.Sprintf("[v%d]", i)
	}
	filterParts = append(filterParts, fmt.Sprintf("[0:v]split=%d%s", len(qualities), strings.Join(splitLabels, "")))

	for i, q := range qualities {
		filterParts = append(filterParts, fmt.Sprintf("%sscale=w=%d:h=%d[v%dout]", splitLabels[i], q.Width, q.Height, i))
	}
	filterComplex := strings.Join(filterParts, ";")

	// Build the var_stream_map string:
	//   v:0,a:0 v:1,a:0 v:2,a:0 ...
	var streamMapParts []string
	for i := range qualities {
		streamMapParts = append(streamMapParts, fmt.Sprintf("v:%d,a:0", i))
	}
	varStreamMap := strings.Join(streamMapParts, " ")

	args := []string{
		"-i", inputPath,
		"-filter_complex", filterComplex,
	}

	// Per-video-stream encoding parameters.
	for i, q := range qualities {
		args = append(args,
			"-map", fmt.Sprintf("[v%dout]", i),
			fmt.Sprintf("-c:v:%d", i), "libx264",
			fmt.Sprintf("-preset:v:%d", i), "medium",
			fmt.Sprintf("-maxrate:%d", i), q.MaxRate,
			fmt.Sprintf("-bufsize:%d", i), q.BufSize,
		)

		if q.CRF > 0 {
			// CRF mode: constant visual quality, variable bitrate
			args = append(args, fmt.Sprintf("-crf:%d", i), strconv.Itoa(q.CRF))
		} else if q.VideoBitrate != "" {
			// Legacy bitrate mode
			args = append(args, fmt.Sprintf("-b:v:%d", i), q.VideoBitrate)
		}
	}

	// Audio encoding (single shared audio stream for all renditions).
	args = append(args,
		"-map", "0:a",
		"-c:a", "aac",
		"-ar", "48000",
		"-ac", "2",
		"-b:a", qualities[0].AudioBitrate,
	)

	// Keyframe alignment parameters.
	args = append(args,
		"-sc_threshold", "0",
		"-g", "48",
		"-keyint_min", "48",
	)

	// HLS parameters.
	args = append(args,
		"-var_stream_map", varStreamMap,
		"-hls_time", "4",
		"-hls_playlist_type", "vod",
		"-hls_flags", "independent_segments",
		"-hls_segment_filename", filepath.Join(outputDir, "%v", "%04d.ts"),
		"-master_pl_name", "master.m3u8",
		filepath.Join(outputDir, "%v.m3u8"),
	)

	return exec.Command(cfg.FFmpegPath, args...)
}

// RenameHLSOutputs renames FFmpeg's numbered HLS outputs (0, 1, 2) to resolution names (360p, 720p, 1080p)
// and updates the master playlist references accordingly.
func RenameHLSOutputs(outputDir string, qualities []Quality) error {
	for i, q := range qualities {
		// Rename playlist: 0.m3u8 → 360p.m3u8
		oldPlaylist := filepath.Join(outputDir, fmt.Sprintf("%d.m3u8", i))
		newPlaylist := filepath.Join(outputDir, fmt.Sprintf("%s.m3u8", q.Name))
		if err := os.Rename(oldPlaylist, newPlaylist); err != nil {
			return fmt.Errorf("rename playlist %s -> %s: %w", oldPlaylist, newPlaylist, err)
		}

		// Rename segment dir: 0/ -> 360p/
		oldDir := filepath.Join(outputDir, fmt.Sprintf("%d", i))
		newDir := filepath.Join(outputDir, q.Name)
		if err := os.Rename(oldDir, newDir); err != nil {
			return fmt.Errorf("rename segment dir %s -> %s: %w", oldDir, newDir, err)
		}
	}

	// Update master playlist stream references
	masterPath := filepath.Join(outputDir, "master.m3u8")
	data, err := os.ReadFile(masterPath)
	if err != nil {
		return fmt.Errorf("read master playlist: %w", err)
	}

	for i, q := range qualities {
		oldRef := fmt.Sprintf("%d.m3u8", i)
		newRef := fmt.Sprintf("%s.m3u8", q.Name)
		data = bytes.ReplaceAll(data, []byte(oldRef), []byte(newRef))
	}

	if err := os.WriteFile(masterPath, data, 0644); err != nil {
		return fmt.Errorf("write master playlist: %w", err)
	}

	return nil
}
