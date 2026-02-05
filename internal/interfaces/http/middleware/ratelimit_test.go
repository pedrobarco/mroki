package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/pedrobarco/mroki/internal/interfaces/http/middleware"
	"github.com/pedrobarco/mroki/pkg/ratelimit"
	"github.com/stretchr/testify/assert"
)

func TestRateLimit_AllowsRequestsUnderLimit(t *testing.T) {
	limiter := ratelimit.NewLimiter(10)
	defer func() { _ = limiter.Stop() }()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := middleware.RateLimit(limiter)

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
	limiter := ratelimit.NewLimiter(3)
	defer func() { _ = limiter.Stop() }()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := middleware.RateLimit(limiter)

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
	limiter := ratelimit.NewLimiter(3)
	defer func() { _ = limiter.Stop() }()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := middleware.RateLimit(limiter)

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
	limiter := ratelimit.NewLimiter(1)
	defer func() { _ = limiter.Stop() }()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := middleware.RateLimit(limiter)

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	// First request should succeed
	rec1 := httptest.NewRecorder()
	mw(handler).ServeHTTP(rec1, req)
	assert.Equal(t, http.StatusOK, rec1.Code)

	// Second request should be rate limited
	rec2 := httptest.NewRecorder()
	mw(handler).ServeHTTP(rec2, req)
	assert.Equal(t, http.StatusTooManyRequests, rec2.Code)

	// Verify Retry-After header is set
	retryAfter := rec2.Header().Get("Retry-After")
	assert.Equal(t, "60", retryAfter, "Retry-After should be 60 seconds")
}

func TestRateLimit_CustomIPExtractor(t *testing.T) {
	limiter := ratelimit.NewLimiter(3)
	defer func() { _ = limiter.Stop() }()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Custom extractor that uses X-Forwarded-For
	mw := middleware.RateLimit(limiter,
		middleware.WithIPExtractor(middleware.ExtractIPWithForwardedFor),
	)

	// Make 3 requests with X-Forwarded-For header
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Forwarded-For", "10.0.0.1")
		rec := httptest.NewRecorder()
		mw(handler).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// 4th request should be rate limited (same X-Forwarded-For)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	rec := httptest.NewRecorder()
	mw(handler).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestRateLimit_CustomErrorHandler(t *testing.T) {
	limiter := ratelimit.NewLimiter(1)
	defer func() { _ = limiter.Stop() }()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	customErrorCalled := false
	customErrorHandler := func(w http.ResponseWriter, r *http.Request) {
		customErrorCalled = true
		w.WriteHeader(http.StatusTeapot) // 418
		_, _ = w.Write([]byte("Custom rate limit error"))
	}

	mw := middleware.RateLimit(limiter,
		middleware.WithRateLimitErrorHandler(customErrorHandler),
	)

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	// First request succeeds
	rec1 := httptest.NewRecorder()
	mw(handler).ServeHTTP(rec1, req)
	assert.Equal(t, http.StatusOK, rec1.Code)
	assert.False(t, customErrorCalled)

	// Second request triggers custom error handler
	rec2 := httptest.NewRecorder()
	mw(handler).ServeHTTP(rec2, req)
	assert.Equal(t, http.StatusTeapot, rec2.Code)
	assert.True(t, customErrorCalled)
	assert.Contains(t, rec2.Body.String(), "Custom rate limit error")
}

func TestRateLimit_ConcurrentRequests(t *testing.T) {
	limiter := ratelimit.NewLimiter(150)
	defer func() { _ = limiter.Stop() }()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := middleware.RateLimit(limiter)

	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	// Make 150 concurrent requests
	for i := 0; i < 150; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.1:12345"
			rec := httptest.NewRecorder()
			mw(handler).ServeHTTP(rec, req)

			if rec.Code == http.StatusOK {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	assert.Equal(t, 150, successCount, "All concurrent requests should succeed under rate limit")
}

func TestRateLimit_BurstBehavior(t *testing.T) {
	limiter := ratelimit.NewLimiter(60)
	defer func() { _ = limiter.Stop() }()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := middleware.RateLimit(limiter)

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	// Should allow burst up to 60 requests immediately
	successCount := 0
	for i := 0; i < 70; i++ {
		rec := httptest.NewRecorder()
		mw(handler).ServeHTTP(rec, req)
		if rec.Code == http.StatusOK {
			successCount++
		}
	}

	assert.GreaterOrEqual(t, successCount, 60, "Should allow burst up to limit")
}

func TestExtractIPWithForwardedFor_MultipleIPs(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8, 9.10.11.12")

	ip := middleware.ExtractIPWithForwardedFor(req)
	assert.Equal(t, "1.2.3.4", ip, "Should extract first IP from X-Forwarded-For")
}

func TestExtractIPWithForwardedFor_XRealIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Real-IP", "1.2.3.4")

	ip := middleware.ExtractIPWithForwardedFor(req)
	assert.Equal(t, "1.2.3.4", ip, "Should extract IP from X-Real-IP")
}

func TestExtractIPWithForwardedFor_FallbackToRemoteAddr(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	ip := middleware.ExtractIPWithForwardedFor(req)
	assert.Equal(t, "192.168.1.1", ip, "Should fall back to RemoteAddr")
}

func TestRateLimit_TokenRefill(t *testing.T) {
	// 60 requests per minute = 1 request per second
	limiter := ratelimit.NewLimiter(60)
	defer func() { _ = limiter.Stop() }()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := middleware.RateLimit(limiter)

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	// Exhaust burst allowance
	for i := 0; i < 60; i++ {
		rec := httptest.NewRecorder()
		mw(handler).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// Should be blocked
	rec1 := httptest.NewRecorder()
	mw(handler).ServeHTTP(rec1, req)
	assert.Equal(t, http.StatusTooManyRequests, rec1.Code)

	// Wait for token refill (1 second)
	time.Sleep(1100 * time.Millisecond)

	// Should have refilled ~1 token
	rec2 := httptest.NewRecorder()
	mw(handler).ServeHTTP(rec2, req)
	assert.Equal(t, http.StatusOK, rec2.Code, "Should allow request after token refill")
}
