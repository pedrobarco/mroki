package traffictesting

import (
	"fmt"
	"time"
)

// Retention holds a gate's per-gate request retention setting.
//
// A gate either uses a custom retention (a positive duration) or falls back to
// the global retention floor. Retention can never be disabled at the gate level;
// the zero value represents "use the global retention".
type Retention struct {
	duration time.Duration
	set      bool
}

// ParseRetention parses a Go duration string (e.g. "168h") into a Retention.
// The duration must be positive; keep-forever (0) and negative values are
// rejected. Callers enforce the global-minimum bound separately, since the
// domain value object has no knowledge of the global configuration.
func ParseRetention(s string) (Retention, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return Retention{}, fmt.Errorf("%w: %q is not a valid duration", ErrInvalidRetention, s)
	}
	if d <= 0 {
		return Retention{}, fmt.Errorf("%w: must be positive, got %s", ErrInvalidRetention, d)
	}
	return Retention{duration: d, set: true}, nil
}

// NoRetention returns a Retention that falls back to the global retention floor.
func NoRetention() Retention {
	return Retention{}
}

// GateRetention pairs a gate identifier with its retention setting. It is used
// by the cleanup job to resolve per-gate effective retention efficiently
// without loading full gate aggregates.
type GateRetention struct {
	ID        GateID
	Retention Retention
}

// IsSet reports whether a custom per-gate retention is configured.
func (r Retention) IsSet() bool {
	return r.set
}

// Duration returns the custom retention duration. It is only meaningful when
// IsSet() is true; otherwise it returns 0.
func (r Retention) Duration() time.Duration {
	return r.duration
}

// Effective returns the retention duration to apply for this gate. The global
// value is an authoritative floor: the result is max(custom, global), so a gate
// is never pruned below the global retention even if the custom value predates a
// floor increase or was written directly to the database below the floor.
func (r Retention) Effective(global time.Duration) time.Duration {
	if r.set && r.duration > global {
		return r.duration
	}
	return global
}

// String returns the Go duration string for a custom retention, or the empty
// string when the gate falls back to the global retention.
func (r Retention) String() string {
	if !r.set {
		return ""
	}
	return r.duration.String()
}
