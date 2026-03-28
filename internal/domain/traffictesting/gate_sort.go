package traffictesting

import "fmt"

// GateSort represents how gates should be ordered (composite value object)
type GateSort struct {
	field GateSortField
	order SortOrder
}

// NewGateSort creates GateSort from validated value objects
// Both field and order must be pre-validated by caller (service layer)
func NewGateSort(field GateSortField, order SortOrder) GateSort {
	return GateSort{
		field: field,
		order: order,
	}
}

// DefaultGateSort returns the business-default sorting (by ID ascending)
func DefaultGateSort() GateSort {
	return GateSort{
		field: SortByGateID(),
		order: Asc(),
	}
}

// Field returns the sort field (immutable)
func (s GateSort) Field() GateSortField {
	return s.field
}

// Order returns the sort order (immutable)
func (s GateSort) Order() SortOrder {
	return s.order
}

// String returns a human-readable representation for logging
func (s GateSort) String() string {
	return fmt.Sprintf("sort by %s %s", s.field, s.order)
}

// Equals checks value equality (value object pattern)
func (s GateSort) Equals(other GateSort) bool {
	return s.field.Equals(other.field) && s.order.Equals(other.order)
}
