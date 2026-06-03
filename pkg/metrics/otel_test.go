package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func gather(t *testing.T, reg *prometheus.Registry, name string) *dto.MetricFamily {
	t.Helper()
	fams, err := reg.Gather()
	require.NoError(t, err)
	for _, f := range fams {
		if f.GetName() == name {
			return f
		}
	}
	t.Fatalf("metric family %q not found", name)
	return nil
}

func hasLabel(m *dto.Metric, name, value string) bool {
	for _, l := range m.GetLabel() {
		if l.GetName() == name && l.GetValue() == value {
			return true
		}
	}
	return false
}

func TestInstrumentClientFunc_RecordsTargetLabel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	reg := prometheus.NewRegistry()
	mp, err := NewMeterProvider(reg)
	require.NoError(t, err)

	client := &http.Client{
		Transport: InstrumentClientFunc(mp, func(*http.Request) string { return "live" }, nil),
	}
	resp, err := client.Get(srv.URL)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	fam := gather(t, reg, "http_client_request_duration_seconds")
	var found bool
	for _, m := range fam.GetMetric() {
		if hasLabel(m, "mroki_target", "live") {
			found = true
			assert.Equal(t, uint64(1), m.GetHistogram().GetSampleCount())
		}
	}
	assert.True(t, found, "expected an http_client_request_duration_seconds sample with mroki_target=live")
}

func TestInstrumentClient_ConstantTarget(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	reg := prometheus.NewRegistry()
	mp, err := NewMeterProvider(reg)
	require.NoError(t, err)

	client := &http.Client{Transport: InstrumentClient(mp, "api", nil)}
	resp, err := client.Get(srv.URL)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	fam := gather(t, reg, "http_client_request_duration_seconds")
	var found bool
	for _, m := range fam.GetMetric() {
		if hasLabel(m, "mroki_target", "api") {
			found = true
		}
	}
	assert.True(t, found, "expected an http_client_request_duration_seconds sample with mroki_target=api")
}

// labelValue returns the value of the named label on m, or "" if it is absent.
func labelValue(m *dto.Metric, name string) string {
	for _, l := range m.GetLabel() {
		if l.GetName() == name {
			return l.GetValue()
		}
	}
	return ""
}

// TestServerHandler_HTTPRouteLabel pins the deliberate cardinality decision:
// the API serves routes through a ServeMux so otelhttp derives a bounded
// http_route label from the templated pattern, while the proxy is served bare so
// r.Pattern stays empty and no http_route label is emitted for unbounded paths.
func TestServerHandler_HTTPRouteLabel(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	t.Run("present with ServeMux pattern (API)", func(t *testing.T) {
		reg := prometheus.NewRegistry()
		mp, err := NewMeterProvider(reg)
		require.NoError(t, err)

		mux := http.NewServeMux()
		mux.Handle("GET /gates/{gate_id}", otelhttp.NewHandler(
			ok, "GET /gates/{gate_id}", otelhttp.WithMeterProvider(mp),
		))
		srv := httptest.NewServer(mux)
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/gates/abc123")
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())

		fam := gather(t, reg, "http_server_request_duration_seconds")
		var found bool
		for _, m := range fam.GetMetric() {
			if hasLabel(m, "http_route", "/gates/{gate_id}") {
				found = true
			}
		}
		assert.True(t, found, "expected http_route=/gates/{gate_id} on the API server histogram")
	})

	t.Run("absent without ServeMux (proxy)", func(t *testing.T) {
		reg := prometheus.NewRegistry()
		mp, err := NewMeterProvider(reg)
		require.NoError(t, err)

		// Served bare, exactly like the proxy's main listener: no ServeMux, so
		// r.Pattern is empty and otelhttp records no route.
		srv := httptest.NewServer(otelhttp.NewHandler(ok, "proxy", otelhttp.WithMeterProvider(mp)))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/anything/unbounded/path")
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())

		fam := gather(t, reg, "http_server_request_duration_seconds")
		require.NotEmpty(t, fam.GetMetric())
		for _, m := range fam.GetMetric() {
			assert.Empty(t, labelValue(m, "http_route"),
				"proxy must not set http_route, to keep unbounded paths out of metric labels")
		}
	})
}

// benchRoundTripper is a no-network base transport so the client benchmark
// isolates the otelhttp instrumentation overhead from real socket I/O.
type benchRoundTripper struct{}

func (benchRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Header: make(http.Header)}, nil
}

// BenchmarkInstrumentClient measures the per-request cost of the outbound client
// instrumentation with metrics disabled (the bare base transport) versus enabled
// (wrapped with otelhttp plus the mroki.target Labeler). The base transport does
// no I/O so the delta is the instrumentation overhead alone.
func BenchmarkInstrumentClient(b *testing.B) {
	base := benchRoundTripper{}
	req, err := http.NewRequest(http.MethodGet, "http://live.example/resource", nil)
	require.NoError(b, err)

	b.Run("disabled", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			resp, _ := base.RoundTrip(req)
			_ = resp.Body.Close()
		}
	})

	b.Run("enabled", func(b *testing.B) {
		reg := prometheus.NewRegistry()
		mp, err := NewMeterProvider(reg)
		require.NoError(b, err)
		rt := InstrumentClientFunc(mp, func(*http.Request) string { return "live" }, base)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			resp, _ := rt.RoundTrip(req)
			_ = resp.Body.Close()
		}
	})
}

// BenchmarkServerHandler measures the per-request cost of the inbound server
// instrumentation with metrics disabled (the bare handler) versus enabled
// (wrapped with otelhttp). A ResponseRecorder is allocated each iteration in
// both arms, so the delta isolates the otelhttp overhead.
func BenchmarkServerHandler(b *testing.B) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/resource", nil)

	b.Run("disabled", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			ok.ServeHTTP(httptest.NewRecorder(), req)
		}
	})

	b.Run("enabled", func(b *testing.B) {
		reg := prometheus.NewRegistry()
		mp, err := NewMeterProvider(reg)
		require.NoError(b, err)
		h := otelhttp.NewHandler(ok, "bench", otelhttp.WithMeterProvider(mp))
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			h.ServeHTTP(httptest.NewRecorder(), req)
		}
	})
}
