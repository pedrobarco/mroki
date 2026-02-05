package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/pedrobarco/mroki/pkg/ratelimit"
)

// RateLimitOption is a functional option for configuring RateLimit middleware
type RateLimitOption func(*rateLimitConfig)

// Internal config struct (not exported)
type rateLimitConfig struct {
	extractKey       func(r *http.Request) string
	onRateLimitError func(w http.ResponseWriter, r *http.Request)
}

// WithIPExtractor sets a custom IP extraction function.
// Default: uses r.RemoteAddr
func WithIPExtractor(extractor func(r *http.Request) string) RateLimitOption {
	return func(c *rateLimitConfig) {
		c.extractKey = extractor
	}
}

// WithRateLimitErrorHandler sets the error handler callback.
// The handler is called when rate limit is exceeded and should write the HTTP response.
func WithRateLimitErrorHandler(handler func(w http.ResponseWriter, r *http.Request)) RateLimitOption {
	return func(c *rateLimitConfig) {
		c.onRateLimitError = handler
	}
}

// RateLimit creates middleware that enforces per-IP rate limiting using the provided rate limiter.
// The limiter's lifecycle (creation and cleanup) is managed by the caller.
//
// Example:
//
//	limiter := ratelimit.NewLimiter(1000)
//	defer limiter.Stop()
//
//	mw := middleware.RateLimit(
//	    limiter,
//	    middleware.WithIPExtractor(middleware.ExtractIPWithForwardedFor),
//	    middleware.WithRateLimitErrorHandler(customHandler),
//	)
//
// The middleware:
// - Extracts a key (typically IP address) from each request
// - Checks if the key is allowed by the rate limiter
// - Returns 429 Too Many Requests with Retry-After header when limit exceeded
func RateLimit(limiter *ratelimit.Limiter, opts ...RateLimitOption) Middleware {
	// Build config with defaults
	cfg := &rateLimitConfig{
		extractKey:       defaultExtractIP,
		onRateLimitError: defaultRateLimitErrorHandler,
	}

	// Apply options
	for _, opt := range opts {
		opt(cfg)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := cfg.extractKey(r)

			if !limiter.Allow(key) {
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

// defaultRateLimitErrorHandler is a fallback that writes a simple 429 response
// Users should provide their own handler via WithRateLimitErrorHandler
func defaultRateLimitErrorHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
}

// ExtractIPWithForwardedFor returns an IP extractor that checks X-Forwarded-For and X-Real-IP headers.
// Use this when the API is behind a proxy/load balancer.
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
