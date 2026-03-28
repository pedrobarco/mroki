package traffictesting

import (
	"fmt"
	"strings"
)

// GateFilters represents filtering criteria for gates (composite value object)
type GateFilters struct {
	liveURL   string
	shadowURL string
}

// NewGateFilters creates GateFilters from provided filter values
// Both filters are plain strings (substring match); empty means no filter
func NewGateFilters(liveURL, shadowURL string) GateFilters {
	return GateFilters{
		liveURL:   strings.TrimSpace(liveURL),
		shadowURL: strings.TrimSpace(shadowURL),
	}
}

// EmptyGateFilters returns filters with no criteria
func EmptyGateFilters() GateFilters {
	return GateFilters{
		liveURL:   "",
		shadowURL: "",
	}
}

// Getters (immutable)

func (f GateFilters) LiveURL() string {
	return f.liveURL
}

func (f GateFilters) ShadowURL() string {
	return f.shadowURL
}

// Business query methods

func (f GateFilters) IsEmpty() bool {
	return f.liveURL == "" && f.shadowURL == ""
}

func (f GateFilters) HasLiveURLFilter() bool {
	return f.liveURL != ""
}

func (f GateFilters) HasShadowURLFilter() bool {
	return f.shadowURL != ""
}

// String returns a human-readable representation for logging
func (f GateFilters) String() string {
	if f.IsEmpty() {
		return "no filters"
	}

	parts := []string{}

	if f.HasLiveURLFilter() {
		parts = append(parts, fmt.Sprintf("live_url='%s'", f.liveURL))
	}

	if f.HasShadowURLFilter() {
		parts = append(parts, fmt.Sprintf("shadow_url='%s'", f.shadowURL))
	}

	return strings.Join(parts, ", ")
}
