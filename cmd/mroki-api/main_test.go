package main

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAPIMetrics_Disabled(t *testing.T) {
	platform, recorder, err := newAPIMetrics(false, nil)

	require.NoError(t, err)
	assert.Nil(t, platform, "no platform should be built when metrics are disabled")
	assert.Nil(t, recorder)
	// A nil platform's Shutdown is a no-op rather than a panic.
	assert.NoError(t, platform.Shutdown(context.Background()))
}

func TestNewAPIMetrics_Enabled(t *testing.T) {
	// sql.Open is lazy and never dials, so a real connection isn't required: the
	// DB-pool collector only samples db.Stats() at scrape time.
	db, err := sql.Open("pgx", "postgres://user:pass@127.0.0.1:5432/mroki?sslmode=disable")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	platform, recorder, err := newAPIMetrics(true, db)
	require.NoError(t, err)
	require.NotNil(t, platform)
	require.NotNil(t, platform.Provider)
	require.NotNil(t, platform.MetricsHandler())
	require.NotNil(t, recorder)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	platform.MetricsHandler().ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	// Runtime + DB-pool collectors registered by newAPIMetrics should be exposed.
	assert.Contains(t, body, "go_goroutines")
	assert.Contains(t, body, "go_sql_max_open_connections")

	assert.NoError(t, platform.Shutdown(context.Background()))
}
