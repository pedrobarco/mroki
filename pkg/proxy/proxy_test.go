package proxy_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
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

	p := proxy.NewProxy(liveURL, shadowURL, proxy.WithSamplingRate(samplingRate))

	assert.NotNil(t, p)
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
		_, _ = w.Write([]byte(`{"live":true}`))
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

	// Wait for callback to be called (it runs in background)
	select {
	case <-done:
		// Callback was called
	case <-time.After(100 * time.Millisecond):
		t.Fatal("callback was not called within timeout")
	}

	assert.Equal(t, "GET", capturedReq.Method)
	assert.Equal(t, "/test", capturedReq.Path)
	assert.Equal(t, http.StatusOK, capturedLive.StatusCode)
	assert.Equal(t, http.StatusOK, capturedShadow.StatusCode)
}

func TestProxy_ServeHTTP_skips_shadow_when_not_sampled(t *testing.T) {
	liveServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
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
		proxy.WithSamplingRate(samplingRate),
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
