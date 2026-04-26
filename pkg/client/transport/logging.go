package transport

import (
	"log/slog"
	"net/http"
	"time"
)

// loggingRoundTripper logs each outgoing HTTP request and its response.
type loggingRoundTripper struct {
	next   http.RoundTripper
	logger *slog.Logger
}

// NewLoggingRoundTripper returns a RoundTripper that logs method, URL, status
// code, and latency for every round trip via logger.
func NewLoggingRoundTripper(next http.RoundTripper, logger *slog.Logger) http.RoundTripper {
	return &loggingRoundTripper{next: next, logger: logger}
}

func (rt *loggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := rt.next.RoundTrip(req)
	latency := time.Since(start)

	if err != nil {
		rt.logger.Warn("outgoing HTTP request failed",
			slog.String("method", req.Method),
			slog.String("url", req.URL.String()),
			slog.String("error", err.Error()),
			slog.Duration("latency", latency),
		)
		return nil, err
	}

	rt.logger.Debug("outgoing HTTP request",
		slog.String("method", req.Method),
		slog.String("url", req.URL.String()),
		slog.Int("status", resp.StatusCode),
		slog.Duration("latency", latency),
	)
	return resp, nil
}
