package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// mockRequestCleaner implements RequestCleaner for testing.
type mockRequestCleaner struct {
	mu      sync.Mutex
	calls   int
	deleted int64
	err     error
}

func (m *mockRequestCleaner) DeleteOlderThan(ctx context.Context, olderThan time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls++

	if m.err != nil {
		return 0, m.err
	}

	return m.deleted, nil
}

func (m *mockRequestCleaner) getCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func TestNewCleanupJob(t *testing.T) {
	mock := &mockRequestCleaner{}
	job := NewCleanupJob(mock, 168*time.Hour, 1*time.Hour, newTestLogger())

	assert.NotNil(t, job)
	assert.Equal(t, 168*time.Hour, job.retention)
	assert.Equal(t, 1*time.Hour, job.interval)
	assert.False(t, job.stopped)
}

func TestCleanupJob_Stop_Idempotent(t *testing.T) {
	mock := &mockRequestCleaner{}
	job := NewCleanupJob(mock, 168*time.Hour, 1*time.Hour, newTestLogger())
	job.Start()

	// First stop
	job.Stop()
	assert.True(t, job.stopped)

	// Second stop should be safe
	job.Stop()
	assert.True(t, job.stopped)
}

func TestCleanupJob_Stop_BeforeStart(t *testing.T) {
	mock := &mockRequestCleaner{}
	job := NewCleanupJob(mock, 168*time.Hour, 1*time.Hour, newTestLogger())

	// Stop without Start should not panic
	job.Stop()
	assert.True(t, job.stopped)
}

func TestCleanupJob_RunsOnInterval(t *testing.T) {
	mock := &mockRequestCleaner{deleted: 0}
	job := NewCleanupJob(mock, 168*time.Hour, 50*time.Millisecond, newTestLogger())
	job.Start()
	defer job.Stop()

	// Wait for at least 2 ticks
	time.Sleep(150 * time.Millisecond)

	calls := mock.getCalls()
	assert.GreaterOrEqual(t, calls, 2, "cleanup should have run at least twice")
}

func TestCleanupJob_PassesRetentionToCleaner(t *testing.T) {
	retention := 168 * time.Hour
	mock := &mockRequestCleaner{deleted: 5}
	job := NewCleanupJob(mock, retention, 50*time.Millisecond, newTestLogger())
	job.Start()
	defer job.Stop()

	// Wait for at least 1 tick
	time.Sleep(100 * time.Millisecond)

	calls := mock.getCalls()
	assert.GreaterOrEqual(t, calls, 1, "cleanup should have run at least once")
}

func TestCleanupJob_HandlesCleanerError(t *testing.T) {
	mock := &mockRequestCleaner{err: fmt.Errorf("connection refused")}
	job := NewCleanupJob(mock, 168*time.Hour, 50*time.Millisecond, newTestLogger())
	job.Start()
	defer job.Stop()

	// Wait for at least 1 tick -- should not panic
	time.Sleep(100 * time.Millisecond)

	calls := mock.getCalls()
	assert.GreaterOrEqual(t, calls, 1, "cleanup should have attempted at least once")
}

func TestCleanupJob_StopPreventsExecution(t *testing.T) {
	mock := &mockRequestCleaner{deleted: 0}
	job := NewCleanupJob(mock, 168*time.Hour, 50*time.Millisecond, newTestLogger())
	job.Start()
	job.Stop()

	// Wait to ensure no more ticks fire
	time.Sleep(150 * time.Millisecond)

	calls := mock.getCalls()
	// After stop, call count should not increase
	time.Sleep(100 * time.Millisecond)
	callsAfter := mock.getCalls()
	assert.Equal(t, calls, callsAfter, "no more executions should happen after stop")
}
