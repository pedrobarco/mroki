package dto

// GateStats holds computed statistics for a single gate.
type GateStats struct {
	RequestCount24h int64   `json:"request_count_24h"`
	DiffCount24h    int64   `json:"diff_count_24h"`
	DiffRate        float64 `json:"diff_rate"`
	LastActive      *string `json:"last_active"`
}

// DiffConfig holds per-gate diff computation settings.
type DiffConfig struct {
	IgnoredFields  []string `json:"ignored_fields"`
	IncludedFields []string `json:"included_fields"`
	FloatTolerance float64  `json:"float_tolerance"`
}

// Gate represents a traffic testing gate with live and shadow URLs.
type Gate struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	LiveURL    string     `json:"live_url"`
	ShadowURL  string     `json:"shadow_url"`
	DiffConfig DiffConfig `json:"diff_config"`
	CreatedAt  string     `json:"created_at"`
	Stats      GateStats  `json:"stats"`
}
