package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMrokiClient(t *testing.T) {
	apiURL, _ := url.Parse("http://localhost:8080")
	gateID := "gate-123"
	apiKey := "test-api-key-1234567890"

	client := NewMrokiClient(apiURL, gateID, apiKey)

	assert.Equal(t, apiURL, client.baseURL)
	assert.Equal(t, gateID, client.gateID)
	assert.Equal(t, apiKey, client.apiKey)
	assert.Equal(t, 3, client.maxRetries)
	assert.Equal(t, 1*time.Second, client.initialDelay)
	assert.NotNil(t, client.httpClient)
	assert.NotNil(t, client.logger)
}

func TestNewMrokiClient_WithOptions(t *testing.T) {
	apiURL, _ := url.Parse("http://localhost:8080")
	gateID := "gate-123"
	apiKey := "test-api-key-1234567890"
	customHTTPClient := &http.Client{Timeout: 5 * time.Second}

	client := NewMrokiClient(
		apiURL,
		gateID,
		apiKey,
		WithHTTPClient(customHTTPClient),
		WithMaxRetries(5),
		WithInitialDelay(2*time.Second),
	)

	assert.Equal(t, customHTTPClient, client.httpClient)
	assert.Equal(t, 5, client.maxRetries)
	assert.Equal(t, 2*time.Second, client.initialDelay)
}

func TestSendRequest_SuccessOnFirstTry(t *testing.T) {
	// Create test server that always succeeds
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)

		// Verify request
		assert.Equal(t, "/gates/gate-123/requests", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

		// Parse body
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var captured CapturedRequest
		err = json.Unmarshal(body, &captured)
		require.NoError(t, err)

		assert.Equal(t, "GET", captured.Method)
		assert.Equal(t, "/test", captured.Path)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client
	apiURL, _ := url.Parse(server.URL)
	client := NewMrokiClient(apiURL, "gate-123", "test-api-key")

	// Send request
	req := &CapturedRequest{
		Method: "GET",
		Path:   "/test",
	}

	err := client.SendRequest(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, int32(1), requestCount.Load(), "should only make 1 request")
}

func TestSendRequest_RetryThenSucceed(t *testing.T) {
	// Create test server that fails twice then succeeds
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := requestCount.Add(1)

		if count <= 2 {
			// First two attempts fail with 500
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Third attempt succeeds
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client with short delays for testing
	apiURL, _ := url.Parse(server.URL)
	client := NewMrokiClient(
		apiURL,
		"gate-123",
		"test-api-key",
		WithInitialDelay(10*time.Millisecond), // Fast for testing
	)

	// Send request
	req := &CapturedRequest{
		Method: "GET",
		Path:   "/test",
	}

	start := time.Now()
	err := client.SendRequest(context.Background(), req)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.Equal(t, int32(3), requestCount.Load(), "should make 3 requests")

	// Verify exponential backoff timing
	// Expected delays: 0ms (immediate) + 10ms (retry 1) + 20ms (retry 2) = ~30ms
	assert.GreaterOrEqual(t, duration, 30*time.Millisecond, "should have delays between retries")
}

func TestSendRequest_ExhaustAllRetries(t *testing.T) {
	// Create test server that always fails
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create client with fast retries for testing
	apiURL, _ := url.Parse(server.URL)
	client := NewMrokiClient(
		apiURL,
		"gate-123",
		"test-api-key",
		WithMaxRetries(3),
		WithInitialDelay(10*time.Millisecond),
	)

	// Send request
	req := &CapturedRequest{
		Method: "GET",
		Path:   "/test",
	}

	err := client.SendRequest(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed after 4 attempts")
	assert.Equal(t, int32(4), requestCount.Load(), "should make 4 attempts (initial + 3 retries)")
}

func TestSendRequest_ContextCancellation(t *testing.T) {
	// Create test server that always fails
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create client with long delays
	apiURL, _ := url.Parse(server.URL)
	client := NewMrokiClient(
		apiURL,
		"gate-123",
		"test-api-key",
		WithInitialDelay(1*time.Second), // Long delay
	)

	// Create cancellable context
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Send request
	req := &CapturedRequest{
		Method: "GET",
		Path:   "/test",
	}

	start := time.Now()
	err := client.SendRequest(ctx, req)
	duration := time.Since(start)

	assert.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, duration, 500*time.Millisecond, "should cancel quickly")
}

func TestSendRequest_NetworkError(t *testing.T) {
	// Create client pointing to non-existent server
	apiURL, _ := url.Parse("http://localhost:9999")
	client := NewMrokiClient(
		apiURL,
		"gate-123",
		"test-api-key",
		WithMaxRetries(1),
		WithInitialDelay(10*time.Millisecond),
	)

	// Send request
	req := &CapturedRequest{
		Method: "GET",
		Path:   "/test",
	}

	err := client.SendRequest(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed after 2 attempts")
}

func TestSendRequest_InvalidJSON(t *testing.T) {
	// This test would require making the Marshal fail, which is hard
	// In practice, Marshal only fails on channels, functions, etc.
	// We'll skip this edge case for now since our structs are always marshallable
	t.Skip("Skipping JSON marshal error test - structs are always marshallable")
}

func TestSendRequest_Non2xxStatusCode(t *testing.T) {
	testCases := []struct {
		statusCode int
		name       string
	}{
		{http.StatusBadRequest, "400 Bad Request"},
		{http.StatusUnauthorized, "401 Unauthorized"},
		{http.StatusForbidden, "403 Forbidden"},
		{http.StatusNotFound, "404 Not Found"},
		{http.StatusInternalServerError, "500 Internal Server Error"},
		{http.StatusServiceUnavailable, "503 Service Unavailable"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test server that returns specific status code
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			}))
			defer server.Close()

			// Create client with no retries for faster test
			apiURL, _ := url.Parse(server.URL)
			client := NewMrokiClient(
				apiURL,
				"gate-123",
				"test-api-key",
				WithMaxRetries(0), // No retries
			)

			// Send request
			req := &CapturedRequest{
				Method: "GET",
				Path:   "/test",
			}

			err := client.SendRequest(context.Background(), req)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), fmt.Sprintf("API returned status %d", tc.statusCode))
		})
	}
}

func TestSendRequest_ExponentialBackoffTiming(t *testing.T) {
	// Create test server that fails 3 times then succeeds
	var requestCount atomic.Int32
	var timestamps []time.Time
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timestamps = append(timestamps, time.Now())
		count := requestCount.Add(1)

		if count <= 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client with measurable delays
	apiURL, _ := url.Parse(server.URL)
	initialDelay := 50 * time.Millisecond
	client := NewMrokiClient(
		apiURL,
		"gate-123",
		"test-api-key",
		WithInitialDelay(initialDelay),
	)

	// Send request
	req := &CapturedRequest{
		Method: "GET",
		Path:   "/test",
	}

	err := client.SendRequest(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, int32(4), requestCount.Load())
	assert.Len(t, timestamps, 4)

	// Verify delays between attempts
	// Attempt 0: immediate
	// Attempt 1: after ~50ms
	// Attempt 2: after ~100ms
	// Attempt 3: after ~200ms

	delay1 := timestamps[1].Sub(timestamps[0])
	delay2 := timestamps[2].Sub(timestamps[1])
	delay3 := timestamps[3].Sub(timestamps[2])

	// Allow 20ms tolerance for timing variance
	tolerance := 20 * time.Millisecond

	assert.InDelta(t, initialDelay, delay1, float64(tolerance),
		"first retry should wait ~50ms")
	assert.InDelta(t, initialDelay*2, delay2, float64(tolerance),
		"second retry should wait ~100ms")
	assert.InDelta(t, initialDelay*4, delay3, float64(tolerance),
		"third retry should wait ~200ms")
}

func TestGetGate_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "/gates/550e8400-e29b-41d4-a716-446655440000", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

		// Send successful response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{
			"data": map[string]interface{}{
				"id":         "550e8400-e29b-41d4-a716-446655440000",
				"live_url":   "https://api.live.com",
				"shadow_url": "https://api.shadow.com",
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	apiURL, _ := url.Parse(server.URL)
	client := NewMrokiClient(
		apiURL,
		"550e8400-e29b-41d4-a716-446655440000",
		"test-api-key",
	)

	// Call GetGate
	gate, err := client.GetGate(context.Background())

	// Verify
	require.NoError(t, err)
	assert.NotNil(t, gate)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", gate.ID)
	assert.Equal(t, "https://api.live.com", gate.LiveURL)
	assert.Equal(t, "https://api.shadow.com", gate.ShadowURL)
}

func TestGetGate_NotFound(t *testing.T) {
	// Create test server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		response := map[string]interface{}{
			"status": 404,
			"title":  "Not Found",
			"detail": "Gate not found",
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	apiURL, _ := url.Parse(server.URL)
	client := NewMrokiClient(
		apiURL,
		"non-existent-gate",
		"test-api-key",
	)

	// Call GetGate
	gate, err := client.GetGate(context.Background())

	// Verify error
	assert.Error(t, err)
	assert.Nil(t, gate)
	assert.Contains(t, err.Error(), "gate not found")
}

func TestGetGate_APIError(t *testing.T) {
	testCases := []struct {
		statusCode int
		name       string
	}{
		{http.StatusBadRequest, "400 Bad Request"},
		{http.StatusUnauthorized, "401 Unauthorized"},
		{http.StatusForbidden, "403 Forbidden"},
		{http.StatusInternalServerError, "500 Internal Server Error"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.statusCode)
				response := map[string]interface{}{
					"status": tc.statusCode,
					"title":  tc.name,
					"detail": "Something went wrong",
				}
				_ = json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			// Create client
			apiURL, _ := url.Parse(server.URL)
			client := NewMrokiClient(apiURL, "gate-123", "test-api-key")

			// Call GetGate
			gate, err := client.GetGate(context.Background())

			// Verify error
			assert.Error(t, err)
			assert.Nil(t, gate)
			assert.Contains(t, err.Error(), "API error")
			assert.Contains(t, err.Error(), tc.name)
		})
	}
}

func TestGetGate_Timeout(t *testing.T) {
	// Create slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client with short timeout
	apiURL, _ := url.Parse(server.URL)
	client := NewMrokiClient(
		apiURL,
		"gate-123",
		"test-api-key",
		WithHTTPClient(&http.Client{Timeout: 50 * time.Millisecond}),
	)

	// Call GetGate with context
	ctx := context.Background()
	gate, err := client.GetGate(ctx)

	// Should timeout
	assert.Error(t, err)
	assert.Nil(t, gate)
	assert.Contains(t, err.Error(), "HTTP request failed")
}

func TestGetGate_MalformedJSON(t *testing.T) {
	// Create test server with invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data": invalid json}`))
	}))
	defer server.Close()

	// Create client
	apiURL, _ := url.Parse(server.URL)
	client := NewMrokiClient(apiURL, "gate-123", "test-api-key")

	// Call GetGate
	gate, err := client.GetGate(context.Background())

	// Should error on JSON decode
	assert.Error(t, err)
	assert.Nil(t, gate)
	assert.Contains(t, err.Error(), "failed to decode response")
}

func TestGetGate_NetworkError(t *testing.T) {
	// Create client pointing to non-existent server
	apiURL, _ := url.Parse("http://localhost:1")
	client := NewMrokiClient(apiURL, "gate-123", "test-api-key")

	// Call GetGate
	gate, err := client.GetGate(context.Background())

	// Should error on connection
	assert.Error(t, err)
	assert.Nil(t, gate)
	assert.Contains(t, err.Error(), "HTTP request failed")
}
