package dto

// GlobalStats holds cross-gate aggregate statistics.
type GlobalStats struct {
	TotalGates       int64   `json:"total_gates"`
	TotalRequests24h int64   `json:"total_requests_24h"`
	TotalDiffRate    float64 `json:"total_diff_rate"`
}
