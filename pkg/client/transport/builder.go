package transport

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/failsafe-go/failsafe-go/circuitbreaker"
	"github.com/failsafe-go/failsafe-go/failsafehttp"
)

// Config holds all parameters needed to build a resilient HTTP client.
type Config struct {
	APIKey             string
	MaxRetries         int
	InitialDelay       time.Duration
	CBFailureThreshold int
	CBDelay            time.Duration
	CBSuccessThreshold int
	Logger             *slog.Logger
}

// NewHTTPClient builds an *http.Client whose transport stack is:
//
//	failsafehttp.RoundTripper  (retry + circuit breaker)
//	  └─ loggingRoundTripper   (log method/URL/status/latency per attempt)
//	       └─ authRoundTripper (Bearer token, Content-Type)
//	            └─ http.DefaultTransport
func NewHTTPClient(cfg Config) *http.Client {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	// Build inner → outer
	var rt = http.DefaultTransport
	rt = NewAuthRoundTripper(rt, cfg.APIKey)
	rt = NewLoggingRoundTripper(rt, logger)

	// Failsafe policies
	cb := circuitbreaker.NewBuilder[*http.Response]().
		HandleIf(func(resp *http.Response, err error) bool {
			if err != nil {
				return true
			}
			if resp != nil && resp.StatusCode >= 500 {
				return true
			}
			return false
		}).
		WithFailureThreshold(uint(cfg.CBFailureThreshold)).
		WithDelay(cfg.CBDelay).
		WithSuccessThreshold(uint(cfg.CBSuccessThreshold)).
		OnStateChanged(func(e circuitbreaker.StateChangedEvent) {
			logger.Warn("circuit breaker state changed",
				"old", e.OldState, "new", e.NewState)
		}).Build()

	rp := failsafehttp.NewRetryPolicyBuilder().
		WithBackoff(cfg.InitialDelay, cfg.InitialDelay*8).
		WithMaxRetries(cfg.MaxRetries).Build()

	rt = failsafehttp.NewRoundTripper(rt, rp, cb)

	// No client-level Timeout — the caller's context controls the overall
	// deadline (including retries).  This avoids two competing timeouts.
	return &http.Client{Transport: rt}
}
