package middleware

import (
	"errors"
	"net/http"
	"strings"
)

// Authentication errors (sentinel errors)
var (
	ErrMissingAuthHeader = errors.New("missing authorization header")
	ErrInvalidAuthFormat = errors.New("invalid authorization format")
	ErrInvalidAPIKey     = errors.New("invalid api key")
)

// APIKeyAuthOption is a functional option for configuring APIKeyAuth middleware
type APIKeyAuthOption func(*apiKeyAuthConfig)

// Internal config struct (not exported)
type apiKeyAuthConfig struct {
	validKey     string
	onAuthError  func(w http.ResponseWriter, r *http.Request, err error)
	extractToken func(authHeader string) (string, bool)
}

// WithAuthErrorHandler sets the error handler callback
// The handler is called when authentication fails and should write the HTTP response
func WithAuthErrorHandler(handler func(w http.ResponseWriter, r *http.Request, err error)) APIKeyAuthOption {
	return func(c *apiKeyAuthConfig) {
		c.onAuthError = handler
	}
}

// WithTokenExtractor sets a custom token extraction function
// Defaults to Bearer token extraction if not provided
func WithTokenExtractor(extractor func(authHeader string) (string, bool)) APIKeyAuthOption {
	return func(c *apiKeyAuthConfig) {
		c.extractToken = extractor
	}
}

// APIKeyAuth creates a middleware that validates bearer tokens
// The validKey parameter is required; use options for additional configuration
func APIKeyAuth(validKey string, opts ...APIKeyAuthOption) Middleware {
	// Build config with defaults
	cfg := &apiKeyAuthConfig{
		validKey:     validKey,
		extractToken: defaultExtractToken,
		onAuthError:  defaultAuthErrorHandler,
	}

	// Apply options
	for _, opt := range opts {
		opt(cfg)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			if authHeader == "" {
				cfg.onAuthError(w, r, ErrMissingAuthHeader)
				return
			}

			token, ok := cfg.extractToken(authHeader)
			if !ok {
				cfg.onAuthError(w, r, ErrInvalidAuthFormat)
				return
			}

			if token != cfg.validKey {
				cfg.onAuthError(w, r, ErrInvalidAPIKey)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// defaultExtractToken extracts bearer token from Authorization header
func defaultExtractToken(authHeader string) (string, bool) {
	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		return "", false
	}
	token := strings.TrimPrefix(authHeader, prefix)
	if token == "" {
		return "", false
	}
	return token, true
}

// defaultAuthErrorHandler is a fallback that writes a simple 401 response
// Users should provide their own handler via WithAuthErrorHandler
func defaultAuthErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}
