package traffictesting

import (
	"fmt"
	"strings"
)

// RequestSortField represents a sortable attribute of a Request
type RequestSortField struct {
	value string
}

const (
	sortFieldCreatedAt = "created_at"
	sortFieldMethod    = "method"
	sortFieldPath      = "path"
)

var validSortFields = map[string]bool{
	sortFieldCreatedAt: true,
	sortFieldMethod:    true,
	sortFieldPath:      true,
}

// NewRequestSortField creates a validated RequestSortField value object
// Empty string defaults to created_at (chronological sorting)
func NewRequestSortField(field string) (RequestSortField, error) {
	normalized := strings.ToLower(strings.TrimSpace(field))

	// Apply default
	if normalized == "" {
		return RequestSortField{value: sortFieldCreatedAt}, nil
	}

	// Validate against whitelist
	if !validSortFields[normalized] {
		return RequestSortField{}, fmt.Errorf(
			"invalid sort field: must be one of [created_at, method, path], got '%s'",
			field,
		)
	}

	return RequestSortField{value: normalized}, nil
}

// SortByCreatedAt returns the created_at sort field (default)
func SortByCreatedAt() RequestSortField {
	return RequestSortField{value: sortFieldCreatedAt}
}

// SortByMethod returns the method sort field
func SortByMethod() RequestSortField {
	return RequestSortField{value: sortFieldMethod}
}

// SortByPath returns the path sort field
func SortByPath() RequestSortField {
	return RequestSortField{value: sortFieldPath}
}

// IsCreatedAt returns true if sorting by creation time
func (f RequestSortField) IsCreatedAt() bool {
	return f.value == sortFieldCreatedAt
}

// IsMethod returns true if sorting by HTTP method
func (f RequestSortField) IsMethod() bool {
	return f.value == sortFieldMethod
}

// IsPath returns true if sorting by request path
func (f RequestSortField) IsPath() bool {
	return f.value == sortFieldPath
}

// String returns the field value
func (f RequestSortField) String() string {
	return f.value
}

// Equals checks value equality
func (f RequestSortField) Equals(other RequestSortField) bool {
	return f.value == other.value
}
