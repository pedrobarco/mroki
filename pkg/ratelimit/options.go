package ratelimit

import "time"

// Option is a functional option for configuring a Limiter
type Option func(*Limiter)

// WithCleanupInterval sets how often to clean up stale limiters.
// Default: 10 minutes
func WithCleanupInterval(interval time.Duration) Option {
	return func(l *Limiter) {
		l.cleanupInterval = interval
	}
}

// WithStaleAge sets how long a limiter can be inactive before being removed.
// Default: 1 hour
func WithStaleAge(age time.Duration) Option {
	return func(l *Limiter) {
		l.staleAge = age
	}
}
