package transcode

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
			fmt.Sprintf("-b:v:%d", i), q.VideoBitrate,
			fmt.Sprintf("-maxrate:%d", i), q.MaxRate,
			fmt.Sprintf("-bufsize:%d", i), q.BufSize,
		)
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
		"-hls_segment_filename", filepath.Join(outputDir, "%v", "seg_%03d.ts"),
		"-master_pl_name", filepath.Join(outputDir, "master.m3u8"),
		filepath.Join(outputDir, "%v", "playlist.m3u8"),
	)

	return exec.Command(cfg.FFmpegPath, args...)
}
