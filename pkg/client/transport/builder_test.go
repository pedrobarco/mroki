package transport

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/failsafe-go/failsafe-go/circuitbreaker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHTTPClient_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(Config{
		APIKey:             "test-key",
		MaxRetries:         2,
		InitialDelay:       10 * time.Millisecond,
		CBFailureThreshold: 5,
		CBDelay:            1 * time.Minute,
		CBSuccessThreshold: 2,
		Logger:             slog.Default(),
	})

	resp, err := client.Get(server.URL)
	require.NoError(t, err)
	defer func() { require.NoError(t, resp.Body.Close()) }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestNewHTTPClient_RetriesOnServerError(t *testing.T) {
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := requestCount.Add(1)
		if count <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(Config{
		APIKey:             "test-key",
		MaxRetries:         3,
		InitialDelay:       10 * time.Millisecond,
		CBFailureThreshold: 10,
		CBDelay:            1 * time.Minute,
		CBSuccessThreshold: 2,
		Logger:             slog.Default(),
	})

	resp, err := client.Get(server.URL)
	require.NoError(t, err)
	defer func() { require.NoError(t, resp.Body.Close()) }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(3), requestCount.Load(), "should retry twice then succeed")
}

func TestNewHTTPClient_CircuitBreakerOpens(t *testing.T) {
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewHTTPClient(Config{
		APIKey:             "test-key",
		MaxRetries:         0, // No retries — each call is one attempt
		InitialDelay:       10 * time.Millisecond,
		CBFailureThreshold: 3,
		CBDelay:            1 * time.Minute,
		CBSuccessThreshold: 2,
		Logger:             slog.Default(),
	})

	// Make enough failing requests to trip the circuit breaker
	for i := 0; i < 5; i++ {
		resp, err := client.Get(server.URL)
		if err != nil {
			// Circuit breaker may have opened
			if errors.Is(err, circuitbreaker.ErrOpen) {
				// Verify the server was not contacted after the CB opened
				assert.LessOrEqual(t, requestCount.Load(), int32(3),
					"should stop hitting server after CB opens")
				return
			}
		} else {
			require.NoError(t, resp.Body.Close())
		}
	}

	t.Fatal("circuit breaker should have opened within 5 requests")
}
