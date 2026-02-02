package middleware

import (
	"net/http"
)

// MaxBodySize returns middleware that limits request body size.
// If the request body exceeds maxBytes, subsequent reads will fail
// and handlers should return a 413 error.
//
// This middleware wraps the request body with http.MaxBytesReader,
// which enforces the size limit when the body is actually read.
func MaxBodySize(maxBytes int64) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Wrap body with size limiter
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

			next.ServeHTTP(w, r)
		})
	}
}
