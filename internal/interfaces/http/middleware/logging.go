package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type wrappedWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *wrappedWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func Logging(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			now := time.Now()

			wrapped := &wrappedWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(wrapped, r)

			latency := time.Since(now)

			// Use a constant message with the status carried as structured
			// attributes so log streams (especially JSON) group cleanly by
			// message rather than by every distinct status line.
			logger.Info("request completed",
				slog.String("request.id", GetRequestID(r.Context())),
				slog.String("request.method", r.Method),
				slog.String("request.path", r.URL.Path),
				slog.Int("response.status", wrapped.statusCode),
				slog.String("response.status_text", http.StatusText(wrapped.statusCode)),
				slog.Duration("response.latency", latency),
			)
		})
	}
}
