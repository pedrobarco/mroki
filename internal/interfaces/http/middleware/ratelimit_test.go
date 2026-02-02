package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/pedrobarco/mroki/internal/interfaces/http/middleware"
	"github.com/stretchr/testify/assert"
)

func TestRateLimit_AllowsRequestsUnderLimit(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Allow 10 requests per minute
	mw := middleware.RateLimit(10)

	// Make 5 requests - all should succeed
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()

		mw(handler).ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code, "Request %d should succeed", i+1)
	}
}

func TestRateLimit_BlocksRequestsOverLimit(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Allow only 3 requests per minute
	mw := middleware.RateLimit(3)

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	// First 3 should succeed
	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		mw(handler).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code, "Request %d should succeed", i+1)
	}

	// 4th request should be rate limited
	rec := httptest.NewRecorder()
	mw(handler).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code, "Request 4 should be rate limited")
	assert.Equal(t, "60", rec.Header().Get("Retry-After"), "Retry-After header should be set")
}

func TestRateLimit_DifferentIPsHaveSeparateLimits(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Allow 3 requests per minute per IP
	mw := middleware.RateLimit(3)

	// IP 1: Make 3 requests (exhaust limit)
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()
		mw(handler).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// IP 1: 4th request should fail
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	rec1 := httptest.NewRecorder()
	mw(handler).ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusTooManyRequests, rec1.Code, "IP 1 should be rate limited")

	// IP 2: Should still be able to make requests
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "192.168.1.2:54321"
	rec2 := httptest.NewRecorder()
	mw(handler).ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusOK, rec2.Code, "IP 2 should not be rate limited")
}

func TestRateLimit_RetryAfterHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := middleware.RateLimit(1) // Only 1 request per minute

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	// First request succeeds
	rec1 := httptest.NewRecorder()
	mw(handler).ServeHTTP(rec1, req)
	assert.Equal(t, http.StatusOK, rec1.Code)

	// Second request is rate limited
	rec2 := httptest.NewRecorder()
	mw(handler).ServeHTTP(rec2, req)
	assert.Equal(t, http.StatusTooManyRequests, rec2.Code)
	assert.Equal(t, "60", rec2.Header().Get("Retry-After"), "Should include Retry-After header")
}

func TestRateLimit_CustomIPExtractor(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Use X-Forwarded-For extractor
	mw := middleware.RateLimit(2,
		middleware.WithIPExtractor(middleware.ExtractIPWithForwardedFor),
	)

	// Make requests with X-Forwarded-For header
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "10.0.0.1:12345"                // Proxy IP
		req.Header.Set("X-Forwarded-For", "203.0.113.1") // Client IP
		rec := httptest.NewRecorder()
		mw(handler).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// 3rd request should be rate limited (based on X-Forwarded-For IP)
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.1")
	rec := httptest.NewRecorder()
	mw(handler).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestRateLimit_CustomErrorHandler(t *testing.T) {
	var capturedRequest *http.Request
	customHandler := func(w http.ResponseWriter, r *http.Request) {
		capturedRequest = r
		w.WriteHeader(http.StatusTeapot) // Use unusual status for testing
		_, _ = w.Write([]byte("custom error"))
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := middleware.RateLimit(1,
		middleware.WithRateLimitErrorHandler(customHandler),
	)

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	// First request succeeds
	rec1 := httptest.NewRecorder()
	mw(handler).ServeHTTP(rec1, req)
	assert.Equal(t, http.StatusOK, rec1.Code)

	// Second request triggers custom error handler
	rec2 := httptest.NewRecorder()
	mw(handler).ServeHTTP(rec2, req)
	assert.Equal(t, http.StatusTeapot, rec2.Code)
	assert.Equal(t, "custom error", rec2.Body.String())
	assert.NotNil(t, capturedRequest, "Custom handler should be called")
}

func TestRateLimit_ConcurrentRequests(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Allow 100 requests per minute
	mw := middleware.RateLimit(100)

	const concurrentRequests = 50
	const requestsPerWorker = 3

	var wg sync.WaitGroup
	successCount := make(chan int, concurrentRequests)

	// Launch concurrent workers making requests
	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			ip := "192.168.1." + string(rune(workerID%255))
			localSuccess := 0

			for j := 0; j < requestsPerWorker; j++ {
				req := httptest.NewRequest("GET", "/test", nil)
				req.RemoteAddr = ip + ":12345"
				rec := httptest.NewRecorder()

				mw(handler).ServeHTTP(rec, req)

				if rec.Code == http.StatusOK {
					localSuccess++
				}
			}

			successCount <- localSuccess
		}(i)
	}

	wg.Wait()
	close(successCount)

	// Count total successful requests
	total := 0
	for count := range successCount {
		total += count
	}

	// All requests should succeed (we're under the limit)
	expected := concurrentRequests * requestsPerWorker
	assert.Equal(t, expected, total, "All concurrent requests should succeed under rate limit")
}

func TestRateLimit_BurstBehavior(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Allow 60 requests per minute (1 per second average)
	// Burst allows all 60 upfront
	mw := middleware.RateLimit(60)

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	// Should be able to burst up to 60 requests immediately
	successCount := 0
	for i := 0; i < 70; i++ { // Try 70
		rec := httptest.NewRecorder()
		mw(handler).ServeHTTP(rec, req)
		if rec.Code == http.StatusOK {
			successCount++
		}
	}

	// Should get roughly 60 successful (the burst limit)
	assert.GreaterOrEqual(t, successCount, 60, "Should allow burst up to limit")
	assert.Less(t, successCount, 70, "Should not allow all requests over burst")
}

func TestExtractIPWithForwardedFor_MultipleIPs(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.1, 198.51.100.1, 192.0.2.1")

	ip := middleware.ExtractIPWithForwardedFor(req)

	// Should extract leftmost IP (original client)
	assert.Equal(t, "203.0.113.1", ip, "Should extract leftmost IP from X-Forwarded-For")
}

func TestExtractIPWithForwardedFor_XRealIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Real-IP", "203.0.113.1")

	ip := middleware.ExtractIPWithForwardedFor(req)

	assert.Equal(t, "203.0.113.1", ip, "Should extract IP from X-Real-IP")
}

func TestExtractIPWithForwardedFor_FallbackToRemoteAddr(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	ip := middleware.ExtractIPWithForwardedFor(req)

	assert.Equal(t, "192.168.1.1", ip, "Should fallback to RemoteAddr when headers absent")
}

func TestRateLimit_TokenRefill(t *testing.T) {
	// This test is time-sensitive and verifies that rate limits refill over time
	if testing.Short() {
		t.Skip("Skipping time-sensitive test in short mode")
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Allow 60 requests per minute = 1 per second
	mw := middleware.RateLimit(60)

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	// Make first request
	rec1 := httptest.NewRecorder()
	mw(handler).ServeHTTP(rec1, req)
	assert.Equal(t, http.StatusOK, rec1.Code)

	// Wait 2 seconds to allow tokens to refill (~2 tokens)
	time.Sleep(2 * time.Second)

	// Should be able to make 2 more requests
	rec2 := httptest.NewRecorder()
	mw(handler).ServeHTTP(rec2, req)
	assert.Equal(t, http.StatusOK, rec2.Code)

	rec3 := httptest.NewRecorder()
	mw(handler).ServeHTTP(rec3, req)
	assert.Equal(t, http.StatusOK, rec3.Code)
}
