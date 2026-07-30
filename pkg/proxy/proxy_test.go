package proxy_test

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pedrobarco/mroki/pkg/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProxy_creates_proxy_with_defaults(t *testing.T) {
	liveURL, _ := url.Parse("http://live.example.com")
	shadowURL, _ := url.Parse("http://shadow.example.com")

	p := proxy.NewProxy(liveURL, shadowURL)

	assert.NotNil(t, p)
	assert.Equal(t, liveURL, p.Live)
	assert.Equal(t, shadowURL, p.Shadow)
}

func TestNewProxy_with_custom_timeouts(t *testing.T) {
	liveURL, _ := url.Parse("http://live.example.com")
	shadowURL, _ := url.Parse("http://shadow.example.com")

	p := proxy.NewProxy(
		liveURL,
		shadowURL,
		proxy.WithLiveTimeout(10*time.Second),
		proxy.WithShadowTimeout(20*time.Second),
	)

	assert.NotNil(t, p)
}

func TestNewProxy_with_sampling_rate(t *testing.T) {
	liveURL, _ := url.Parse("http://live.example.com")
	shadowURL, _ := url.Parse("http://shadow.example.com")
	samplingRate, _ := proxy.NewSamplingRate(0.5)

	p := proxy.NewProxy(liveURL, shadowURL, proxy.WithShouldProxyToShadow(proxy.SamplingRateCheck(samplingRate)))

	assert.NotNil(t, p)
}

func TestNewHTTPClient_applies_config_verbatim(t *testing.T) {
	cfg := proxy.HTTPClientConfig{
		MaxIdleConns:        7,
		MaxIdleConnsPerHost: 3,
		MaxConnsPerHost:     11,
		IdleConnTimeout:     42 * time.Second,
	}

	client := proxy.NewHTTPClient(cfg)
	require.NotNil(t, client)

	tr, ok := client.Transport.(*http.Transport)
	require.True(t, ok, "transport should be *http.Transport")

	// Configurable fields are applied verbatim (no clamping or fallback).
	assert.Equal(t, 7, tr.MaxIdleConns)
	assert.Equal(t, 3, tr.MaxIdleConnsPerHost)
	assert.Equal(t, 11, tr.MaxConnsPerHost)
	assert.Equal(t, 42*time.Second, tr.IdleConnTimeout)

	// Zero values are passed through as-is (net/http semantics), not defaulted.
	zero := proxy.NewHTTPClient(proxy.HTTPClientConfig{})
	ztr, ok := zero.Transport.(*http.Transport)
	require.True(t, ok)
	assert.Equal(t, 0, ztr.MaxIdleConns)
	assert.Equal(t, 0, ztr.MaxIdleConnsPerHost)
	assert.Equal(t, 0, ztr.MaxConnsPerHost)
	assert.Equal(t, time.Duration(0), ztr.IdleConnTimeout)

	// Non-tunable transport settings are fixed regardless of config.
	assert.Equal(t, 5*time.Second, tr.TLSHandshakeTimeout)
	assert.Equal(t, 1*time.Second, tr.ExpectContinueTimeout)
	assert.True(t, tr.ForceAttemptHTTP2)
	assert.Zero(t, client.Timeout, "client uses context timeouts, not a client-level timeout")
}

func TestProxy_ServeHTTP_returns_live_response(t *testing.T) {
	// Create mock live server
	liveServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"source":"live"}`))
	}))
	defer liveServer.Close()

	// Create mock shadow server
	shadowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"source":"shadow"}`))
	}))
	defer shadowServer.Close()

	liveURL, _ := url.Parse(liveServer.URL)
	shadowURL, _ := url.Parse(shadowServer.URL)
	p := proxy.NewProxy(liveURL, shadowURL)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"source":"live"`)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
}

func TestProxy_ServeHTTP_forwards_request_body(t *testing.T) {
	receivedBody := ""

	// Create mock live server that captures body
	liveServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedBody = string(body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"received":true}`))
	}))
	defer liveServer.Close()

	shadowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer shadowServer.Close()

	liveURL, _ := url.Parse(liveServer.URL)
	shadowURL, _ := url.Parse(shadowServer.URL)
	p := proxy.NewProxy(liveURL, shadowURL)

	requestBody := `{"test":"data"}`
	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(requestBody))
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, requestBody, receivedBody)
}

func TestProxy_ServeHTTP_handles_live_timeout(t *testing.T) {
	// Create slow live server
	liveServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer liveServer.Close()

	shadowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer shadowServer.Close()

	liveURL, _ := url.Parse(liveServer.URL)
	shadowURL, _ := url.Parse(shadowServer.URL)
	p := proxy.NewProxy(
		liveURL,
		shadowURL,
		proxy.WithLiveTimeout(10*time.Millisecond),
	)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusGatewayTimeout, rec.Code)
	assert.Contains(t, rec.Body.String(), "timeout")
}

func TestProxy_ServeHTTP_handles_live_error(t *testing.T) {
	// Create live server that returns error
	liveServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	liveServer.Close() // Close immediately to cause connection error

	shadowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer shadowServer.Close()

	liveURL, _ := url.Parse(liveServer.URL)
	shadowURL, _ := url.Parse(shadowServer.URL)
	p := proxy.NewProxy(liveURL, shadowURL)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadGateway, rec.Code)
	assert.Contains(t, rec.Body.String(), "live backend error")
}

func TestProxy_ServeHTTP_with_callback(t *testing.T) {
	liveServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("live response"))
	}))
	defer liveServer.Close()

	shadowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"shadow":true}`))
	}))
	defer shadowServer.Close()

	done := make(chan struct{})
	var capturedReq proxy.ProxyRequest
	var capturedLive, capturedShadow proxy.ProxyResponse

	liveURL, _ := url.Parse(liveServer.URL)
	shadowURL, _ := url.Parse(shadowServer.URL)
	p := proxy.NewProxy(
		liveURL,
		shadowURL,
		proxy.WithCallbackFn(func(req proxy.ProxyRequest, live, shadow proxy.ProxyResponse) error {
			capturedReq = req
			capturedLive = live
			capturedShadow = shadow
			close(done)
			return nil
		}),
	)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	// Wait for callback to be called (it runs in background goroutine)
	// The callback is called after both live and shadow requests complete,
	// so we need a reasonable timeout that accounts for:
	// - Network roundtrips (even to localhost test servers)
	// - Goroutine scheduling delays
	// - Callback execution
	select {
	case <-done:
		// Callback was called successfully
	case <-time.After(1 * time.Second):
		t.Fatal("callback was not called within timeout")
	}

	assert.Equal(t, "GET", capturedReq.Method)
	assert.Equal(t, "/test", capturedReq.Path)
	assert.Equal(t, http.StatusOK, capturedLive.StatusCode)
	assert.Equal(t, http.StatusOK, capturedShadow.StatusCode)

	// Verify latency is captured (>= 0; may be 0ms for localhost test servers)
	assert.GreaterOrEqual(t, capturedLive.LatencyMs, int64(0), "live latency should be captured")
	assert.GreaterOrEqual(t, capturedShadow.LatencyMs, int64(0), "shadow latency should be captured")
}

func TestProxy_ServeHTTP_skips_shadow_when_not_sampled(t *testing.T) {
	liveServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("live response"))
	}))
	defer liveServer.Close()

	shadowCalled := make(chan struct{})
	shadowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(shadowCalled)
		w.WriteHeader(http.StatusOK)
	}))
	defer shadowServer.Close()

	// Use 0% sampling rate
	samplingRate, _ := proxy.NewSamplingRate(0.0)
	liveURL, _ := url.Parse(liveServer.URL)
	shadowURL, _ := url.Parse(shadowServer.URL)
	p := proxy.NewProxy(
		liveURL,
		shadowURL,
		proxy.WithShouldProxyToShadow(proxy.SamplingRateCheck(samplingRate)),
	)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	// Wait a bit to ensure shadow would have been called if sampled
	select {
	case <-shadowCalled:
		t.Fatal("shadow should not have been called when sampling rate is 0")
	case <-time.After(50 * time.Millisecond):
		// Expected - shadow was not called
	}

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestProxy_ServeHTTP_forwards_request_method(t *testing.T) {
	receivedMethod := ""

	liveServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer liveServer.Close()

	shadowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer shadowServer.Close()

	liveURL, _ := url.Parse(liveServer.URL)
	shadowURL, _ := url.Parse(shadowServer.URL)
	p := proxy.NewProxy(liveURL, shadowURL)

	req := httptest.NewRequest("POST", "/test", nil)
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	assert.Equal(t, "POST", receivedMethod)
}

func TestProxy_ServeHTTP_forwards_headers(t *testing.T) {
	var receivedHeaders http.Header

	liveServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer liveServer.Close()

	shadowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer shadowServer.Close()

	liveURL, _ := url.Parse(liveServer.URL)
	shadowURL, _ := url.Parse(shadowServer.URL)
	p := proxy.NewProxy(liveURL, shadowURL)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Custom-Header", "test-value")
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	assert.Equal(t, "test-value", receivedHeaders.Get("X-Custom-Header"))
}

func TestProxy_ServeHTTP_preserves_query_params(t *testing.T) {
	var receivedQuery string

	liveServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
	}))
	defer liveServer.Close()

	shadowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer shadowServer.Close()

	liveURL, _ := url.Parse(liveServer.URL)
	shadowURL, _ := url.Parse(shadowServer.URL)
	p := proxy.NewProxy(liveURL, shadowURL)

	req := httptest.NewRequest("GET", "/test?param1=value1&param2=value2", nil)
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	assert.Contains(t, receivedQuery, "param1=value1")
	assert.Contains(t, receivedQuery, "param2=value2")
}

func TestProxy_ServeHTTP_callback_captures_raw_query(t *testing.T) {
	liveServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer liveServer.Close()

	shadowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer shadowServer.Close()

	done := make(chan struct{})
	var capturedReq proxy.ProxyRequest

	liveURL, _ := url.Parse(liveServer.URL)
	shadowURL, _ := url.Parse(shadowServer.URL)
	p := proxy.NewProxy(
		liveURL,
		shadowURL,
		proxy.WithCallbackFn(func(req proxy.ProxyRequest, live, shadow proxy.ProxyResponse) error {
			capturedReq = req
			close(done)
			return nil
		}),
	)

	req := httptest.NewRequest("GET", "/test?foo=bar&baz=qux", nil)
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	select {
	case <-done:
		// Callback was called successfully
	case <-time.After(1 * time.Second):
		t.Fatal("callback was not called within timeout")
	}

	assert.Equal(t, "foo=bar&baz=qux", capturedReq.RawQuery)
}

func TestProxy_ServeHTTP_copies_response_headers(t *testing.T) {
	liveServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom-Response", "custom-value")
		w.Header().Add("X-Multi", "value1")
		w.Header().Add("X-Multi", "value2")
		w.WriteHeader(http.StatusOK)
	}))
	defer liveServer.Close()

	shadowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer shadowServer.Close()

	liveURL, _ := url.Parse(liveServer.URL)
	shadowURL, _ := url.Parse(shadowServer.URL)
	p := proxy.NewProxy(liveURL, shadowURL)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	require.Equal(t, "custom-value", rec.Header().Get("X-Custom-Response"))
	assert.Equal(t, []string{"value1", "value2"}, rec.Header()["X-Multi"])
}

func TestProxy_ServeHTTP_skips_shadow_when_body_too_large(t *testing.T) {
	var shadowCalled atomic.Bool
	liveBody := make(chan []byte, 1)

	liveServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		liveBody <- b
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("live response"))
	}))
	defer liveServer.Close()

	shadowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		shadowCalled.Store(true)
		w.WriteHeader(http.StatusOK)
	}))
	defer shadowServer.Close()

	liveURL, _ := url.Parse(liveServer.URL)
	shadowURL, _ := url.Parse(shadowServer.URL)

	// Set max body size to 10 bytes
	p := proxy.NewProxy(liveURL, shadowURL, proxy.WithMaxBodySize(10))

	// Create request with 20 bytes body (exceeds limit)
	body := bytes.Repeat([]byte("a"), 20)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	req.ContentLength = 20
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	// Should return live response
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "live response", rec.Body.String())

	// Live must have received the FULL body despite exceeding the limit (streamed,
	// never rejected or truncated).
	select {
	case got := <-liveBody:
		assert.Equal(t, body, got, "live must receive the full oversized body")
	case <-time.After(time.Second):
		t.Fatal("live server did not receive the request body")
	}

	// Shadow should not be called (give it time to process if it was)
	time.Sleep(50 * time.Millisecond)
	assert.False(t, shadowCalled.Load(), "shadow service should not be called for large bodies")
}

func TestProxy_ServeHTTP_proxies_shadow_when_body_under_limit(t *testing.T) {
	var shadowCalled atomic.Bool

	liveServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("live response"))
	}))
	defer liveServer.Close()

	shadowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		shadowCalled.Store(true)
		w.WriteHeader(http.StatusOK)
	}))
	defer shadowServer.Close()

	liveURL, _ := url.Parse(liveServer.URL)
	shadowURL, _ := url.Parse(shadowServer.URL)

	// Set max body size to 100 bytes
	p := proxy.NewProxy(liveURL, shadowURL, proxy.WithMaxBodySize(100))

	// Create request with 20 bytes body (under limit)
	body := bytes.Repeat([]byte("a"), 20)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	req.ContentLength = 20
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	// Should return live response
	assert.Equal(t, http.StatusOK, rec.Code)

	// Shadow should be called (give it time to process)
	time.Sleep(50 * time.Millisecond)
	assert.True(t, shadowCalled.Load(), "shadow service should be called for small bodies")
}

func TestProxy_ServeHTTP_shadows_chunked_within_limit(t *testing.T) {
	var shadowCalled atomic.Bool
	shadowBody := make(chan []byte, 1)

	liveServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("live response"))
	}))
	defer liveServer.Close()

	shadowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		shadowCalled.Store(true)
		b, _ := io.ReadAll(r.Body)
		shadowBody <- b
		w.WriteHeader(http.StatusOK)
	}))
	defer shadowServer.Close()

	liveURL, _ := url.Parse(liveServer.URL)
	shadowURL, _ := url.Parse(shadowServer.URL)

	// Set max body size to 100 bytes
	p := proxy.NewProxy(liveURL, shadowURL, proxy.WithMaxBodySize(100))

	// Create request with unknown Content-Length (chunked) that fits the limit.
	payload := []byte("test body")
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(payload))
	req.ContentLength = -1 // Simulate chunked encoding
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	// Should return live response
	assert.Equal(t, http.StatusOK, rec.Code)

	// Shadow must run for chunked bodies that fit within the limit, and receive
	// the full buffered body.
	select {
	case got := <-shadowBody:
		assert.Equal(t, payload, got, "shadow must receive the full chunked body")
	case <-time.After(time.Second):
		t.Fatal("shadow service was not called for chunked body within limit")
	}
	assert.True(t, shadowCalled.Load(), "shadow service should be called for chunked bodies within limit")
}

func TestProxy_ServeHTTP_streams_chunked_over_limit(t *testing.T) {
	var shadowCalled atomic.Bool
	liveBody := make(chan []byte, 1)

	liveServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		liveBody <- b
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("live response"))
	}))
	defer liveServer.Close()

	shadowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		shadowCalled.Store(true)
		w.WriteHeader(http.StatusOK)
	}))
	defer shadowServer.Close()

	liveURL, _ := url.Parse(liveServer.URL)
	shadowURL, _ := url.Parse(shadowServer.URL)

	// Set max body size to 10 bytes
	p := proxy.NewProxy(liveURL, shadowURL, proxy.WithMaxBodySize(10))

	// Create request with unknown Content-Length (chunked) that exceeds the limit.
	// The proxy must not buffer it all: it streams the buffered prefix plus the
	// remainder to live, and skips shadow.
	payload := bytes.Repeat([]byte("a"), 100)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(payload))
	req.ContentLength = -1 // Simulate chunked encoding
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	// Should return live response
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "live response", rec.Body.String())

	// Live must receive the FULL body even though it exceeds the buffer limit.
	select {
	case got := <-liveBody:
		assert.Equal(t, payload, got, "live must receive the full over-limit chunked body")
	case <-time.After(time.Second):
		t.Fatal("live server did not receive the request body")
	}

	// Shadow must be skipped for over-limit chunked bodies.
	time.Sleep(50 * time.Millisecond)
	assert.False(t, shadowCalled.Load(), "shadow service should not be called for over-limit chunked bodies")
}

func TestProxy_ServeHTTP_unlimited_when_zero(t *testing.T) {
	var shadowCalled atomic.Bool

	liveServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("live response"))
	}))
	defer liveServer.Close()

	shadowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		shadowCalled.Store(true)
		w.WriteHeader(http.StatusOK)
	}))
	defer shadowServer.Close()

	liveURL, _ := url.Parse(liveServer.URL)
	shadowURL, _ := url.Parse(shadowServer.URL)

	// Set max body size to 0 (unlimited) - no check needed, default behavior
	p := proxy.NewProxy(liveURL, shadowURL)

	// Create request with large body
	body := bytes.Repeat([]byte("a"), 1000)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	req.ContentLength = 1000
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	// Should return live response
	assert.Equal(t, http.StatusOK, rec.Code)

	// Shadow should be called even with large body (give it time to process)
	time.Sleep(50 * time.Millisecond)
	assert.True(t, shadowCalled.Load(), "shadow service should be called when max body size is 0 (unlimited)")
}

func TestProxy_ServeHTTP_adds_shadow_header_to_shadow_only(t *testing.T) {
	var liveMode, liveHost atomic.Value
	liveServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		liveMode.Store(r.Header.Get(proxy.ShadowHeader))
		liveHost.Store(r.Host)
		w.WriteHeader(http.StatusOK)
	}))
	defer liveServer.Close()

	shadowReceived := make(chan struct {
		mode string
		host string
	}, 1)
	shadowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		shadowReceived <- struct {
			mode string
			host string
		}{mode: r.Header.Get(proxy.ShadowHeader), host: r.Host}
		w.WriteHeader(http.StatusOK)
	}))
	defer shadowServer.Close()

	liveURL, _ := url.Parse(liveServer.URL)
	shadowURL, _ := url.Parse(shadowServer.URL)
	p := proxy.NewProxy(liveURL, shadowURL)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	select {
	case got := <-shadowReceived:
		assert.Equal(t, "shadow", got.mode, "shadow request should carry the identification header")
		// Host must not get a "-shadow" suffix (avoid Envoy anti-pattern); it
		// is the shadow target host as set by normal URL rewriting.
		assert.Equal(t, shadowURL.Host, got.host, "shadow Host must be the target host, unmodified")
		assert.NotContains(t, got.host, "-shadow", "Host must never be suffixed with -shadow")
	case <-time.After(1 * time.Second):
		t.Fatal("shadow service was not called within timeout")
	}

	// Live request must not be modified in any way.
	assert.Equal(t, "", liveMode.Load(), "live request must not carry the shadow identification header")
	assert.Equal(t, liveURL.Host, liveHost.Load(), "live Host must be the target host, unmodified")
}

func TestProxy_ServeHTTP_captures_shadow_header_in_request_data(t *testing.T) {
	liveServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer liveServer.Close()

	shadowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer shadowServer.Close()

	done := make(chan struct{})
	var capturedReq proxy.ProxyRequest

	liveURL, _ := url.Parse(liveServer.URL)
	shadowURL, _ := url.Parse(shadowServer.URL)
	p := proxy.NewProxy(
		liveURL,
		shadowURL,
		proxy.WithCallbackFn(func(req proxy.ProxyRequest, live, shadow proxy.ProxyResponse) error {
			capturedReq = req
			close(done)
			return nil
		}),
	)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("callback was not called within timeout")
	}

	assert.Equal(t, "shadow", capturedReq.Headers.Get(proxy.ShadowHeader),
		"stored request data should capture the shadow identification header")
}

func TestProxy_ServeHTTP_bounds_callback_concurrency(t *testing.T) {
	liveServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"source":"live"}`))
	}))
	defer liveServer.Close()

	shadowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"source":"shadow"}`))
	}))
	defer shadowServer.Close()

	// The callback occupies its slot until release is closed, so a single-slot
	// semaphore is guaranteed full for the duration of the second request.
	var callbackCount atomic.Int32
	callbackStarted := make(chan struct{}, 1)
	callbackDone := make(chan struct{}, 1)
	release := make(chan struct{})

	// Capture proxy logs to assert the drop warning is emitted. slog's handlers
	// serialize writes internally, so concurrent goroutines are safe.
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	liveURL, _ := url.Parse(liveServer.URL)
	shadowURL, _ := url.Parse(shadowServer.URL)
	p := proxy.NewProxy(
		liveURL,
		shadowURL,
		proxy.WithMaxConcurrentCallbacks(1),
		proxy.WithLogger(logger),
		proxy.WithCallbackFn(func(req proxy.ProxyRequest, live, shadow proxy.ProxyResponse) error {
			callbackCount.Add(1)
			callbackStarted <- struct{}{}
			<-release
			callbackDone <- struct{}{}
			return nil
		}),
	)

	// First request: live returns immediately; its callback acquires the only
	// slot and blocks, holding the semaphore full.
	rec1 := httptest.NewRecorder()
	p.ServeHTTP(rec1, httptest.NewRequest("GET", "/one", nil))
	require.Equal(t, http.StatusOK, rec1.Code)
	select {
	case <-callbackStarted:
	case <-time.After(time.Second):
		close(release)
		t.Fatal("first callback did not start within timeout")
	}

	// Second request: live still succeeds, but the semaphore is full so the
	// shadow comparison is dropped synchronously with a warning — the callback
	// is never invoked a second time.
	rec2 := httptest.NewRecorder()
	p.ServeHTTP(rec2, httptest.NewRequest("GET", "/two", nil))
	assert.Equal(t, http.StatusOK, rec2.Code, "live response must succeed even when the callback is dropped")
	assert.Contains(t, rec2.Body.String(), `"source":"live"`)
	assert.Contains(t, logBuf.String(), "callback semaphore full, dropping shadow comparison")

	// Release the first callback and wait for it to finish so its slot release
	// is observed (rather than racing test teardown), then confirm only one
	// callback ever ran.
	close(release)
	select {
	case <-callbackDone:
	case <-time.After(time.Second):
		t.Fatal("first callback did not finish after release")
	}
	assert.Equal(t, int32(1), callbackCount.Load(), "second shadow comparison should have been dropped, not queued")
}

func TestProxy_ServeHTTP_unbounded_callbacks_when_unset(t *testing.T) {
	liveServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"source":"live"}`))
	}))
	defer liveServer.Close()

	shadowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer shadowServer.Close()

	const n = 5
	var wg sync.WaitGroup
	wg.Add(n)

	liveURL, _ := url.Parse(liveServer.URL)
	shadowURL, _ := url.Parse(shadowServer.URL)
	// No WithMaxConcurrentCallbacks => unbounded; every callback must run even
	// while others are still blocked.
	releaseAll := make(chan struct{})
	p := proxy.NewProxy(
		liveURL,
		shadowURL,
		proxy.WithCallbackFn(func(req proxy.ProxyRequest, live, shadow proxy.ProxyResponse) error {
			wg.Done()
			<-releaseAll
			return nil
		}),
	)

	for i := 0; i < n; i++ {
		rec := httptest.NewRecorder()
		p.ServeHTTP(rec, httptest.NewRequest("GET", "/x", nil))
		require.Equal(t, http.StatusOK, rec.Code)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		close(releaseAll)
		t.Fatal("not all callbacks ran concurrently when unbounded")
	}
	close(releaseAll)
}

// TestProxy_ServeHTTP_unbounded_callbacks_with_zero is the explicit companion to
// the "unset" case above: WithMaxConcurrentCallbacks(0) documents 0 as unbounded,
// so every callback must still run concurrently. This guards the n>0 allocation
// guard in NewProxy against an accidental n>=0 regression.
func TestProxy_ServeHTTP_unbounded_callbacks_with_zero(t *testing.T) {
	liveServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"source":"live"}`))
	}))
	defer liveServer.Close()

	shadowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer shadowServer.Close()

	const n = 5
	var wg sync.WaitGroup
	wg.Add(n)

	liveURL, _ := url.Parse(liveServer.URL)
	shadowURL, _ := url.Parse(shadowServer.URL)
	releaseAll := make(chan struct{})
	p := proxy.NewProxy(
		liveURL,
		shadowURL,
		proxy.WithMaxConcurrentCallbacks(0), // 0 = unbounded, same as not setting the option
		proxy.WithCallbackFn(func(req proxy.ProxyRequest, live, shadow proxy.ProxyResponse) error {
			wg.Done()
			<-releaseAll
			return nil
		}),
	)

	for i := 0; i < n; i++ {
		rec := httptest.NewRecorder()
		p.ServeHTTP(rec, httptest.NewRequest("GET", "/x", nil))
		require.Equal(t, http.StatusOK, rec.Code)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		close(releaseAll)
		t.Fatal("not all callbacks ran concurrently when limit set to 0 (unbounded)")
	}
	close(releaseAll)
}

// TestProxy_ServeHTTP_recovers_from_callback_panic asserts the core invariant:
// a panic inside the comparison callback (diff/redaction/API client) is
// contained — the live response is still returned, the process stays up, and
// the callback semaphore slot is released so subsequent comparisons still run.
// Without recovery the first panic would crash the whole test process.
func TestProxy_ServeHTTP_recovers_from_callback_panic(t *testing.T) {
	liveServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("live response"))
	}))
	defer liveServer.Close()

	shadowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"source":"shadow"}`))
	}))
	defer shadowServer.Close()

	// The panic path logs a stack trace at error level; discard it to keep the
	// test output clean.
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	var count atomic.Int32
	invoked := make(chan int32, 16)

	liveURL, _ := url.Parse(liveServer.URL)
	shadowURL, _ := url.Parse(shadowServer.URL)
	// Limit to a single slot so a leaked (never-released) slot on the panic path
	// would permanently starve every subsequent callback.
	p := proxy.NewProxy(
		liveURL,
		shadowURL,
		proxy.WithMaxConcurrentCallbacks(1),
		proxy.WithLogger(logger),
		proxy.WithCallbackFn(func(req proxy.ProxyRequest, live, shadow proxy.ProxyResponse) error {
			n := count.Add(1)
			invoked <- n
			if n == 1 {
				panic("boom in callback")
			}
			return nil
		}),
	)

	// First request: the callback panics. The live response must still be
	// returned to the client unaffected.
	rec1 := httptest.NewRecorder()
	p.ServeHTTP(rec1, httptest.NewRequest("GET", "/one", nil))
	require.Equal(t, http.StatusOK, rec1.Code)
	require.Equal(t, "live response", rec1.Body.String())

	select {
	case n := <-invoked:
		require.Equal(t, int32(1), n, "first callback should have run")
	case <-time.After(time.Second):
		t.Fatal("first (panicking) callback was not invoked within timeout")
	}

	// Reaching here proves the panic was contained (an unrecovered panic in the
	// background goroutine would have crashed the test process). The slot must
	// also have been released: subsequent callbacks must still run. Retry since
	// slot release on the panic path races with the next request.
	require.Eventually(t, func() bool {
		rec := httptest.NewRecorder()
		p.ServeHTTP(rec, httptest.NewRequest("GET", "/two", nil))
		if rec.Code != http.StatusOK {
			return false
		}
		select {
		case <-invoked:
			return true
		case <-time.After(50 * time.Millisecond):
			return false
		}
	}, 2*time.Second, 10*time.Millisecond,
		"callback slot was not released after panic; subsequent callbacks never ran")
}
