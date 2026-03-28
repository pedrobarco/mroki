package traffictesting

import (
	"fmt"
	"strings"
)

// GateSortField represents a sortable attribute of a Gate
type GateSortField struct {
	value string
}

const (
	gateSortFieldID        = "id"
	gateSortFieldLiveURL   = "live_url"
	gateSortFieldShadowURL = "shadow_url"
)

var validGateSortFields = map[string]bool{
	gateSortFieldID:        true,
	gateSortFieldLiveURL:   true,
	gateSortFieldShadowURL: true,
}

// NewGateSortField creates a validated GateSortField value object
// Empty string defaults to id
func NewGateSortField(field string) (GateSortField, error) {
	normalized := strings.ToLower(strings.TrimSpace(field))

	// Apply default
	if normalized == "" {
		return GateSortField{value: gateSortFieldID}, nil
	}

	// Validate against whitelist
	if !validGateSortFields[normalized] {
		return GateSortField{}, fmt.Errorf(
			"invalid sort field: must be one of [id, live_url, shadow_url], got '%s'",
			field,
		)
	}

	return GateSortField{value: normalized}, nil
}

// SortByGateID returns the id sort field (default)
func SortByGateID() GateSortField {
	return GateSortField{value: gateSortFieldID}
}

// SortByLiveURL returns the live_url sort field
func SortByLiveURL() GateSortField {
	return GateSortField{value: gateSortFieldLiveURL}
}

// SortByShadowURL returns the shadow_url sort field
func SortByShadowURL() GateSortField {
	return GateSortField{value: gateSortFieldShadowURL}
}

// IsID returns true if sorting by ID
func (f GateSortField) IsID() bool {
	return f.value == gateSortFieldID
}

// IsLiveURL returns true if sorting by live URL
func (f GateSortField) IsLiveURL() bool {
	return f.value == gateSortFieldLiveURL
}

// IsShadowURL returns true if sorting by shadow URL
func (f GateSortField) IsShadowURL() bool {
	return f.value == gateSortFieldShadowURL
}

// String returns the field value
func (f GateSortField) String() string {
	return f.value
}

// Equals checks value equality
func (f GateSortField) Equals(other GateSortField) bool {
	return f.value == other.value
}
