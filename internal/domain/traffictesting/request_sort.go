package traffictesting

import "fmt"

// RequestSort represents how requests should be ordered (composite value object)
type RequestSort struct {
	field RequestSortField
	order SortOrder
}

// NewRequestSort creates RequestSort from validated value objects
// Both field and order must be pre-validated by caller (service layer)
func NewRequestSort(field RequestSortField, order SortOrder) RequestSort {
	return RequestSort{
		field: field,
		order: order,
	}
}

// DefaultRequestSort returns the business-default sorting (newest first)
func DefaultRequestSort() RequestSort {
	return RequestSort{
		field: SortByCreatedAt(),
		order: Desc(),
	}
}

// Field returns the sort field (immutable)
func (s RequestSort) Field() RequestSortField {
	return s.field
}

// Order returns the sort order (immutable)
func (s RequestSort) Order() SortOrder {
	return s.order
}

// String returns a human-readable representation for logging
func (s RequestSort) String() string {
	return fmt.Sprintf("sort by %s %s", s.field, s.order)
}

// Equals checks value equality (value object pattern)
func (s RequestSort) Equals(other RequestSort) bool {
	return s.field.Equals(other.field) && s.order.Equals(other.order)
}
