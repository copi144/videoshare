package transcode

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"time"

	"videoshare/model"
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
	jobs   chan Job
	cfg    *TranscodeConfig
	store  *model.ResourceStore
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewQueue creates a transcode queue with the given worker count and resource store.
func NewQueue(cfg *TranscodeConfig, store *model.ResourceStore) *Queue {
	ctx, cancel := context.WithCancel(context.Background())
	q := &Queue{
		jobs:   make(chan Job, 100),
		cfg:    cfg,
		store:  store,
		ctx:    ctx,
		cancel: cancel,
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

// processJob runs the actual FFmpeg command with status tracking.
func (q *Queue) processJob(job Job) {
	// Check if transcode was opted-out.
	resource, checkErr := q.store.GetByID(job.ResourceID)
	if checkErr == nil && resource.NoTranscode {
		slog.Info("transcode skipped (no_transcode flag)", "resource_id", job.ResourceID)
		return
	}

	// Update status to processing.
	if err := q.store.UpdateTranscodeStatus(job.ResourceID, "processing"); err != nil {
		slog.Error("failed to update transcode status", "resource_id", job.ResourceID, "error", err)
		return
	}

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

	// Run FFmpeg with 30-minute timeout.
	ctx, cancel := context.WithTimeout(q.ctx, 30*time.Minute)
	defer cancel()

	cmd := BuildHLSCommand(q.cfg, job.InputPath, job.OutputDir, qualities)
	cmdWithCtx := exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)

	output, err := cmdWithCtx.CombinedOutput()
	if err != nil {
		slog.Error("transcode failed", "resource_id", job.ResourceID, "error", err, "output", string(output))
		// Clean up partial HLS output.
		if rmErr := os.RemoveAll(job.OutputDir); rmErr != nil {
			slog.Error("failed to clean up HLS output after failed transcode", "resource_id", job.ResourceID, "error", rmErr)
		}
		if statusErr := q.store.UpdateTranscodeStatus(job.ResourceID, "failed"); statusErr != nil {
			slog.Error("failed to update transcode status to failed", "resource_id", job.ResourceID, "error", statusErr)
		}
		return
	}

	// Rename numbered outputs (0, 1, 2) to resolution names (360p, 720p, 1080p)
	if err := RenameHLSOutputs(job.OutputDir, qualities); err != nil {
		slog.Error("failed to rename HLS outputs", "resource_id", job.ResourceID, "error", err)
		if updateErr := q.store.UpdateTranscodeStatus(job.ResourceID, "failed"); updateErr != nil {
			slog.Error("failed to update transcode status", "resource_id", job.ResourceID, "error", updateErr)
		}
		return
	}

	slog.Info("transcode completed", "resource_id", job.ResourceID)
	if err := q.store.UpdateTranscodeStatus(job.ResourceID, "done"); err != nil {
		slog.Error("failed to update transcode status", "resource_id", job.ResourceID, "error", err)
	}
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
