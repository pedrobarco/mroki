package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pedrobarco/mroki/cmd/mroki-api/config"
	"github.com/pedrobarco/mroki/internal/application/events"
	"github.com/pedrobarco/mroki/internal/domain/pagination"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/pedrobarco/mroki/pkg/diff"
	"github.com/pedrobarco/mroki/pkg/dto"
	"github.com/pedrobarco/mroki/pkg/ratelimit"
)

// stubHealthChecker implements handlers.HealthChecker with a configurable Ping
// result so the readiness probe can be exercised without a real database.
type stubHealthChecker struct{ err error }

func (s stubHealthChecker) Ping(context.Context) error { return s.err }

type stubGateRepository struct{}

func (stubGateRepository) Save(context.Context, *traffictesting.Gate) error { return nil }
func (stubGateRepository) Update(context.Context, *traffictesting.Gate) error { return nil }
func (stubGateRepository) Delete(context.Context, traffictesting.GateID) error { return nil }
func (stubGateRepository) GetByID(context.Context, traffictesting.GateID) (*traffictesting.Gate, error) {
	return nil, nil
}
func (stubGateRepository) GetAll(
	context.Context,
	traffictesting.GateFilters,
	traffictesting.GateSort,
	*pagination.Params,
) (*pagination.PagedResult[*traffictesting.Gate], error) {
	return &pagination.PagedResult[*traffictesting.Gate]{Items: []*traffictesting.Gate{}}, nil
}
func (stubGateRepository) ListRetentions(context.Context) ([]traffictesting.GateRetention, error) {
	return nil, nil
}

type stubRequestRepository struct{}

func (stubRequestRepository) Save(context.Context, *traffictesting.Request) error { return nil }
func (stubRequestRepository) GetByID(context.Context, traffictesting.RequestID, traffictesting.GateID) (*traffictesting.Request, error) {
	return nil, nil
}
func (stubRequestRepository) GetAllByGateID(
	context.Context,
	traffictesting.GateID,
	traffictesting.RequestFilters,
	traffictesting.RequestSort,
	*pagination.Params,
) (*pagination.PagedResult[*traffictesting.Request], error) {
	return &pagination.PagedResult[*traffictesting.Request]{Items: []*traffictesting.Request{}}, nil
}

type stubStatsRepository struct{}

func (stubStatsRepository) GetGlobalStats(context.Context) (*traffictesting.GlobalStats, error) {
	return &traffictesting.GlobalStats{}, nil
}
func (stubStatsRepository) GetStatsByGateIDs(context.Context, []traffictesting.GateID) (map[traffictesting.GateID]traffictesting.GateStats, error) {
	return map[traffictesting.GateID]traffictesting.GateStats{}, nil
}

func assertSecurityHeaders(t *testing.T, resp *http.Response) {
	t.Helper()
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))
	assert.Equal(t, "no-referrer", resp.Header.Get("Referrer-Policy"))
	assert.Empty(t, resp.Header.Get("Strict-Transport-Security"))
}

// TestNewHandler_securityHeadersOnInfraAndAPI exercises the full handler assembled
// by newHandler via httptest.NewServer: security headers on infrastructure and API
// routes (including auth failures), with HSTS off by default.
func TestNewHandler_securityHeadersOnInfraAndAPI(t *testing.T) {
	const apiKey = "test-api-key-min-16-chars"

	db, err := sql.Open("pgx", "postgres://user:pass@127.0.0.1:5432/mroki?sslmode=disable")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	platform, _, err := newAPIMetrics(true, db)
	require.NoError(t, err)
	t.Cleanup(func() { _ = platform.Shutdown(context.Background()) })

	var cfg config.Config
	cfg.App.APIKey = apiKey

	limiter := ratelimit.NewLimiter(1000)
	t.Cleanup(func() { _ = limiter.Stop() })

	handler, err := newHandler(handlerDeps{
		cfg:         cfg,
		logger:      slog.New(slog.DiscardHandler),
		gateRepo:    stubGateRepository{},
		requestRepo: stubRequestRepository{},
		statsRepo:   stubStatsRepository{},
		dispatcher:  events.NewBus(),
		limiter:     limiter,
		metrics:     platform,
		health:      stubHealthChecker{},
	})
	require.NoError(t, err)

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	for _, path := range []string{"/health/live", "/health/ready", "/metrics"} {
		t.Run("infra "+path, func(t *testing.T) {
			resp, err := http.Get(srv.URL + path)
			require.NoError(t, err)
			t.Cleanup(func() { _ = resp.Body.Close() })
			require.Equal(t, http.StatusOK, resp.StatusCode)
			assertSecurityHeaders(t, resp)
		})
	}

	t.Run("API GET /gates unauthenticated", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/gates")
		require.NoError(t, err)
		t.Cleanup(func() { _ = resp.Body.Close() })
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		assertSecurityHeaders(t, resp)
	})

	t.Run("API GET /gates authenticated", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, srv.URL+"/gates", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+apiKey)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		t.Cleanup(func() { _ = resp.Body.Close() })
		require.Equal(t, http.StatusOK, resp.StatusCode)
		assertSecurityHeaders(t, resp)

		var body dto.PaginatedResponse[[]dto.Gate]
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Empty(t, body.Data)
		assert.Equal(t, int64(0), body.Pagination.Total)
	})
}

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
// differ by diffOps operations; NewRequest raises the comparison domain event.
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
