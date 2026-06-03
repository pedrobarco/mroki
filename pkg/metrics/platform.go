package metrics

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// Platform bundles the per-binary metrics plumbing built on a single isolated
// registry: the runtime/process/build-info collectors, an OTel MeterProvider
// bridged onto that registry, and the /metrics scrape handler. It lets the
// composition roots share one bootstrap instead of repeating the
// registry → collectors → provider → handler sequence.
//
// A nil *Platform is a no-op (its fields read as nil and Shutdown does nothing)
// so callers can wire metrics unconditionally and treat a disabled platform the
// same as an enabled one.
type Platform struct {
	Registry *prometheus.Registry
	Provider *sdkmetric.MeterProvider
	handler  http.Handler
}

// PlatformOption registers extra collectors on the platform registry during
// NewPlatform, for collectors that not every binary has (e.g. the DB pool).
type PlatformOption func(*prometheus.Registry) error

// WithDBStats registers the database/sql pool collector (go_sql_*) on the
// platform registry, for binaries that have a database pool.
func WithDBStats(db *sql.DB, name string) PlatformOption {
	return func(reg *prometheus.Registry) error {
		return RegisterDBStats(reg, db, name)
	}
}

// NewPlatform builds the metrics platform on a fresh isolated registry: it
// registers the runtime/process/build-info collectors plus any from opts,
// bridges an OTel MeterProvider onto the registry, and prepares the /metrics
// handler. Callers build their domain recorder from the returned Provider.
func NewPlatform(opts ...PlatformOption) (*Platform, error) {
	reg := NewRegistry()
	if err := RegisterRuntime(reg); err != nil {
		return nil, fmt.Errorf("register runtime metrics: %w", err)
	}
	for _, opt := range opts {
		if err := opt(reg); err != nil {
			return nil, err
		}
	}
	mp, err := NewMeterProvider(reg)
	if err != nil {
		return nil, fmt.Errorf("create meter provider: %w", err)
	}
	return &Platform{Registry: reg, Provider: mp, handler: Handler(reg)}, nil
}

// Shutdown flushes and releases the MeterProvider, and should be called during
// graceful shutdown. A nil *Platform (metrics disabled) is a no-op.
func (p *Platform) Shutdown(ctx context.Context) error {
	if p == nil || p.Provider == nil {
		return nil
	}
	return p.Provider.Shutdown(ctx)
}

// MetricsHandler returns the /metrics scrape handler, or nil when metrics are
// disabled (a nil *Platform). Callers should mount the result only when non-nil.
func (p *Platform) MetricsHandler() http.Handler {
	if p == nil {
		return nil
	}
	return p.handler
}

// InstrumentHandler wraps h with server-side otelhttp instrumentation under the
// platform MeterProvider, recording the semconv http_server_* metrics with route
// as the operation label. When metrics are disabled (a nil *Platform) it returns
// h unchanged.
func (p *Platform) InstrumentHandler(route string, h http.Handler) http.Handler {
	if p == nil || p.Provider == nil {
		return h
	}
	return InstrumentHandler(p.Provider, route, h)
}

// InstrumentClient wraps base with client-side otelhttp instrumentation under the
// platform MeterProvider, tagging outbound requests with a constant target role.
// When metrics are disabled (a nil *Platform) it returns base unchanged.
func (p *Platform) InstrumentClient(target string, base http.RoundTripper) http.RoundTripper {
	if p == nil || p.Provider == nil {
		return base
	}
	return InstrumentClient(p.Provider, target, base)
}

// InstrumentClientFunc is like InstrumentClient but resolves the target role per
// request from targetFn. When metrics are disabled (a nil *Platform) it returns
// base unchanged.
func (p *Platform) InstrumentClientFunc(targetFn func(*http.Request) string, base http.RoundTripper) http.RoundTripper {
	if p == nil || p.Provider == nil {
		return base
	}
	return InstrumentClientFunc(p.Provider, targetFn, base)
}
