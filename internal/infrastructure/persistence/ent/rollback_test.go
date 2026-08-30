package ent

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pedrobarco/mroki/ent/enttest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errOriginal stands in for the primary (non-rollback) error a caller passes to
// rollback.
var errOriginal = errors.New("original save failure")

// TestRollback_SuccessReturnsOriginal verifies that when the rollback succeeds
// the helper returns the original error unchanged, without any rollback text.
func TestRollback_SuccessReturnsOriginal(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	tx, err := client.Tx(context.Background())
	require.NoError(t, err)

	got := rollback(tx, errOriginal)

	require.ErrorIs(t, got, errOriginal)
	assert.Equal(t, errOriginal, got)
	assert.NotContains(t, got.Error(), "rollback failed")
}

// TestRollback_FailureJoinsBothErrors simulates a rollback failure by rolling
// the transaction back before invoking the helper, so the helper's own
// tx.Rollback() call returns sql.ErrTxDone. It asserts that both the original
// error (still unwrappable via errors.Is) and the rollback failure surface.
func TestRollback_FailureJoinsBothErrors(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	tx, err := client.Tx(context.Background())
	require.NoError(t, err)
	// Roll back once so the helper's rollback fails on the already-done tx.
	require.NoError(t, tx.Rollback())

	got := rollback(tx, errOriginal)

	require.Error(t, got)
	// Original error stays the primary, unwrappable error.
	assert.ErrorIs(t, got, errOriginal)
	// Rollback failure is also surfaced.
	assert.Contains(t, got.Error(), "rollback failed")
}

// TestNewRequestRepository_LoggerInjection verifies that WithLogger injects the
// provided logger and that the constructor falls back to slog.Default().
func TestNewRequestRepository_LoggerInjection(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	custom := slog.New(slog.DiscardHandler)
	withLogger := NewRequestRepository(client, WithLogger(custom))
	assert.Same(t, custom, withLogger.logger)

	// A nil logger is ignored, leaving the default in place.
	defaulted := NewRequestRepository(client, WithLogger(nil))
	assert.Same(t, slog.Default(), defaulted.logger)
}
