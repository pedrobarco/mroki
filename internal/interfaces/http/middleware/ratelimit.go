package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimitOption is a functional option for configuring RateLimit middleware
type RateLimitOption func(*rateLimitConfig)

// Internal config struct (not exported)
type rateLimitConfig struct {
	requestsPerMinute int
	extractIP         func(r *http.Request) string
	onRateLimitError  func(w http.ResponseWriter, r *http.Request)
}

// WithIPExtractor sets a custom IP extraction function
// Default: uses r.RemoteAddr
func WithIPExtractor(extractor func(r *http.Request) string) RateLimitOption {
	return func(c *rateLimitConfig) {
		c.extractIP = extractor
	}
}

// WithRateLimitErrorHandler sets the error handler callback
// The handler is called when rate limit is exceeded and should write the HTTP response
func WithRateLimitErrorHandler(handler func(w http.ResponseWriter, r *http.Request)) RateLimitOption {
	return func(c *rateLimitConfig) {
		c.onRateLimitError = handler
	}
}

// ipLimiter wraps a rate limiter with last seen timestamp for cleanup
type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimit creates middleware that enforces per-IP rate limiting using token bucket algorithm.
// requestsPerMinute: number of requests allowed per minute per IP (e.g., 1000 = ~16.67 req/sec)
//
// The middleware:
// - Tracks rate limits per unique IP address
// - Uses token bucket algorithm allowing bursts up to the full minute limit
// - Periodically cleans up limiters for IPs inactive for more than 1 hour
// - Returns 429 Too Many Requests with Retry-After header when limit exceeded
func RateLimit(requestsPerMinute int, opts ...RateLimitOption) Middleware {
	// Build config with defaults
	cfg := &rateLimitConfig{
		requestsPerMinute: requestsPerMinute,
		extractIP:         defaultExtractIP,
		onRateLimitError:  defaultRateLimitErrorHandler,
	}

	// Apply options
	for _, opt := range opts {
		opt(cfg)
	}

	limiters := &sync.Map{}

	// Cleanup goroutine removes stale limiters every 10 minutes
	// Limiters inactive for more than 1 hour are deleted to prevent memory leaks
	cleanupTicker := time.NewTicker(10 * time.Minute)
	go func() {
		for range cleanupTicker.C {
			cleanupStaleLimiters(limiters, 1*time.Hour)
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := cfg.extractIP(r)
			limiter := getOrCreateLimiter(limiters, ip, cfg.requestsPerMinute)

			if !limiter.Allow() {
				// Set Retry-After header (60 seconds = 1 minute)
				w.Header().Set("Retry-After", "60")
				cfg.onRateLimitError(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// defaultExtractIP extracts IP from RemoteAddr (format: "IP:port")
// This is secure for direct connections without a proxy
func defaultExtractIP(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// If SplitHostPort fails, RemoteAddr might already be just an IP
		return r.RemoteAddr
	}
	return ip
}

// getOrCreateLimiter retrieves or creates a rate limiter for the given IP
func getOrCreateLimiter(limiters *sync.Map, ip string, rpm int) *rate.Limiter {
	// Convert requests per minute to requests per second
	rps := float64(rpm) / 60.0

	// Allow burst up to full minute worth of requests
	// This permits legitimate clients to make burst requests while still enforcing average rate
	burst := rpm

	val, exists := limiters.Load(ip)
	if exists {
		ipl := val.(*ipLimiter)
		ipl.lastSeen = time.Now()
		return ipl.limiter
	}

	limiter := rate.NewLimiter(rate.Limit(rps), burst)
	limiters.Store(ip, &ipLimiter{
		limiter:  limiter,
		lastSeen: time.Now(),
	})

	return limiter
}

// cleanupStaleLimiters removes limiters that haven't been used in maxAge duration
func cleanupStaleLimiters(limiters *sync.Map, maxAge time.Duration) {
	now := time.Now()
	limiters.Range(func(key, value interface{}) bool {
		ipl := value.(*ipLimiter)
		if now.Sub(ipl.lastSeen) > maxAge {
			limiters.Delete(key)
		}
		return true
	})
}

// defaultRateLimitErrorHandler is a fallback that writes a simple 429 response
// Users should provide their own handler via WithRateLimitErrorHandler
func defaultRateLimitErrorHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
}

// ExtractIPWithForwardedFor returns an IP extractor that checks X-Forwarded-For and X-Real-IP headers
// Use this when the API is behind a proxy/load balancer
//
// WARNING: Only use this if your proxy is configured to set these headers correctly.
// If misconfigured, clients can spoof their IP addresses to bypass rate limits.
//
// Priority order:
// 1. X-Forwarded-For (leftmost IP = original client)
// 2. X-Real-IP
// 3. RemoteAddr (fallback)
func ExtractIPWithForwardedFor(r *http.Request) string {
	// Check X-Forwarded-For header (comma-separated list, leftmost is original client)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP (leftmost = original client)
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fallback to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
