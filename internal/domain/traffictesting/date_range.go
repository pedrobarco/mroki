package traffictesting

import (
	"fmt"
	"time"
)

// DateRange represents a time range for filtering (immutable)
type DateRange struct {
	from *time.Time
	to   *time.Time
}

// NewDateRange creates a validated DateRange value object
// Both from and to are optional (nil means unbounded)
// If both provided, from must be <= to
func NewDateRange(from *time.Time, to *time.Time) (DateRange, error) {
	// Validate range if both provided
	if from != nil && to != nil {
		if from.After(*to) {
			return DateRange{}, fmt.Errorf(
				"invalid date range: from (%s) must be before or equal to to (%s)",
				from.Format(time.RFC3339),
				to.Format(time.RFC3339),
			)
		}
	}

	return DateRange{
		from: from,
		to:   to,
	}, nil
}

// EmptyDateRange returns a date range with no filtering
func EmptyDateRange() DateRange {
	return DateRange{}
}

// From returns the start date (may be nil)
func (d DateRange) From() *time.Time {
	return d.from
}

// To returns the end date (may be nil)
func (d DateRange) To() *time.Time {
	return d.to
}

// IsEmpty returns true if no date filtering is applied
func (d DateRange) IsEmpty() bool {
	return d.from == nil && d.to == nil
}

// HasFrom returns true if a start date is set
func (d DateRange) HasFrom() bool {
	return d.from != nil
}

// HasTo returns true if an end date is set
func (d DateRange) HasTo() bool {
	return d.to != nil
}

// String returns a human-readable representation
func (d DateRange) String() string {
	if d.IsEmpty() {
		return "no date range"
	}

	if d.from != nil && d.to != nil {
		return fmt.Sprintf("%s to %s",
			d.from.Format(time.RFC3339),
			d.to.Format(time.RFC3339))
	}

	if d.from != nil {
		return fmt.Sprintf("from %s", d.from.Format(time.RFC3339))
	}

	return fmt.Sprintf("until %s", d.to.Format(time.RFC3339))
}

// Equals checks value equality
func (d DateRange) Equals(other DateRange) bool {
	fromEqual := (d.from == nil && other.from == nil) ||
		(d.from != nil && other.from != nil && d.from.Equal(*other.from))

	toEqual := (d.to == nil && other.to == nil) ||
		(d.to != nil && other.to != nil && d.to.Equal(*other.to))

	return fromEqual && toEqual
}
