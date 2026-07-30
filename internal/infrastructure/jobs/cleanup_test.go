package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/pedrobarco/mroki/internal/domain/pagination"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// deleteCall records a single per-gate delete invocation.
type deleteCall struct {
	gateID    traffictesting.GateID
	olderThan time.Duration
}

// mockRequestCleaner implements RequestCleaner for testing.
type mockRequestCleaner struct {
	mu      sync.Mutex
	calls   int
	perCall []deleteCall
	deleted int64
	err     error
}

func (m *mockRequestCleaner) DeleteOlderThanForGate(_ context.Context, gateID traffictesting.GateID, olderThan time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls++
	m.perCall = append(m.perCall, deleteCall{gateID: gateID, olderThan: olderThan})

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

func (m *mockRequestCleaner) getPerCall() []deleteCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]deleteCall, len(m.perCall))
	copy(out, m.perCall)
	return out
}

// mockGateLister implements traffictesting.GateRepository for testing. Only
// ListRetentions is exercised by the cleanup job; the remaining methods are
// unimplemented stubs present to satisfy the interface.
type mockGateLister struct {
	mu    sync.Mutex
	gates []traffictesting.GateRetention
	err   error
}

func (m *mockGateLister) ListRetentions(_ context.Context) ([]traffictesting.GateRetention, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, m.err
	}
	return m.gates, nil
}

func (m *mockGateLister) Save(_ context.Context, _ *traffictesting.Gate) error {
	panic("not implemented")
}

func (m *mockGateLister) Update(_ context.Context, _ *traffictesting.Gate) error {
	panic("not implemented")
}

func (m *mockGateLister) Delete(_ context.Context, _ traffictesting.GateID) error {
	panic("not implemented")
}

func (m *mockGateLister) GetByID(_ context.Context, _ traffictesting.GateID) (*traffictesting.Gate, error) {
	panic("not implemented")
}

func (m *mockGateLister) GetAll(_ context.Context, _ traffictesting.GateFilters, _ traffictesting.GateSort, _ *pagination.Params) (*pagination.PagedResult[*traffictesting.Gate], error) {
	panic("not implemented")
}

// oneGateLister returns a lister with a single gate using the given retention.
func oneGateLister(r traffictesting.Retention) *mockGateLister {
	return &mockGateLister{
		gates: []traffictesting.GateRetention{
			{ID: traffictesting.NewGateID(), Retention: r},
		},
	}
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func TestNewCleanupJob(t *testing.T) {
	mock := &mockRequestCleaner{}
	job := NewCleanupJob(mock, oneGateLister(traffictesting.NoRetention()), 168*time.Hour, 1*time.Hour, newTestLogger())

	assert.NotNil(t, job)
	assert.Equal(t, 168*time.Hour, job.retention)
	assert.Equal(t, 1*time.Hour, job.interval)
	assert.False(t, job.stopped)
}

func TestCleanupJob_Stop_Idempotent(t *testing.T) {
	mock := &mockRequestCleaner{}
	job := NewCleanupJob(mock, oneGateLister(traffictesting.NoRetention()), 168*time.Hour, 1*time.Hour, newTestLogger())
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
	job := NewCleanupJob(mock, oneGateLister(traffictesting.NoRetention()), 168*time.Hour, 1*time.Hour, newTestLogger())

	// Stop without Start should not panic
	job.Stop()
	assert.True(t, job.stopped)
}

func TestCleanupJob_RunsOnInterval(t *testing.T) {
	mock := &mockRequestCleaner{deleted: 0}
	job := NewCleanupJob(mock, oneGateLister(traffictesting.NoRetention()), 168*time.Hour, 50*time.Millisecond, newTestLogger())
	job.Start()
	defer job.Stop()

	// Wait for at least 2 ticks
	time.Sleep(150 * time.Millisecond)

	calls := mock.getCalls()
	assert.GreaterOrEqual(t, calls, 2, "cleanup should have run at least twice")
}

func TestCleanupJob_PassesGlobalRetentionToCleaner(t *testing.T) {
	global := 168 * time.Hour
	mock := &mockRequestCleaner{deleted: 5}
	job := NewCleanupJob(mock, oneGateLister(traffictesting.NoRetention()), global, 50*time.Millisecond, newTestLogger())
	job.Start()
	defer job.Stop()

	// Wait for at least 1 tick
	time.Sleep(100 * time.Millisecond)

	perCall := mock.getPerCall()
	require.NotEmpty(t, perCall, "cleanup should have run at least once")
	// A gate without a custom retention uses the global floor.
	assert.Equal(t, global, perCall[0].olderThan)
}

func TestCleanupJob_UsesCustomRetentionForGate(t *testing.T) {
	global := 168 * time.Hour
	custom, err := traffictesting.ParseRetention("1000h")
	require.NoError(t, err)

	mock := &mockRequestCleaner{deleted: 3}
	job := NewCleanupJob(mock, oneGateLister(custom), global, 50*time.Millisecond, newTestLogger())
	job.Start()
	defer job.Stop()

	time.Sleep(100 * time.Millisecond)

	perCall := mock.getPerCall()
	require.NotEmpty(t, perCall, "cleanup should have run at least once")
	assert.Equal(t, 1000*time.Hour, perCall[0].olderThan)
}

func TestCleanupJob_DeletesPerGate(t *testing.T) {
	global := 168 * time.Hour
	custom, err := traffictesting.ParseRetention("1000h")
	require.NoError(t, err)

	lister := &mockGateLister{
		gates: []traffictesting.GateRetention{
			{ID: traffictesting.NewGateID(), Retention: traffictesting.NoRetention()},
			{ID: traffictesting.NewGateID(), Retention: custom},
		},
	}
	mock := &mockRequestCleaner{deleted: 1}
	job := NewCleanupJob(mock, lister, global, 50*time.Millisecond, newTestLogger())
	job.Start()
	defer job.Stop()

	time.Sleep(100 * time.Millisecond)

	perCall := mock.getPerCall()
	require.GreaterOrEqual(t, len(perCall), 2, "each gate should be cleaned per cycle")
	// The first two calls in a cycle correspond to the two gates.
	durations := []time.Duration{perCall[0].olderThan, perCall[1].olderThan}
	assert.Contains(t, durations, global)
	assert.Contains(t, durations, 1000*time.Hour)
}

func TestCleanupJob_HandlesCleanerError(t *testing.T) {
	mock := &mockRequestCleaner{err: fmt.Errorf("connection refused")}
	job := NewCleanupJob(mock, oneGateLister(traffictesting.NoRetention()), 168*time.Hour, 50*time.Millisecond, newTestLogger())
	job.Start()
	defer job.Stop()

	// Wait for at least 1 tick -- should not panic
	time.Sleep(100 * time.Millisecond)

	calls := mock.getCalls()
	assert.GreaterOrEqual(t, calls, 1, "cleanup should have attempted at least once")
}

func TestCleanupJob_HandlesListerError(t *testing.T) {
	mock := &mockRequestCleaner{deleted: 0}
	lister := &mockGateLister{err: fmt.Errorf("db down")}
	job := NewCleanupJob(mock, lister, 168*time.Hour, 50*time.Millisecond, newTestLogger())
	job.Start()
	defer job.Stop()

	// Wait for at least 1 tick -- should not panic and should not attempt deletes.
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 0, mock.getCalls(), "no deletes should happen when listing gates fails")
}

func TestCleanupJob_StopPreventsExecution(t *testing.T) {
	mock := &mockRequestCleaner{deleted: 0}
	job := NewCleanupJob(mock, oneGateLister(traffictesting.NoRetention()), 168*time.Hour, 50*time.Millisecond, newTestLogger())
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
