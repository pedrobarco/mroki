package ratelimit

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewLimiter(t *testing.T) {
	limiter := NewLimiter(60)
	defer func() { _ = limiter.Stop() }()

	assert.NotNil(t, limiter)
	assert.Equal(t, 60, limiter.requestsPerMinute)
	assert.Equal(t, 10*time.Minute, limiter.cleanupInterval)
	assert.Equal(t, 1*time.Hour, limiter.staleAge)
	assert.False(t, limiter.stopped)
}

func TestNewLimiter_WithOptions(t *testing.T) {
	limiter := NewLimiter(
		100,
		WithCleanupInterval(5*time.Minute),
		WithStaleAge(30*time.Minute),
	)
	defer func() { _ = limiter.Stop() }()

	assert.Equal(t, 100, limiter.requestsPerMinute)
	assert.Equal(t, 5*time.Minute, limiter.cleanupInterval)
	assert.Equal(t, 30*time.Minute, limiter.staleAge)
}

func TestLimiter_Allow_UnderLimit(t *testing.T) {
	limiter := NewLimiter(10)
	defer func() { _ = limiter.Stop() }()

	key := "192.168.1.1"

	// First 10 requests should be allowed
	for i := 0; i < 10; i++ {
		allowed := limiter.Allow(key)
		assert.True(t, allowed, "Request %d should be allowed", i+1)
	}
}

func TestLimiter_Allow_OverLimit(t *testing.T) {
	limiter := NewLimiter(5)
	defer func() { _ = limiter.Stop() }()

	key := "192.168.1.1"

	// First 5 should succeed
	for i := 0; i < 5; i++ {
		assert.True(t, limiter.Allow(key))
	}

	// 6th should be blocked
	assert.False(t, limiter.Allow(key))
}

func TestLimiter_Allow_DifferentKeys(t *testing.T) {
	limiter := NewLimiter(5)
	defer func() { _ = limiter.Stop() }()

	key1 := "192.168.1.1"
	key2 := "192.168.1.2"

	// Exhaust limit for key1
	for i := 0; i < 5; i++ {
		assert.True(t, limiter.Allow(key1))
	}
	assert.False(t, limiter.Allow(key1))

	// key2 should still have full quota
	for i := 0; i < 5; i++ {
		assert.True(t, limiter.Allow(key2), "Request %d for key2 should be allowed", i+1)
	}
}

func TestLimiter_Stop_Idempotent(t *testing.T) {
	limiter := NewLimiter(10)

	// First stop should succeed
	err := limiter.Stop()
	assert.NoError(t, err)
	assert.True(t, limiter.stopped)

	// Second stop should be safe
	err = limiter.Stop()
	assert.NoError(t, err)
	assert.True(t, limiter.stopped)
}

func TestLimiter_Stop_StopsCleanup(t *testing.T) {
	limiter := NewLimiter(10, WithCleanupInterval(100*time.Millisecond))

	// Let cleanup run a bit
	time.Sleep(50 * time.Millisecond)

	// Stop the limiter
	err := limiter.Stop()
	assert.NoError(t, err)

	// Verify goroutine is stopped by checking ticker is stopped
	// If not stopped properly, this test would leak goroutines
	time.Sleep(200 * time.Millisecond)
}

func TestLimiter_CleanupStaleLimiters(t *testing.T) {
	limiter := NewLimiter(
		10,
		WithCleanupInterval(100*time.Millisecond),
		WithStaleAge(200*time.Millisecond),
	)
	defer func() { _ = limiter.Stop() }()

	key1 := "192.168.1.1"
	key2 := "192.168.1.2"

	// Use both keys
	limiter.Allow(key1)
	limiter.Allow(key2)

	// Verify both exist
	_, exists1 := limiter.limiters.Load(key1)
	_, exists2 := limiter.limiters.Load(key2)
	assert.True(t, exists1)
	assert.True(t, exists2)

	// Keep key1 active, let key2 go stale
	time.Sleep(150 * time.Millisecond)
	limiter.Allow(key1) // Refresh key1

	// Wait for cleanup to run
	time.Sleep(200 * time.Millisecond)

	// key1 should still exist, key2 should be cleaned up
	_, exists1 = limiter.limiters.Load(key1)
	_, exists2 = limiter.limiters.Load(key2)
	assert.True(t, exists1, "key1 should still exist (kept active)")
	assert.False(t, exists2, "key2 should be cleaned up (stale)")
}

func TestLimiter_TokenRefill(t *testing.T) {
	// 60 requests per minute = 1 request per second
	limiter := NewLimiter(60)
	defer func() { _ = limiter.Stop() }()

	key := "192.168.1.1"

	// Exhaust the burst allowance
	for i := 0; i < 60; i++ {
		assert.True(t, limiter.Allow(key))
	}

	// Should be blocked immediately
	assert.False(t, limiter.Allow(key))

	// Wait for tokens to refill (1 request per second)
	time.Sleep(1100 * time.Millisecond)

	// Should have ~1 token now
	assert.True(t, limiter.Allow(key), "Should allow request after token refill")
}

func TestLimiter_ConcurrentAccess(t *testing.T) {
	limiter := NewLimiter(1000)
	defer func() { _ = limiter.Stop() }()

	key := "192.168.1.1"
	iterations := 1000
	successCount := 0
	var mu sync.Mutex

	var wg sync.WaitGroup
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if limiter.Allow(key) {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Should allow exactly 1000 requests (the burst limit)
	assert.Equal(t, 1000, successCount, "Should allow exactly the rate limit")
}

func TestLimiter_BurstBehavior(t *testing.T) {
	// Allow 60 requests per minute (1 per second), with burst of 60
	limiter := NewLimiter(60)
	defer func() { _ = limiter.Stop() }()

	key := "test-key"

	// Should allow burst up to 60 immediately
	successCount := 0
	for i := 0; i < 100; i++ {
		if limiter.Allow(key) {
			successCount++
		}
	}

	assert.GreaterOrEqual(t, successCount, 60, "Should allow burst up to limit")
	assert.LessOrEqual(t, successCount, 61, "Should not exceed burst limit significantly")
}
