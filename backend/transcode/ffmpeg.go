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
	//   [0:v]scale=w=W1:h=H1[v0out];[0:v]scale=w=W2:h=H2[v1out];...
	//   [0:a]asplit=N[a0][a1]...
	// Each scale and asplit output gets its own stream, which avoids
	// FFmpeg 7.x "Same elementary stream found more than once" error.
	var filterParts []string
	for i, q := range qualities {
		filterParts = append(filterParts, fmt.Sprintf("[0:v]scale=w=%d:h=%d[v%dout]", q.Width, q.Height, i))
	}
	if len(qualities) > 1 {
		// Create N copies of the audio stream so each variant has its own.
		labels := make([]string, len(qualities))
		for i := range qualities {
			labels[i] = fmt.Sprintf("[a%dout]", i)
		}
		filterParts = append(filterParts, fmt.Sprintf("[0:a]asplit=%d%s", len(qualities), strings.Join(labels, "")))
	} else {
		filterParts = append(filterParts, "[0:a]acopy[a0out]")
	}
	filterComplex := strings.Join(filterParts, ";")

	// Build the var_stream_map string:
	//   v:0,a:0 v:1,a:1 v:2,a:2 ...
	var streamMapParts []string
	for i := range qualities {
		streamMapParts = append(streamMapParts, fmt.Sprintf("v:%d,a:%d", i, i))
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

	// Audio encoding (separate audio stream per rendition).
	for i, q := range qualities {
		args = append(args,
			"-map", fmt.Sprintf("[a%dout]", i),
			fmt.Sprintf("-c:a:%d", i), "aac",
			fmt.Sprintf("-ar:a:%d", i), "48000",
			fmt.Sprintf("-ac:a:%d", i), "2",
			fmt.Sprintf("-b:a:%d", i), q.AudioBitrate,
		)
	}

	// Keyframe alignment parameters.
	args = append(args,
		"-sc_threshold", "0",
		"-g", "48",
		"-keyint_min", "48",
	)

	// HLS parameters.
	args = append(args,
		"-var_stream_map", varStreamMap,
		"-hls_time", "6",
		"-hls_playlist_type", "vod",
		"-hls_flags", "independent_segments",
		"-hls_segment_filename", filepath.Join(outputDir, "%v", "%04d.ts"),
		"-master_pl_name", "master.m3u8",
		filepath.Join(outputDir, "%v.m3u8"),
	)

	return exec.Command(cfg.FFmpegPath, args...)
}

// RenameHLSOutputs renames FFmpeg's numbered HLS outputs (0, 1, 2) to resolution names (360p, 720p, 1080p)
// and updates the variant and master playlist references accordingly.
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

	// Update variant playlist segment references (e.g., "0/0000.ts" → "360p/0000.ts")
	for i, q := range qualities {
		playlistPath := filepath.Join(outputDir, fmt.Sprintf("%s.m3u8", q.Name))
		data, err := os.ReadFile(playlistPath)
		if err != nil {
			return fmt.Errorf("read variant playlist %s: %w", playlistPath, err)
		}
		// Replace the old numbered directory prefix in segment paths
		oldPrefix := fmt.Sprintf("%d/", i)
		newPrefix := fmt.Sprintf("%s/", q.Name)
		data = bytes.ReplaceAll(data, []byte(oldPrefix), []byte(newPrefix))
		if err := os.WriteFile(playlistPath, data, 0644); err != nil {
			return fmt.Errorf("write variant playlist %s: %w", playlistPath, err)
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
