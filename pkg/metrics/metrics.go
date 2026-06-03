// Package metrics provides the shared Prometheus + OpenTelemetry plumbing used
// by the mroki binaries: a per-binary registry, runtime/process collector
// registration, an HTTP handler for the /metrics endpoint, and an OTel
// MeterProvider bridged onto that registry (NewMeterProvider) together with the
// otelhttp client helpers (InstrumentClient/InstrumentClientFunc) used to
// instrument outbound calls. It deliberately defines no business metrics —
// those live with the components that own them.
package metrics

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewRegistry constructs a fresh, isolated registry rather than reusing the
// global default. A per-binary registry keeps process metrics scoped to the
// binary and makes tests hermetic (no cross-test global state).
func NewRegistry() *prometheus.Registry {
	return prometheus.NewRegistry()
}

// RegisterRuntime registers the standard Go runtime, process and build-info
// collectors on the given registerer. These provide the default go_* and
// process_* metrics (goroutines, GC, memory, file descriptors, CPU, …) plus the
// go_build_info gauge. The Go collector opts into the richer runtime/metrics
// series (scheduler latency, per-size-class memory, …) on top of the defaults.
func RegisterRuntime(reg prometheus.Registerer) error {
	goCollector := collectors.NewGoCollector(
		collectors.WithGoCollectorRuntimeMetrics(collectors.MetricsAll),
	)
	if err := reg.Register(goCollector); err != nil {
		return fmt.Errorf("register go collector: %w", err)
	}
	if err := reg.Register(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{})); err != nil {
		return fmt.Errorf("register process collector: %w", err)
	}
	if err := reg.Register(collectors.NewBuildInfoCollector()); err != nil {
		return fmt.Errorf("register build info collector: %w", err)
	}
	return nil
}

// RegisterDBStats registers the standard database/sql pool collector on the
// given registerer. It exposes the go_sql_* series (open / in-use / idle
// connections, wait count and duration, idle/lifetime closures) labelled with
// db_name=name, sampled from db.Stats() at scrape time.
func RegisterDBStats(reg prometheus.Registerer, db *sql.DB, name string) error {
	if err := reg.Register(collectors.NewDBStatsCollector(db, name)); err != nil {
		return fmt.Errorf("register db stats collector: %w", err)
	}
	return nil
}

// Handler returns an http.Handler that serves the metrics gathered by the given
// gatherer in the Prometheus text exposition format. Mount it at /metrics.
func Handler(g prometheus.Gatherer) http.Handler {
	return promhttp.HandlerFor(g, promhttp.HandlerOpts{
		ErrorHandling: promhttp.ContinueOnError,
	})
}
