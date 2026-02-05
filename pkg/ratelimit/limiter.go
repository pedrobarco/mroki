package ratelimit

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Limiter manages rate limiting state and cleanup for multiple keys.
// It uses a token bucket algorithm to enforce per-key rate limits.
// Limiter is safe for concurrent use.
type Limiter struct {
	limiters          *sync.Map
	ticker            *time.Ticker
	done              chan struct{}
	requestsPerMinute int
	cleanupInterval   time.Duration
	staleAge          time.Duration
	mu                sync.Mutex
	stopped           bool
}

// keyLimiter wraps a rate limiter with last seen timestamp for cleanup
type keyLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
	mu       sync.Mutex
}

// updateLastSeen safely updates the last seen timestamp
func (kl *keyLimiter) updateLastSeen() {
	kl.mu.Lock()
	kl.lastSeen = time.Now()
	kl.mu.Unlock()
}

// getLastSeen safely retrieves the last seen timestamp
func (kl *keyLimiter) getLastSeen() time.Time {
	kl.mu.Lock()
	defer kl.mu.Unlock()
	return kl.lastSeen
}

// NewLimiter creates a new rate limiter.
// requestsPerMinute: number of requests allowed per minute per key (e.g., 1000 = ~16.67 req/sec)
//
// The limiter:
// - Uses token bucket algorithm allowing bursts up to the full minute limit
// - Periodically cleans up limiters for keys inactive for more than staleAge
// - Is safe for concurrent use
//
// Important: Call Stop() when done to prevent goroutine leaks.
func NewLimiter(requestsPerMinute int, opts ...Option) *Limiter {
	l := &Limiter{
		limiters:          &sync.Map{},
		done:              make(chan struct{}),
		requestsPerMinute: requestsPerMinute,
		cleanupInterval:   10 * time.Minute,
		staleAge:          1 * time.Hour,
		stopped:           false,
	}

	// Apply options
	for _, opt := range opts {
		opt(l)
	}

	// Start cleanup goroutine
	l.startCleanup()

	return l
}

// Allow checks if a request with the given key is allowed under the rate limit.
// Returns true if allowed, false if rate limit exceeded.
// key is typically an IP address, user ID, API key, etc.
func (l *Limiter) Allow(key string) bool {
	limiter := l.getOrCreateLimiter(key)
	return limiter.Allow()
}

// Stop gracefully stops the cleanup goroutine and releases resources.
// It is safe to call Stop multiple times.
func (l *Limiter) Stop() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.stopped {
		return nil
	}

	l.stopped = true
	close(l.done)
	if l.ticker != nil {
		l.ticker.Stop()
	}

	return nil
}

// startCleanup begins the background goroutine that cleans up stale limiters
func (l *Limiter) startCleanup() {
	l.ticker = time.NewTicker(l.cleanupInterval)

	go func() {
		for {
			select {
			case <-l.ticker.C:
				l.cleanupStaleLimiters()
			case <-l.done:
				return
			}
		}
	}()
}

// getOrCreateLimiter retrieves or creates a rate limiter for the given key
func (l *Limiter) getOrCreateLimiter(key string) *rate.Limiter {
	// Convert requests per minute to requests per second
	rps := float64(l.requestsPerMinute) / 60.0

	// Allow burst up to full minute worth of requests
	// This permits legitimate clients to make burst requests while still enforcing average rate
	burst := l.requestsPerMinute

	val, exists := l.limiters.Load(key)
	if exists {
		kl := val.(*keyLimiter)
		kl.updateLastSeen()
		return kl.limiter
	}

	limiter := rate.NewLimiter(rate.Limit(rps), burst)
	l.limiters.Store(key, &keyLimiter{
		limiter:  limiter,
		lastSeen: time.Now(),
	})

	return limiter
}

// cleanupStaleLimiters removes limiters that haven't been used in staleAge duration
func (l *Limiter) cleanupStaleLimiters() {
	now := time.Now()
	l.limiters.Range(func(key, value interface{}) bool {
		kl := value.(*keyLimiter)
		if now.Sub(kl.getLastSeen()) > l.staleAge {
			l.limiters.Delete(key)
		}
		return true
	})
}
