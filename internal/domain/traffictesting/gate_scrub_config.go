package traffictesting

import (
	"fmt"
	"strings"
)

// defaultScrubFields is the default set of fields always scrubbed (gjson path notation).
var defaultScrubFields = []string{
	"headers.Authorization",
	"headers.Cookie",
	"headers.Set-Cookie",
	"headers.X-Api-Key",
}

// DefaultScrubFields returns a copy of the default scrub fields.
// Callers cannot mutate the canonical list.
func DefaultScrubFields() []string {
	out := make([]string, len(defaultScrubFields))
	copy(out, defaultScrubFields)
	return out
}

// ScrubConfig holds per-gate header scrubbing settings.
type ScrubConfig struct {
	AdditionalFields []string // extra fields beyond the defaults (gjson paths)
}

// NewScrubConfig creates a validated ScrubConfig.
// The input slice is copied — the caller's slice is never mutated.
func NewScrubConfig(additionalFields []string) (ScrubConfig, error) {
	cleaned := make([]string, len(additionalFields))
	for i, f := range additionalFields {
		trimmed := strings.TrimSpace(f)
		if trimmed == "" {
			return ScrubConfig{}, fmt.Errorf("%w: scrub_fields[%d] must not be empty", ErrInvalidScrubConfig, i)
		}
		cleaned[i] = trimmed
	}
	return ScrubConfig{AdditionalFields: cleaned}, nil
}

// DefaultScrubConfig returns the zero-value config (no additional fields).
func DefaultScrubConfig() ScrubConfig {
	return ScrubConfig{AdditionalFields: []string{}}
}

// AllFields returns the merged list of default + additional fields.
func (c ScrubConfig) AllFields() []string {
	all := make([]string, 0, len(defaultScrubFields)+len(c.AdditionalFields))
	all = append(all, defaultScrubFields...)
	all = append(all, c.AdditionalFields...)
	return all
}

// HeaderNames extracts plain header names from all fields with a "headers." prefix.
// For example, "headers.Authorization" returns "Authorization".
// Non-header paths (e.g. "body.user.ssn") are skipped.
func (c ScrubConfig) HeaderNames() []string {
	var names []string
	for _, f := range c.AllFields() {
		if after, ok := strings.CutPrefix(f, "headers."); ok && after != "" {
			names = append(names, after)
		}
	}
	return names
}
