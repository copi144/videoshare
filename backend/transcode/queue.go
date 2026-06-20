package transcode

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"videoshare/model"
	"videoshare/storage"
	"videoshare/upload"
)

// Job represents a transcoding job.
type Job struct {
	ResourceID string
	InputPath  string
	OutputDir  string
}

// Queue manages a pool of transcoding workers.
type Queue struct {
	jobs    chan Job
	cfg     *TranscodeConfig
	store   *model.ResourceStore
	dataDir string
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// NewQueue creates a transcode queue with the given worker count and resource store.
func NewQueue(cfg *TranscodeConfig, store *model.ResourceStore, dataDir string) *Queue {
	ctx, cancel := context.WithCancel(context.Background())
	q := &Queue{
		jobs:    make(chan Job, 100),
		cfg:     cfg,
		store:   store,
		dataDir: dataDir,
		ctx:     ctx,
		cancel:  cancel,
	}
	for i := 0; i < cfg.Workers; i++ {
		q.wg.Add(1)
		go q.worker(i)
	}
	return q
}

// Submit adds a job to the queue (non-blocking).
func (q *Queue) Submit(job Job) {
	select {
	case q.jobs <- job:
		slog.Info("transcode job submitted", "resource_id", job.ResourceID)
	default:
		slog.Warn("transcode queue full, dropping job", "resource_id", job.ResourceID)
	}
}

// worker processes jobs from the queue.
func (q *Queue) worker(id int) {
	defer q.wg.Done()
	slog.Info("transcode worker started", "worker_id", id)
	for {
		select {
		case job, ok := <-q.jobs:
			if !ok {
				return
			}
			slog.Info("transcode worker processing job", "worker_id", id, "resource_id", job.ResourceID)
			q.processJob(job)
		case <-q.ctx.Done():
			return
		}
	}
}

// processJob dispatches processing based on resource type.
func (q *Queue) processJob(job Job) {
	// Look up the resource to determine its type and transcode opt-out.
	resource, err := q.store.GetByID(job.ResourceID)
	if err != nil {
		slog.Error("failed to lookup resource for transcode", "resource_id", job.ResourceID, "error", err)
		return
	}

	// Check if transcode was opted-out.
	if resource.NoTranscode {
		slog.Info("transcode skipped (no_transcode flag)", "resource_id", job.ResourceID)
		return
	}

	// Update status to processing.
	if err := q.store.UpdateTranscodeStatus(job.ResourceID, "processing"); err != nil {
		slog.Error("failed to update transcode status", "resource_id", job.ResourceID, "error", err)
		return
	}

	switch resource.ResourceType {
	case storage.ResourceTypeVideo:
		q.processVideoJob(job)
	case storage.ResourceTypeAudio:
		q.processAudioJob(job)
	case storage.ResourceTypeImage:
		q.processImageJob(job)
	default:
		slog.Warn("unknown resource type, skipping transcode", "resource_id", job.ResourceID, "type", resource.ResourceType)
		if err := q.store.UpdateTranscodeStatus(job.ResourceID, "done"); err != nil {
			slog.Error("failed to mark transcode done", "resource_id", job.ResourceID, "error", err)
		}
	}
}

func (q *Queue) processVideoJob(job Job) {
	// Probe input video dimensions to filter quality ladder.
	dims, probeErr := upload.ProbeVideoDimensions(upload.FFprobePath(q.cfg.FFmpegPath), job.InputPath)
	var qualities []Quality
	if probeErr != nil {
		slog.Warn("failed to probe video dimensions, using full quality ladder", "resource_id", job.ResourceID, "error", probeErr)
		qualities = DefaultQualities
	} else {
		qualities = FilterQualitiesByInput(DefaultQualities, dims)
		slog.Info("adaptive quality ladder", "resource_id", job.ResourceID, "width", dims.Width, "height", dims.Height, "renditions", len(qualities))
	}

	// Use data/tmp/{ResourceID}/hls/ as temp output directory.
	tmpDir := filepath.Join(q.dataDir, "tmp", job.ResourceID)
	tmpHLSDir := filepath.Join(tmpDir, "hls")
	finalHLSDir := storage.HLSPath(q.dataDir, job.ResourceID)

	if err := os.MkdirAll(tmpHLSDir, 0o755); err != nil {
		slog.Error("failed to create temp HLS directory", "resource_id", job.ResourceID, "error", err)
		if statusErr := q.store.UpdateTranscodeStatus(job.ResourceID, "failed"); statusErr != nil {
			slog.Error("failed to update transcode status", "resource_id", job.ResourceID, "error", statusErr)
		}
		return
	}

	// Run FFmpeg with 30-minute timeout.
	ctx, cancel := context.WithTimeout(q.ctx, 30*time.Minute)
	defer cancel()

	cmd := BuildHLSCommand(q.cfg, job.InputPath, tmpHLSDir, qualities)
	cmdWithCtx := exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)

	output, err := cmdWithCtx.CombinedOutput()
	if err != nil {
		slog.Error("transcode failed", "resource_id", job.ResourceID, "error", err, "output", string(output))
		os.RemoveAll(tmpDir)
		if statusErr := q.store.UpdateTranscodeStatus(job.ResourceID, "failed"); statusErr != nil {
			slog.Error("failed to update transcode status to failed", "resource_id", job.ResourceID, "error", statusErr)
		}
		return
	}

	// Rename numbered outputs (0, 1, 2) to resolution names (360p, 720p, 1080p)
	if err := RenameHLSOutputs(tmpHLSDir, qualities); err != nil {
		slog.Error("failed to rename HLS outputs", "resource_id", job.ResourceID, "error", err)
		os.RemoveAll(tmpDir)
		if updateErr := q.store.UpdateTranscodeStatus(job.ResourceID, "failed"); updateErr != nil {
			slog.Error("failed to update transcode status", "resource_id", job.ResourceID, "error", updateErr)
		}
		return
	}

	// Move HLS directory to final location.
	os.RemoveAll(finalHLSDir)
	if err := os.Rename(tmpHLSDir, finalHLSDir); err != nil {
		slog.Error("failed to move HLS output to final location", "resource_id", job.ResourceID, "error", err)
		os.RemoveAll(tmpDir)
		if updateErr := q.store.UpdateTranscodeStatus(job.ResourceID, "failed"); updateErr != nil {
			slog.Error("failed to update transcode status", "resource_id", job.ResourceID, "error", updateErr)
		}
		return
	}

	// Clean up temp dir.
	os.RemoveAll(tmpDir)

	slog.Info("transcode completed", "resource_id", job.ResourceID)
	if err := q.store.UpdateTranscodeStatus(job.ResourceID, "done"); err != nil {
		slog.Error("failed to update transcode status", "resource_id", job.ResourceID, "error", err)
	}
}

func (q *Queue) processAudioJob(job Job) {
	tmpDir := filepath.Join(q.dataDir, "tmp", job.ResourceID)
	tmpOutputPath := filepath.Join(tmpDir, "transcoded", "output.mp3")
	finalOutputPath := storage.AudioOutputPath(q.dataDir, job.ResourceID)

	if err := os.MkdirAll(filepath.Dir(tmpOutputPath), 0o755); err != nil {
		slog.Error("failed to create temp audio output dir", "resource_id", job.ResourceID, "error", err)
		q.store.UpdateTranscodeStatus(job.ResourceID, "failed")
		return
	}

	ctx, cancel := context.WithTimeout(q.ctx, 10*time.Minute)
	defer cancel()
	cmd := BuildAudioCommand(q.cfg, job.InputPath, tmpOutputPath)
	cmdWithCtx := exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)
	output, err := cmdWithCtx.CombinedOutput()
	if err != nil {
		slog.Error("audio transcode failed", "resource_id", job.ResourceID, "error", err, "output", string(output))
		os.RemoveAll(tmpDir)
		q.store.UpdateTranscodeStatus(job.ResourceID, "failed")
		return
	}

	// Move to final location.
	if err := os.MkdirAll(filepath.Dir(finalOutputPath), 0o755); err != nil {
		slog.Error("failed to create final audio output dir", "resource_id", job.ResourceID, "error", err)
		os.RemoveAll(tmpDir)
		q.store.UpdateTranscodeStatus(job.ResourceID, "failed")
		return
	}
	os.Remove(finalOutputPath) // remove old transcode output if any
	if err := os.Rename(tmpOutputPath, finalOutputPath); err != nil {
		slog.Error("failed to move audio output", "resource_id", job.ResourceID, "error", err)
		os.RemoveAll(tmpDir)
		q.store.UpdateTranscodeStatus(job.ResourceID, "failed")
		return
	}

	os.RemoveAll(tmpDir)
	slog.Info("audio transcode completed", "resource_id", job.ResourceID)
	q.store.UpdateTranscodeStatus(job.ResourceID, "done")
}

func (q *Queue) processImageJob(job Job) {
	tmpDir := filepath.Join(q.dataDir, "tmp", job.ResourceID)
	tmpThumbPath := filepath.Join(tmpDir, "thumb.jpg")
	finalThumbPath := filepath.Join(storage.HashPath(q.dataDir, storage.ResourceTypeImage, job.ResourceID), "thumb.jpg")

	if err := os.MkdirAll(filepath.Dir(tmpThumbPath), 0o755); err != nil {
		slog.Error("failed to create temp image output dir", "resource_id", job.ResourceID, "error", err)
		q.store.UpdateTranscodeStatus(job.ResourceID, "failed")
		return
	}

	ctx, cancel := context.WithTimeout(q.ctx, 30*time.Second)
	defer cancel()
	cmd := BuildThumbnailCommand(q.cfg, job.InputPath, tmpThumbPath, 800)
	cmdWithCtx := exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)
	output, err := cmdWithCtx.CombinedOutput()
	if err != nil {
		slog.Error("image thumbnail failed", "resource_id", job.ResourceID, "error", err, "output", string(output))
		os.RemoveAll(tmpDir)
		q.store.UpdateTranscodeStatus(job.ResourceID, "failed")
		return
	}

	// Move to final location.
	if err := os.MkdirAll(filepath.Dir(finalThumbPath), 0o755); err != nil {
		slog.Error("failed to create final image output dir", "resource_id", job.ResourceID, "error", err)
		os.RemoveAll(tmpDir)
		q.store.UpdateTranscodeStatus(job.ResourceID, "failed")
		return
	}
	os.Remove(finalThumbPath)
	if err := os.Rename(tmpThumbPath, finalThumbPath); err != nil {
		slog.Error("failed to move image thumbnail", "resource_id", job.ResourceID, "error", err)
		os.RemoveAll(tmpDir)
		q.store.UpdateTranscodeStatus(job.ResourceID, "failed")
		return
	}

	os.RemoveAll(tmpDir)
	slog.Info("image thumbnail completed", "resource_id", job.ResourceID)
	q.store.UpdateTranscodeStatus(job.ResourceID, "done")
}

// Shutdown gracefully stops all workers.
func (q *Queue) Shutdown() {
	close(q.jobs)
	q.wg.Wait()
	q.cancel()
	slog.Info("transcode queue shut down")
}

// String returns a debug-friendly description of the queue.
func (q *Queue) String() string {
	return fmt.Sprintf("Queue{workers=%d, buffer=%d}", q.cfg.Workers, cap(q.jobs))
}

// StartupRecovery resets stalled 'processing' jobs back to 'pending' on server restart.
func StartupRecovery(store *model.ResourceStore) {
	stalled, err := store.ListByTranscodeStatus("processing")
	if err != nil {
		slog.Error("failed to scan for stalled transcodes", "error", err)
		return
	}
	for _, r := range stalled {
		if err := store.UpdateTranscodeStatus(r.ID, "pending"); err != nil {
			slog.Error("failed to reset stalled transcode", "id", r.ID, "error", err)
		} else {
			slog.Info("reset stalled transcode to pending", "id", r.ID)
		}
	}
}
