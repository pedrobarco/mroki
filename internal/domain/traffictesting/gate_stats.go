package traffictesting

import "time"

// GateStats holds computed statistics for a single gate.
type GateStats struct {
	RequestCount24h int64
	DiffCount24h    int64
	DiffRate        float64    // DiffCount24h / RequestCount24h * 100, 0.0 when no requests
	LastActive      *time.Time // nil if no requests exist for this gate
}
