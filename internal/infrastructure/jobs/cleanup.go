package jobs

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// RequestCleaner deletes requests older than a given duration.
// Implemented by postgres.requestRepository.
type RequestCleaner interface {
	DeleteOlderThan(ctx context.Context, olderThan time.Duration) (int64, error)
}

// CleanupJob periodically deletes requests older than the configured retention period.
// Responses and diffs are automatically removed via ON DELETE CASCADE.
//
// The job follows the same lifecycle pattern as pkg/ratelimit.Limiter:
// ticker + done channel + idempotent Stop().
//
// Important: Call Stop() when done to prevent goroutine leaks.
type CleanupJob struct {
	cleaner   RequestCleaner
	ticker    *time.Ticker
	done      chan struct{}
	stopped   bool
	mu        sync.Mutex
	interval  time.Duration
	retention time.Duration
	logger    *slog.Logger
}

// NewCleanupJob creates a new cleanup job.
// retention is the maximum age of requests to keep (e.g., 168h for 7 days).
// interval is how often the cleanup runs (e.g., 1h).
//
// Call Start() to begin the background goroutine.
func NewCleanupJob(cleaner RequestCleaner, retention, interval time.Duration, logger *slog.Logger) *CleanupJob {
	return &CleanupJob{
		cleaner:   cleaner,
		done:      make(chan struct{}),
		interval:  interval,
		retention: retention,
		logger:    logger,
	}
}

// Start begins the background cleanup goroutine.
func (j *CleanupJob) Start() {
	j.ticker = time.NewTicker(j.interval)

	go func() {
		for {
			select {
			case <-j.ticker.C:
				j.run()
			case <-j.done:
				return
			}
		}
	}()

	j.logger.Info("Cleanup job started",
		slog.Duration("retention", j.retention),
		slog.Duration("interval", j.interval),
	)
}

// Stop gracefully stops the cleanup goroutine.
// It is safe to call Stop multiple times.
func (j *CleanupJob) Stop() {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.stopped {
		return
	}

	j.stopped = true
	close(j.done)
	if j.ticker != nil {
		j.ticker.Stop()
	}

	j.logger.Info("Cleanup job stopped")
}

// run executes a single cleanup cycle.
func (j *CleanupJob) run() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()

	deleted, err := j.cleaner.DeleteOlderThan(ctx, j.retention)
	if err != nil {
		j.logger.Error("Cleanup failed",
			slog.String("error", err.Error()),
			slog.Duration("retention", j.retention),
			slog.Duration("elapsed", time.Since(start)),
		)
		return
	}

	duration := time.Since(start)

	if deleted > 0 {
		j.logger.Info("Cleanup completed",
			slog.Int64("deleted_requests", deleted),
			slog.Duration("duration", duration),
		)
	}
}
