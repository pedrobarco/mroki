package jobs

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
)

// RequestCleaner deletes a single gate's requests older than a given duration.
// Implemented by the ent requestRepository.
type RequestCleaner interface {
	DeleteOlderThanForGate(ctx context.Context, gateID traffictesting.GateID, olderThan time.Duration) (int64, error)
}

// CleanupJob periodically deletes each gate's requests older than that gate's
// effective retention (its custom retention, or the global floor when unset).
// Responses and diffs are automatically removed via ON DELETE CASCADE.
//
// The job follows the same lifecycle pattern as pkg/ratelimit.Limiter:
// ticker + done channel + idempotent Stop().
//
// Important: Call Stop() when done to prevent goroutine leaks.
type CleanupJob struct {
	cleaner   RequestCleaner
	gates     traffictesting.GateRepository
	ticker    *time.Ticker
	done      chan struct{}
	stopped   bool
	mu        sync.Mutex
	interval  time.Duration
	retention time.Duration
	logger    *slog.Logger
}

// NewCleanupJob creates a new cleanup job.
// retention is the global retention floor (must be > 0); it applies to every
// gate that has not configured a custom retention.
// interval is how often the cleanup runs (e.g., 1h).
//
// Call Start() to begin the background goroutine.
func NewCleanupJob(cleaner RequestCleaner, gates traffictesting.GateRepository, retention, interval time.Duration, logger *slog.Logger) *CleanupJob {
	return &CleanupJob{
		cleaner:   cleaner,
		gates:     gates,
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

	j.logger.Info("cleanup job started",
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

	j.logger.Info("cleanup job stopped")
}

// run executes a single cleanup cycle, deleting each gate's expired requests
// based on that gate's effective retention.
func (j *CleanupJob) run() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()

	gates, err := j.gates.ListRetentions(ctx)
	if err != nil {
		j.logger.Error("cleanup failed to list gate retentions",
			slog.String("error", err.Error()),
			slog.Duration("elapsed", time.Since(start)),
		)
		return
	}

	// One DELETE per gate. Gates sharing an identical effective retention
	// (most use the global floor) could be batched into a single query, but the
	// per-gate loop is kept deliberately: it isolates failures so one gate's
	// error does not abort the others, and each delete already uses the
	// composite (gate_id, created_at) index efficiently.
	var totalDeleted int64
	for _, g := range gates {
		effective := g.Retention.Effective(j.retention)

		deleted, err := j.cleaner.DeleteOlderThanForGate(ctx, g.ID, effective)
		if err != nil {
			j.logger.Error("cleanup failed for gate",
				slog.String("error", err.Error()),
				slog.String("gate_id", g.ID.String()),
				slog.Duration("retention", effective),
			)
			continue
		}
		totalDeleted += deleted
	}

	duration := time.Since(start)

	if totalDeleted > 0 {
		j.logger.Info("cleanup completed",
			slog.Int64("deleted_requests", totalDeleted),
			slog.Int("gates", len(gates)),
			slog.Duration("duration", duration),
		)
	}
}
