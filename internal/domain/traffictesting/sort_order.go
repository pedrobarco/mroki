package traffictesting

import (
	"fmt"
	"strings"
)

// SortOrder represents the direction of sorting
type SortOrder struct {
	value string
}

const (
	sortOrderAsc  = "asc"
	sortOrderDesc = "desc"
)

// NewSortOrder creates a validated SortOrder value object
// Accepts only "asc" or "desc" (case-insensitive)
// Empty string defaults to "desc"
func NewSortOrder(order string) (SortOrder, error) {
	normalized := strings.ToLower(strings.TrimSpace(order))

	// Apply default
	if normalized == "" {
		return SortOrder{value: sortOrderDesc}, nil
	}

	// Validate: only "asc" or "desc" accepted
	if normalized != sortOrderAsc && normalized != sortOrderDesc {
		return SortOrder{}, fmt.Errorf("invalid sort order: must be 'asc' or 'desc', got '%s'", order)
	}

	return SortOrder{value: normalized}, nil
}

// Asc returns a SortOrder for ascending direction
func Asc() SortOrder {
	return SortOrder{value: sortOrderAsc}
}

// Desc returns a SortOrder for descending direction (default)
func Desc() SortOrder {
	return SortOrder{value: sortOrderDesc}
}

// IsAsc returns true if this is ascending order
func (s SortOrder) IsAsc() bool {
	return s.value == sortOrderAsc
}

// IsDesc returns true if this is descending order
func (s SortOrder) IsDesc() bool {
	return s.value == sortOrderDesc
}

// String returns the value ("asc" or "desc")
func (s SortOrder) String() string {
	return s.value
}

// Equals checks value equality
func (s SortOrder) Equals(other SortOrder) bool {
	return s.value == other.value
}
