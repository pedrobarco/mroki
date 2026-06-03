package metrics

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/otlptranslator"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// TargetAttr is the namespaced attribute key used to tag outbound HTTP client
// metrics with a caller-supplied logical target role. It extends the standard
// semconv server_address dimension with a stable, human-readable alias derived
// 1:1 from the host, so it adds no cardinality. In the Prometheus exposition it
// surfaces as the mroki_target label.
const TargetAttr = "mroki.target"

// roundTripperFunc adapts a function to the http.RoundTripper interface.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// NewMeterProvider builds an OTel MeterProvider that exports through the given
// Prometheus registry via the otelprom bridge, so the OTel-produced HTTP and
// domain metrics are gathered on the same /metrics endpoint as the
// client_golang runtime/process/DB collectors already registered on reg.
//
// The UnderscoreEscapingWithSuffixes translation strategy is pinned explicitly
// so the exposition uses conventional Prometheus names (dots to underscores,
// the _seconds unit suffix, and _total on counters) regardless of the process
// global name-validation scheme. The scope- and target-info series are
// suppressed to keep the exposition lean.
func NewMeterProvider(reg prometheus.Registerer) (*sdkmetric.MeterProvider, error) {
	exporter, err := otelprom.New(
		otelprom.WithRegisterer(reg),
		otelprom.WithTranslationStrategy(otlptranslator.UnderscoreEscapingWithSuffixes),
		otelprom.WithoutScopeInfo(),
		otelprom.WithoutTargetInfo(),
	)
	if err != nil {
		return nil, fmt.Errorf("create prometheus exporter: %w", err)
	}
	return sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter)), nil
}

// InstrumentClient wraps base with otelhttp client instrumentation recording the
// semconv http_client_* metrics under mp, tagging every outbound request with a
// constant mroki.target attribute. Use it for clients pinned to a single
// upstream, where the target role is fixed at wrap time. A nil base falls back
// to http.DefaultTransport. It never alters or fails the request.
func InstrumentClient(mp metric.MeterProvider, target string, base http.RoundTripper) http.RoundTripper {
	return InstrumentClientFunc(mp, func(*http.Request) string { return target }, base)
}

// InstrumentClientFunc is like InstrumentClient but resolves the mroki.target
// per request from targetFn, for clients shared across upstreams where the
// target role cannot be fixed at wrap time. The target is attached via the
// otelhttp Labeler so it lands on the semconv
// http_client_request_duration_seconds histogram without a separate metric.
func InstrumentClientFunc(mp metric.MeterProvider, targetFn func(*http.Request) string, base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	transport := otelhttp.NewTransport(base, otelhttp.WithMeterProvider(mp))
	return roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		labeler, _ := otelhttp.LabelerFromContext(r.Context())
		labeler.Add(attribute.String(TargetAttr, targetFn(r)))
		ctx := otelhttp.ContextWithLabeler(r.Context(), labeler)
		return transport.RoundTrip(r.WithContext(ctx))
	})
}

// InstrumentHandler wraps h with otelhttp server instrumentation recording the
// semconv http_server_* metrics under mp. route is the otelhttp operation label;
// when h is mounted on a ServeMux the matched pattern also surfaces as the
// bounded http_route label, while a bare handler records none.
func InstrumentHandler(mp metric.MeterProvider, route string, h http.Handler) http.Handler {
	return otelhttp.NewHandler(h, route, otelhttp.WithMeterProvider(mp))
}
