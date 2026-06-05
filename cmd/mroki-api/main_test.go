package main

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pedrobarco/mroki/internal/application/events"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/pedrobarco/mroki/pkg/diff"
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

// comparedRequest builds a persisted-style Request whose live/shadow responses
// differ by diffOps operations, with the comparison domain event recorded.
func comparedRequest(t *testing.T, diffOps int) *traffictesting.Request {
	t.Helper()

	method, err := traffictesting.NewHTTPMethod("GET")
	require.NoError(t, err)
	path, err := traffictesting.ParsePath("/api/test")
	require.NoError(t, err)
	live, err := traffictesting.ParseStatusCode(200)
	require.NoError(t, err)
	shadow, err := traffictesting.ParseStatusCode(500)
	require.NoError(t, err)

	ops := make([]diff.PatchOp, diffOps)
	for i := range ops {
		ops[i] = diff.PatchOp{Op: "replace", Path: "/x", Value: i}
	}
	d, err := traffictesting.NewDiff(ops, traffictesting.DiffConfig{})
	require.NoError(t, err)

	now := time.Now()
	req, err := traffictesting.NewRequest(
		traffictesting.NewGateID(), method, path, "",
		traffictesting.NewHeaders(http.Header{}), nil, now,
		traffictesting.Response{StatusCode: live, LatencyMs: 12, CreatedAt: now},
		traffictesting.Response{StatusCode: shadow, LatencyMs: 34, CreatedAt: now},
		*d,
	)
	require.NoError(t, err)
	req.RecordCompared()
	return req
}

// TestComparisonMetricsListener_RecordsOnEvent verifies the composition-root
// wiring: a RequestCompared event dispatched through the bus reaches the metrics
// listener, which records the shared business metrics onto the /metrics output.
func TestComparisonMetricsListener_RecordsOnEvent(t *testing.T) {
	db, err := sql.Open("pgx", "postgres://user:pass@127.0.0.1:5432/mroki?sslmode=disable")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	platform, recorder, err := newAPIMetrics(true, db)
	require.NoError(t, err)
	require.NotNil(t, recorder)
	t.Cleanup(func() { _ = platform.Shutdown(context.Background()) })

	bus := events.NewBus()
	bus.Subscribe(traffictesting.EventRequestCompared, newComparisonMetricsListener(recorder))

	req := comparedRequest(t, 3)
	bus.Dispatch(context.Background(), req.PullEvents()...)

	w := httptest.NewRecorder()
	platform.MetricsHandler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.Equal(t, http.StatusOK, w.Code)

	body := w.Body.String()
	gate := req.GateID.String()
	assert.Contains(t, body, `mroki_responses_compared_total{gate="`+gate+`",result="diff"}`)
	// The diff_operations histogram must carry the gate label and record one
	// observation summing to the three diff operations.
	assert.Contains(t, body, `mroki_diff_operations_count{gate="`+gate+`"} 1`)
	assert.Contains(t, body, `mroki_diff_operations_sum{gate="`+gate+`"} 3`)
}
