package traffictesting

import (
	"fmt"
	"strings"
)

// GateFilters represents filtering criteria for gates (composite value object)
type GateFilters struct {
	name      string
	liveURL   string
	shadowURL string
}

// NewGateFilters creates GateFilters from provided filter values
// All filters are plain strings (substring match); empty means no filter
func NewGateFilters(name, liveURL, shadowURL string) GateFilters {
	return GateFilters{
		name:      strings.TrimSpace(name),
		liveURL:   strings.TrimSpace(liveURL),
		shadowURL: strings.TrimSpace(shadowURL),
	}
}

// EmptyGateFilters returns filters with no criteria
func EmptyGateFilters() GateFilters {
	return GateFilters{
		name:      "",
		liveURL:   "",
		shadowURL: "",
	}
}

// Getters (immutable)

func (f GateFilters) Name() string {
	return f.name
}

func (f GateFilters) LiveURL() string {
	return f.liveURL
}

func (f GateFilters) ShadowURL() string {
	return f.shadowURL
}

// Business query methods

func (f GateFilters) IsEmpty() bool {
	return f.name == "" && f.liveURL == "" && f.shadowURL == ""
}

func (f GateFilters) HasNameFilter() bool {
	return f.name != ""
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

	if f.HasNameFilter() {
		parts = append(parts, fmt.Sprintf("name='%s'", f.name))
	}

	if f.HasLiveURLFilter() {
		parts = append(parts, fmt.Sprintf("live_url='%s'", f.liveURL))
	}

	if f.HasShadowURLFilter() {
		parts = append(parts, fmt.Sprintf("shadow_url='%s'", f.shadowURL))
	}

	return strings.Join(parts, ", ")
}
