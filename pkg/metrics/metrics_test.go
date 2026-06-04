package metrics

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noopHandler and noopRoundTripper are identity sentinels so the Platform tests
// can assert whether an instrumentation seam returns its input unchanged.
type noopHandler struct{}

func (noopHandler) ServeHTTP(http.ResponseWriter, *http.Request) {}

type noopRoundTripper struct{}

func (noopRoundTripper) RoundTrip(*http.Request) (*http.Response, error) { return nil, nil }

// TestPlatform_NilIsNoop pins the disabled-metrics contract: a nil *Platform
// (what newXMetrics returns when metrics are off) leaves every instrumentation
// seam a no-op, so the composition roots can wire metrics unconditionally.
func TestPlatform_NilIsNoop(t *testing.T) {
	var p *Platform // metrics disabled

	assert.Nil(t, p.MetricsHandler())
	assert.NoError(t, p.Shutdown(context.Background()))

	h := &noopHandler{}
	assert.Same(t, h, p.InstrumentHandler("route", h),
		"disabled InstrumentHandler must return the handler unchanged")

	rt := &noopRoundTripper{}
	assert.Same(t, rt, p.InstrumentClient("api", rt),
		"disabled InstrumentClient must return the transport unchanged")
	assert.Same(t, rt, p.InstrumentClientFunc(func(*http.Request) string { return "x" }, rt),
		"disabled InstrumentClientFunc must return the transport unchanged")
}

// TestPlatform_Enabled checks the enabled path: NewPlatform bridges a provider
// and handler, and each seam actually wraps its input (identity changes).
func TestPlatform_Enabled(t *testing.T) {
	p, err := NewPlatform()
	require.NoError(t, err)
	require.NotNil(t, p.Provider)
	require.NotNil(t, p.MetricsHandler())

	// otelhttp wraps the input in a new concrete type, so a changed dynamic type
	// (not pointer identity, as the wrappers may be value types) proves the seam
	// instrumented rather than passed through.
	h := &noopHandler{}
	gotH := p.InstrumentHandler("route", h)
	require.NotNil(t, gotH)
	assert.NotEqual(t, reflect.TypeOf(h), reflect.TypeOf(gotH),
		"enabled InstrumentHandler must wrap the handler")

	rt := &noopRoundTripper{}
	gotRT := p.InstrumentClient("api", rt)
	require.NotNil(t, gotRT)
	assert.NotEqual(t, reflect.TypeOf(rt), reflect.TypeOf(gotRT),
		"enabled InstrumentClient must wrap the transport")

	gotRTFn := p.InstrumentClientFunc(func(*http.Request) string { return "x" }, rt)
	require.NotNil(t, gotRTFn)
	assert.NotEqual(t, reflect.TypeOf(rt), reflect.TypeOf(gotRTFn),
		"enabled InstrumentClientFunc must wrap the transport")

	require.NoError(t, p.Shutdown(context.Background()))
}

func TestNewRegistry(t *testing.T) {
	reg := NewRegistry()
	require.NotNil(t, reg)

	// A fresh registry must be independent of the global default and of any
	// other registry created here.
	other := NewRegistry()
	assert.NotSame(t, reg, other)
}

func TestRegisterRuntime(t *testing.T) {
	reg := NewRegistry()

	require.NoError(t, RegisterRuntime(reg))

	families, err := reg.Gather()
	require.NoError(t, err)

	var hasGo, hasProcess, hasBuildInfo bool
	for _, f := range families {
		switch {
		case f.GetName() == "go_build_info":
			hasBuildInfo = true
		case strings.HasPrefix(f.GetName(), "go_"):
			hasGo = true
		case strings.HasPrefix(f.GetName(), "process_"):
			hasProcess = true
		}
	}
	assert.True(t, hasGo, "expected go_* runtime metrics to be registered")
	assert.True(t, hasProcess, "expected process_* metrics to be registered")
	assert.True(t, hasBuildInfo, "expected go_build_info to be registered")
}

func TestRegisterRuntime_DuplicateFails(t *testing.T) {
	reg := NewRegistry()

	require.NoError(t, RegisterRuntime(reg))
	// Registering the same collectors twice must surface an error rather than
	// panicking the caller.
	assert.Error(t, RegisterRuntime(reg))
}

func TestRegisterDBStats(t *testing.T) {
	reg := NewRegistry()

	// sql.Open is lazy (no connection is made), and the collector samples
	// db.Stats() at scrape time, so this needs no live database.
	db, err := sql.Open("pgx", "postgres://localhost:5432/test")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	require.NoError(t, RegisterDBStats(reg, db, "test"))

	families, err := reg.Gather()
	require.NoError(t, err)

	var hasDBStats bool
	for _, f := range families {
		if strings.HasPrefix(f.GetName(), "go_sql_") {
			hasDBStats = true
			break
		}
	}
	assert.True(t, hasDBStats, "expected go_sql_* pool metrics to be registered")
}

func TestHandler(t *testing.T) {
	reg := NewRegistry()
	require.NoError(t, RegisterRuntime(reg))

	srv := httptest.NewServer(Handler(reg))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/plain")
}
