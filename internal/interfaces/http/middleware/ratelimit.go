package middleware

import (
	"fmt"
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
//	extractIP, _ := middleware.NewForwardedForExtractor(cfg.ParseTrustedProxies())
//	mw := middleware.RateLimit(
//	    limiter,
//	    middleware.WithIPExtractor(extractIP),
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

// NewForwardedForExtractor returns an IP extractor that honors the
// X-Forwarded-For (and X-Real-IP) headers only when a request's immediate peer
// (RemoteAddr) is one of the configured trusted proxies. This prevents clients
// from spoofing their source IP to evade per-IP rate limiting.
//
// trustedProxies is a list of CIDRs (e.g. "10.0.0.0/8") or bare IPs (e.g.
// "192.168.1.1"). An empty list means no peer is ever trusted, so the extractor
// always keys off RemoteAddr and ignores forwarding headers. An invalid entry
// returns an error.
//
// When the peer is trusted, the client IP is taken as the right-most entry in
// the X-Forwarded-For chain that is NOT itself a trusted proxy. This walks past
// any chained trusted proxies to reach the real client while ignoring
// attacker-supplied left-hand entries. All X-Forwarded-For header lines are
// joined into a single chain, and each entry is normalized (an "ip:port" pair
// is reduced to its IP and malformed entries are skipped) so the resulting key
// cannot be rotated via ports or spoofed junk. If X-Forwarded-For yields no
// untrusted hop, the extractor falls back to X-Real-IP and finally to
// RemoteAddr.
func NewForwardedForExtractor(trustedProxies []string) (func(r *http.Request) string, error) {
	nets := make([]*net.IPNet, 0, len(trustedProxies))
	for _, entry := range trustedProxies {
		n, err := parseTrustedEntry(entry)
		if err != nil {
			return nil, err
		}
		nets = append(nets, n)
	}

	return func(r *http.Request) string {
		peer := defaultExtractIP(r)

		// Untrusted (or no) peer: never trust forwarding headers.
		if !ipInNets(peer, nets) {
			return peer
		}

		// Trusted peer: return the right-most X-Forwarded-For entry that is not
		// itself a trusted proxy (the real client behind the proxy chain). All
		// X-Forwarded-For header lines are joined so that proxies which append a
		// separate header (rather than extending the existing one) are still
		// considered as part of the chain.
		if values := r.Header.Values("X-Forwarded-For"); len(values) > 0 {
			chain := strings.Split(strings.Join(values, ","), ",")
			for i := len(chain) - 1; i >= 0; i-- {
				ip := normalizeIP(chain[i])
				if ip == "" {
					continue
				}
				if !ipInNets(ip, nets) {
					return ip
				}
			}
		}

		// No untrusted X-Forwarded-For hop; fall back to X-Real-IP then peer.
		if xri := normalizeIP(r.Header.Get("X-Real-IP")); xri != "" {
			return xri
		}
		return peer
	}, nil
}

// normalizeIP trims and parses a forwarding-header entry, which may be a bare
// IP or an "ip:port" pair, and returns the canonical IP string (e.g.
// "::ffff:1.2.3.4" becomes "1.2.3.4"). It returns "" when the entry does not
// contain a valid IP, so callers can skip spoofed or malformed hops rather than
// keying rate limiting off attacker-controlled junk.
func normalizeIP(entry string) string {
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return ""
	}
	if ip := net.ParseIP(entry); ip != nil {
		return ip.String()
	}
	if host, _, err := net.SplitHostPort(entry); err == nil {
		if ip := net.ParseIP(strings.TrimSpace(host)); ip != nil {
			return ip.String()
		}
	}
	return ""
}

// parseTrustedEntry parses a trusted-proxy entry as either a CIDR or a bare IP
// (converted to a single-host network).
func parseTrustedEntry(entry string) (*net.IPNet, error) {
	if _, n, err := net.ParseCIDR(entry); err == nil {
		return n, nil
	}
	if ip := net.ParseIP(entry); ip != nil {
		bits := 32
		if ip.To4() == nil {
			bits = 128
		}
		return &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)}, nil
	}
	return nil, fmt.Errorf("invalid trusted proxy %q: must be a valid IP or CIDR", entry)
}

// ipInNets reports whether ipStr parses to an IP contained in any of nets.
// An unparseable IP is never contained.
func ipInNets(ipStr string, nets []*net.IPNet) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, n := range nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}
