package middleware

import (
	"net/http"
	"strconv"
	"time"
)

// SecurityHeadersOptions configures the SecurityHeaders middleware.
type SecurityHeadersOptions struct {
	// HSTSEnabled controls whether the Strict-Transport-Security header is sent.
	// It is off by default because mroki does not terminate TLS; operators must
	// opt in once a TLS-terminating reverse proxy is in front of the API.
	HSTSEnabled bool
	// HSTSMaxAge is the max-age advertised in the HSTS header. It is only used
	// when HSTSEnabled is true and must be positive.
	HSTSMaxAge time.Duration
}

// SecurityHeaders returns a middleware that emits standard security response
// headers on every response. The headers are set before the request is passed
// to the next handler so they are present even on error or non-200 responses.
//
// Always-on headers:
//   - X-Content-Type-Options: nosniff
//   - X-Frame-Options: DENY
//   - Referrer-Policy: no-referrer
//
// Strict-Transport-Security is only emitted when opts.HSTSEnabled is true;
// mroki never auto-detects TLS from the request.
func SecurityHeaders(opts SecurityHeadersOptions) Middleware {
	var hstsValue string
	if opts.HSTSEnabled && opts.HSTSMaxAge > 0 {
		seconds := int64(opts.HSTSMaxAge.Seconds())
		hstsValue = "max-age=" + strconv.FormatInt(seconds, 10) + "; includeSubDomains"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "no-referrer")
			if hstsValue != "" {
				h.Set("Strict-Transport-Security", hstsValue)
			}
			next.ServeHTTP(w, r)
		})
	}
}
