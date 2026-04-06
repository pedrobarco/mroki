package traffictesting

// GlobalStats holds cross-gate aggregate statistics.
type GlobalStats struct {
	TotalGates       int64
	TotalRequests24h int64
	TotalDiffRate    float64 // computed: total diffs 24h / total requests 24h * 100
}
