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

	client := NewMrokiClient(apiURL, gateID)

	assert.Equal(t, apiURL, client.baseURL)
	assert.Equal(t, gateID, client.gateID)
	assert.NotNil(t, client.httpClient)
	assert.NotNil(t, client.logger)
}

func TestNewMrokiClient_WithOptions(t *testing.T) {
	apiURL, _ := url.Parse("http://localhost:8080")
	customHTTPClient := &http.Client{Timeout: 5 * time.Second}

	client := NewMrokiClient(
		apiURL,
		"gate-123",
		WithHTTPClient(customHTTPClient),
	)

	assert.Equal(t, customHTTPClient, client.httpClient)
}

func TestSendRequest_SuccessOnFirstTry(t *testing.T) {
	// Create test server that always succeeds
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)

		// Verify request
		assert.Equal(t, "/gates/gate-123/requests", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

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
	client := NewMrokiClient(apiURL, "gate-123")

	// Send request
	req := &CapturedRequest{
		Method: "GET",
		Path:   "/test",
	}

	err := client.SendRequest(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, int32(1), requestCount.Load(), "should only make 1 request")
}

func TestSendRequest_NetworkError(t *testing.T) {
	// Create client pointing to non-existent server
	apiURL, _ := url.Parse("http://localhost:9999")
	client := NewMrokiClient(apiURL, "gate-123")

	// Send request
	req := &CapturedRequest{
		Method: "GET",
		Path:   "/test",
	}

	err := client.SendRequest(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP request failed")
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

			apiURL, _ := url.Parse(server.URL)
			client := NewMrokiClient(apiURL, "gate-123")

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



func TestGetGate_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "/gates/550e8400-e29b-41d4-a716-446655440000", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

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
	client := NewMrokiClient(apiURL, "550e8400-e29b-41d4-a716-446655440000")

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
	client := NewMrokiClient(apiURL, "non-existent-gate")

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
			client := NewMrokiClient(apiURL, "gate-123")

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
	client := NewMrokiClient(apiURL, "gate-123")

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
	client := NewMrokiClient(apiURL, "gate-123")

	// Call GetGate
	gate, err := client.GetGate(context.Background())

	// Should error on connection
	assert.Error(t, err)
	assert.Nil(t, gate)
	assert.Contains(t, err.Error(), "HTTP request failed")
}
