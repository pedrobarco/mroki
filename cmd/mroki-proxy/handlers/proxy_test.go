package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/pedrobarco/mroki/pkg/client"
	"github.com/pedrobarco/mroki/pkg/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateStandaloneCallback_redacts_headers_and_body(t *testing.T) {
	redactor := traffictesting.NewRedactor([]string{
		"headers.Authorization",
		"headers.Cookie",
		"headers.Set-Cookie",
		"body.secret",
	})
	cfg := ProxyConfig{
		Redactor: redactor,
	}

	callback := createStandaloneCallback(cfg)

	req := proxy.ProxyRequest{
		Method: "GET",
		Path:   "/api/test",
		Headers: http.Header{
			"Authorization": {"Bearer super-secret"},
			"Content-Type":  {"application/json"},
			"X-Request-ID":  {"req-123"},
		},
		Body: []byte(`{}`),
	}

	live := proxy.ProxyResponse{
		StatusCode: 200,
		Response: &http.Response{
			StatusCode: 200,
			Header: http.Header{
				"Set-Cookie":   {"session=abc123"},
				"Content-Type": {"application/json"},
			},
		},
		Body: []byte(`{"status":"ok","secret":"s3cret"}`),
	}

	shadow := proxy.ProxyResponse{
		StatusCode: 200,
		Response: &http.Response{
			StatusCode: 200,
			Header: http.Header{
				"Cookie":       {"session=abc123"},
				"Content-Type": {"application/json"},
			},
		},
		Body: []byte(`{"status":"ok","secret":"s3cret"}`),
	}

	err := callback(req, live, shadow)
	require.NoError(t, err)

	// Verify headers were redacted (Response pointer's Header is reassigned in callback)
	assert.Equal(t, []string{traffictesting.RedactedValue}, live.Response.Header["Set-Cookie"])
	assert.Equal(t, []string{"application/json"}, live.Response.Header["Content-Type"])

	assert.Equal(t, []string{traffictesting.RedactedValue}, shadow.Response.Header["Cookie"])
	assert.Equal(t, []string{"application/json"}, shadow.Response.Header["Content-Type"])
}

func TestCreateStandaloneCallback_nil_redactor_preserves_all(t *testing.T) {
	cfg := ProxyConfig{
		Redactor: nil,
	}

	callback := createStandaloneCallback(cfg)

	req := proxy.ProxyRequest{
		Method: "GET",
		Path:   "/api/test",
		Headers: http.Header{
			"Authorization": {"Bearer token"},
			"X-Request-ID":  {"req-456"},
		},
		Body: []byte(`{}`),
	}

	live := proxy.ProxyResponse{
		StatusCode: 200,
		Response: &http.Response{
			StatusCode: 200,
			Header: http.Header{
				"Set-Cookie": {"session=xyz"},
			},
		},
		Body: []byte(`{"status":"ok"}`),
	}

	shadow := proxy.ProxyResponse{
		StatusCode: 200,
		Response: &http.Response{
			StatusCode: 200,
			Header: http.Header{
				"Set-Cookie": {"session=xyz"},
			},
		},
		Body: []byte(`{"status":"ok"}`),
	}

	err := callback(req, live, shadow)
	require.NoError(t, err)

	// Headers should remain untouched when no redactor configured
	assert.Equal(t, []string{"session=xyz"}, live.Response.Header["Set-Cookie"])
	assert.Equal(t, []string{"session=xyz"}, shadow.Response.Header["Set-Cookie"])
}

func TestCreateStandaloneCallback_redacts_body_before_diffing(t *testing.T) {
	logs := &syncBuffer{}
	cfg := ProxyConfig{
		Redactor: traffictesting.NewRedactor([]string{"body.secret"}),
		Logger:   slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug})),
	}

	callback := createStandaloneCallback(cfg)

	req := proxy.ProxyRequest{
		Method:  "GET",
		Path:    "/api/test",
		Headers: http.Header{"X-Request-ID": {"req-789"}},
		Body:    []byte(`{}`),
	}

	// Identical status and headers; the bodies differ ONLY in the redacted
	// "secret" field. The body is reassigned on a value copy inside the
	// callback, so it cannot be inspected directly here. Instead we assert the
	// observable effect: redaction collapses both secrets to [REDACTED], so the
	// only field that differed becomes identical and the diff finds no changes.
	// Were the body not redacted before diffing, the differing secret would log
	// "response diff detected" instead.
	header := http.Header{"Content-Type": {"application/json"}}
	live := proxy.ProxyResponse{
		StatusCode: 200,
		Response:   &http.Response{StatusCode: 200, Header: header.Clone()},
		Body:       []byte(`{"secret":"live-value","status":"ok"}`),
	}
	shadow := proxy.ProxyResponse{
		StatusCode: 200,
		Response:   &http.Response{StatusCode: 200, Header: header.Clone()},
		Body:       []byte(`{"secret":"shadow-value","status":"ok"}`),
	}

	err := callback(req, live, shadow)
	require.NoError(t, err)

	assert.Contains(t, logs.String(), "responses match",
		"expected body.secret to be redacted on both sides before diffing")
}

// liveResponseBody is the body returned by both fake backends in the wiring
// harness. Live and shadow return identical payloads so the standalone diff
// callback finds no differences and stays quiet during tests.
const liveResponseBody = `{"source":"live"}`

// shadowHit records that the shadow backend received a request.
type shadowHit struct {
	method string
	path   string
}

// newWiringHarness builds the proxy handler from cfg via handlers.Proxy, wiring
// in fake live + shadow servers. It returns the handler and a buffered channel
// that receives a shadowHit whenever the shadow backend is reached. The caller
// supplies only the check-related fields (MaxBodySize, SamplingRate,
// ShadowRules); Live, Shadow and Logger are filled in here.
func newWiringHarness(t *testing.T, cfg ProxyConfig) (http.HandlerFunc, <-chan shadowHit) {
	t.Helper()

	liveServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(liveResponseBody))
	}))
	t.Cleanup(liveServer.Close)

	hits := make(chan shadowHit, 1)
	shadowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case hits <- shadowHit{method: r.Method, path: r.URL.Path}:
		default:
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(liveResponseBody))
	}))
	t.Cleanup(shadowServer.Close)

	liveURL, err := url.Parse(liveServer.URL)
	require.NoError(t, err)
	shadowURL, err := url.Parse(shadowServer.URL)
	require.NoError(t, err)

	cfg.Live = liveURL
	cfg.Shadow = shadowURL
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	// handlers.Proxy forwards these durations directly to the proxy; a zero
	// value produces an already-expired context, so set generous defaults
	// unless the caller overrides them (e.g. timeout tests).
	if cfg.LiveTimeout == 0 {
		cfg.LiveTimeout = 5 * time.Second
	}
	if cfg.ShadowTimeout == 0 {
		cfg.ShadowTimeout = 10 * time.Second
	}

	return Proxy(cfg), hits
}

// assertShadowed waits for the shadow backend to be hit and returns the hit.
func assertShadowed(t *testing.T, hits <-chan shadowHit) shadowHit {
	t.Helper()
	select {
	case got := <-hits:
		return got
	case <-time.After(time.Second):
		t.Fatal("expected request to be shadowed, but shadow backend was not hit")
		return shadowHit{}
	}
}

// assertNotShadowed asserts the shadow backend is not hit within a short window.
func assertNotShadowed(t *testing.T, hits <-chan shadowHit) {
	t.Helper()
	select {
	case got := <-hits:
		t.Fatalf("expected request not to be shadowed, but shadow backend was hit: %s %s", got.method, got.path)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestProxy_wiring_shadow_rules(t *testing.T) {
	rules, err := proxy.ParseShadowRules("deny POST:*")
	require.NoError(t, err)

	handler, hits := newWiringHarness(t, ProxyConfig{ShadowRules: rules})

	// Denied: POST matches "deny POST:*" and must not reach the shadow backend.
	denied := httptest.NewRequest("POST", "/orders", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, denied)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, liveResponseBody, rec.Body.String())
	assertNotShadowed(t, hits)

	// Allowed: GET does not match the deny rule and must be shadowed.
	allowed := httptest.NewRequest("GET", "/products", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, allowed)
	got := assertShadowed(t, hits)
	assert.Equal(t, "GET", got.method)
	assert.Equal(t, "/products", got.path)
}

func TestProxy_wiring_max_body_size(t *testing.T) {
	handler, hits := newWiringHarness(t, ProxyConfig{MaxBodySize: 10})

	// Body over the limit must not be shadowed.
	large := httptest.NewRequest("POST", "/big", bytes.NewReader(bytes.Repeat([]byte("a"), 20)))
	large.ContentLength = 20
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, large)
	assert.Equal(t, http.StatusOK, rec.Code)
	assertNotShadowed(t, hits)

	// Body under the limit must be shadowed.
	small := httptest.NewRequest("POST", "/small", bytes.NewReader(bytes.Repeat([]byte("a"), 5)))
	small.ContentLength = 5
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, small)
	assertShadowed(t, hits)
}

func TestProxy_wiring_sampling_rate_zero_skips_shadow(t *testing.T) {
	rate, err := proxy.NewSamplingRate(0.0)
	require.NoError(t, err)

	handler, hits := newWiringHarness(t, ProxyConfig{SamplingRate: rate})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	assertNotShadowed(t, hits)
}

func TestProxy_wiring_sampling_rate_full_always_shadows(t *testing.T) {
	rate, err := proxy.NewSamplingRate(1.0)
	require.NoError(t, err)

	handler, hits := newWiringHarness(t, ProxyConfig{SamplingRate: rate})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	assertShadowed(t, hits)
}

func TestProxy_wiring_live_response_unaffected_by_shadow_decision(t *testing.T) {
	denyRules, err := proxy.ParseShadowRules("deny *:*")
	require.NoError(t, err)

	cases := []struct {
		name string
		cfg  ProxyConfig
	}{
		{name: "shadowed", cfg: ProxyConfig{}},
		{name: "not shadowed", cfg: ProxyConfig{ShadowRules: denyRules}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler, _ := newWiringHarness(t, tc.cfg)

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest("GET", "/test", nil))

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, liveResponseBody, rec.Body.String())
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
			// The proxy stamps the request ID onto the live response it returns.
			assert.NotEmpty(t, rec.Header().Get("X-Request-ID"))
		})
	}
}

// capturedHit records a captured request payload received by the fake API.
type capturedHit struct {
	method string
	path   string
}

// newFakeAPIServer starts a fake mroki-api that records the captured request
// POSTed to /gates/{id}/requests and returns a client pointing at it.
func newFakeAPIServer(t *testing.T) (*client.MrokiClient, <-chan capturedHit) {
	t.Helper()

	hits := make(chan capturedHit, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var captured client.CapturedRequest
		if err := json.NewDecoder(r.Body).Decode(&captured); err == nil {
			select {
			case hits <- capturedHit{method: captured.Method, path: captured.Path}:
			default:
			}
		}
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(server.Close)

	apiURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	return client.NewMrokiClient(apiURL, "gate-test"), hits
}

// syncBuffer is a goroutine-safe buffer for capturing background log output.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestProxy_wiring_api_mode_callback_sends_to_api(t *testing.T) {
	apiClient, apiHits := newFakeAPIServer(t)

	// A non-nil APIClient must wire createAPICallback (API mode).
	handler, _ := newWiringHarness(t, ProxyConfig{APIClient: apiClient})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, http.StatusOK, rec.Code)

	select {
	case got := <-apiHits:
		assert.Equal(t, "GET", got.method)
		assert.Equal(t, "/test", got.path)
	case <-time.After(time.Second):
		t.Fatal("expected API-mode callback to POST the captured request, but the API was not called")
	}
}

func TestProxy_wiring_api_timeout_is_honored(t *testing.T) {
	logs := &syncBuffer{}
	logger := slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// The API server is slower than APITimeout, so the client-side context
	// deadline must fire and the callback (which runs asynchronously in a
	// background goroutine) logs a send failure rather than success. With
	// APITimeout unset (default 30s) the call would succeed. We poll with
	// assert.Eventually since the callback's timing is not deterministic
	// relative to the live response.
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(apiServer.Close)

	apiURL, err := url.Parse(apiServer.URL)
	require.NoError(t, err)
	apiClient := client.NewMrokiClient(apiURL, "gate-test")

	handler, _ := newWiringHarness(t, ProxyConfig{
		APIClient:  apiClient,
		APITimeout: 20 * time.Millisecond,
		Logger:     logger,
	})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, http.StatusOK, rec.Code)

	assert.Eventually(t, func() bool {
		return strings.Contains(logs.String(), "failed to send request to API")
	}, time.Second, 10*time.Millisecond, "expected APITimeout to abort the slow API call")
}

func TestProxy_wiring_standalone_mode_callback_runs_diff(t *testing.T) {
	logs := &syncBuffer{}
	logger := slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// A nil APIClient must wire createStandaloneCallback (local diff).
	handler, _ := newWiringHarness(t, ProxyConfig{Logger: logger})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, http.StatusOK, rec.Code)

	// The standalone diff callback runs in the background and only it emits this
	// log line. The harness's live and shadow backends return identical
	// responses, so the local diff finds no changes and logs "responses match",
	// proving the standalone branch was selected and ran the diff.
	assert.Eventually(t, func() bool {
		return strings.Contains(logs.String(), "responses match")
	}, time.Second, 10*time.Millisecond, "expected standalone diff callback to run")
}

func TestProxy_wiring_combined_checks_use_and_logic(t *testing.T) {
	rate, err := proxy.NewSamplingRate(1.0)
	require.NoError(t, err)
	rules, err := proxy.ParseShadowRules("deny POST:*")
	require.NoError(t, err)

	handler, hits := newWiringHarness(t, ProxyConfig{
		MaxBodySize:  100,
		SamplingRate: rate,
		ShadowRules:  rules,
	})

	// All checks pass: small GET that no deny rule matches -> shadowed.
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/products", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	got := assertShadowed(t, hits)
	assert.Equal(t, "/products", got.path)

	// Fails the body-size check only -> not shadowed (AND logic).
	large := httptest.NewRequest("GET", "/products", bytes.NewReader(bytes.Repeat([]byte("a"), 200)))
	large.ContentLength = 200
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, large)
	assert.Equal(t, http.StatusOK, rec.Code)
	assertNotShadowed(t, hits)

	// Fails the shadow-rule check only -> not shadowed (AND logic).
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("POST", "/orders", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	assertNotShadowed(t, hits)
}

func TestProxy_wiring_live_timeout_returns_gateway_timeout(t *testing.T) {
	liveServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(liveServer.Close)
	shadowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(shadowServer.Close)

	liveURL, err := url.Parse(liveServer.URL)
	require.NoError(t, err)
	shadowURL, err := url.Parse(shadowServer.URL)
	require.NoError(t, err)

	// LiveTimeout must be honored: a slow live backend yields a gateway timeout.
	handler := Proxy(ProxyConfig{
		Live:          liveURL,
		Shadow:        shadowURL,
		LiveTimeout:   20 * time.Millisecond,
		ShadowTimeout: time.Second,
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/test", nil))

	assert.Equal(t, http.StatusGatewayTimeout, rec.Code)
	assert.Contains(t, rec.Body.String(), "timeout")
}

func TestProxy_wiring_shadow_timeout_skips_callback(t *testing.T) {
	apiClient, apiHits := newFakeAPIServer(t)

	liveServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(liveResponseBody))
	}))
	t.Cleanup(liveServer.Close)
	// Shadow is slower than ShadowTimeout, so the shadow request is cancelled
	// before responding and the callback must not fire.
	shadowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(shadowServer.Close)

	liveURL, err := url.Parse(liveServer.URL)
	require.NoError(t, err)
	shadowURL, err := url.Parse(shadowServer.URL)
	require.NoError(t, err)

	handler := Proxy(ProxyConfig{
		Live:          liveURL,
		Shadow:        shadowURL,
		LiveTimeout:   time.Second,
		ShadowTimeout: 20 * time.Millisecond,
		APIClient:     apiClient,
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, http.StatusOK, rec.Code)

	select {
	case got := <-apiHits:
		t.Fatalf("expected shadow timeout to skip the callback, but API was called: %s %s", got.method, got.path)
	case <-time.After(150 * time.Millisecond):
		// Expected: shadow timed out before responding, so no callback ran.
	}
}
