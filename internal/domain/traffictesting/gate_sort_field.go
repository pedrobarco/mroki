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
	gateSortFieldName      = "name"
	gateSortFieldLiveURL   = "live_url"
	gateSortFieldShadowURL = "shadow_url"
	gateSortFieldCreatedAt = "created_at"
)

var validGateSortFields = map[string]bool{
	gateSortFieldID:        true,
	gateSortFieldName:      true,
	gateSortFieldLiveURL:   true,
	gateSortFieldShadowURL: true,
	gateSortFieldCreatedAt: true,
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
			"invalid sort field: must be one of [id, name, live_url, shadow_url, created_at], got '%s'",
			field,
		)
	}

	return GateSortField{value: normalized}, nil
}

// SortByGateID returns the id sort field (default)
func SortByGateID() GateSortField {
	return GateSortField{value: gateSortFieldID}
}

// SortByGateName returns the name sort field
func SortByGateName() GateSortField {
	return GateSortField{value: gateSortFieldName}
}

// SortByLiveURL returns the live_url sort field
func SortByLiveURL() GateSortField {
	return GateSortField{value: gateSortFieldLiveURL}
}

// SortByShadowURL returns the shadow_url sort field
func SortByShadowURL() GateSortField {
	return GateSortField{value: gateSortFieldShadowURL}
}

// SortByGateCreatedAt returns the created_at sort field
func SortByGateCreatedAt() GateSortField {
	return GateSortField{value: gateSortFieldCreatedAt}
}

// IsID returns true if sorting by ID
func (f GateSortField) IsID() bool {
	return f.value == gateSortFieldID
}

// IsName returns true if sorting by name
func (f GateSortField) IsName() bool {
	return f.value == gateSortFieldName
}

// IsLiveURL returns true if sorting by live URL
func (f GateSortField) IsLiveURL() bool {
	return f.value == gateSortFieldLiveURL
}

// IsShadowURL returns true if sorting by shadow URL
func (f GateSortField) IsShadowURL() bool {
	return f.value == gateSortFieldShadowURL
}

// IsCreatedAt returns true if sorting by created_at
func (f GateSortField) IsCreatedAt() bool {
	return f.value == gateSortFieldCreatedAt
}

// String returns the field value
func (f GateSortField) String() string {
	return f.value
}

// Equals checks value equality
func (f GateSortField) Equals(other GateSortField) bool {
	return f.value == other.value
}
