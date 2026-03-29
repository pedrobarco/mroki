package traffictesting

import (
	"fmt"
	"strings"
)

// RequestFilters represents filtering criteria (composite value object)
type RequestFilters struct {
	methods     []HTTPMethod
	pathPattern PathPattern
	dateRange   DateRange
	hasDiff     *bool
}

// NewRequestFilters creates RequestFilters from validated value objects
// All value objects must be pre-validated by caller (service layer)
func NewRequestFilters(
	methods []HTTPMethod,
	pathPattern PathPattern,
	dateRange DateRange,
	hasDiff *bool,
) RequestFilters {
	// Normalize nil to empty slice
	if methods == nil {
		methods = []HTTPMethod{}
	}

	return RequestFilters{
		methods:     methods,
		pathPattern: pathPattern,
		dateRange:   dateRange,
		hasDiff:     hasDiff,
	}
}

// EmptyRequestFilters returns filters with no criteria
func EmptyRequestFilters() RequestFilters {
	return RequestFilters{
		methods:     []HTTPMethod{},
		pathPattern: EmptyPathPattern(),
		dateRange:   EmptyDateRange(),
		hasDiff:     nil,
	}
}

// Getters (immutable - return copies where applicable)

func (f RequestFilters) Methods() []HTTPMethod {
	// Return copy to maintain immutability
	if len(f.methods) == 0 {
		return []HTTPMethod{}
	}
	copy := make([]HTTPMethod, len(f.methods))
	copy = append(copy[:0], f.methods...)
	return copy
}

func (f RequestFilters) PathPattern() PathPattern {
	return f.pathPattern
}

func (f RequestFilters) DateRange() DateRange {
	return f.dateRange
}

func (f RequestFilters) HasDiff() *bool {
	return f.hasDiff
}

// Business query methods

func (f RequestFilters) IsEmpty() bool {
	return len(f.methods) == 0 &&
		f.pathPattern.IsEmpty() &&
		f.dateRange.IsEmpty() &&
		f.hasDiff == nil
}

func (f RequestFilters) HasMethodFilter() bool {
	return len(f.methods) > 0
}

func (f RequestFilters) HasPathFilter() bool {
	return !f.pathPattern.IsEmpty()
}

func (f RequestFilters) HasDateRangeFilter() bool {
	return !f.dateRange.IsEmpty()
}

func (f RequestFilters) HasDiffFilter() bool {
	return f.hasDiff != nil
}

// String returns a human-readable representation for logging
func (f RequestFilters) String() string {
	if f.IsEmpty() {
		return "no filters"
	}

	parts := []string{}

	if f.HasMethodFilter() {
		methodStrs := make([]string, len(f.methods))
		for i, m := range f.methods {
			methodStrs[i] = m.String()
		}
		parts = append(parts, fmt.Sprintf("methods=[%s]", strings.Join(methodStrs, ",")))
	}

	if f.HasPathFilter() {
		parts = append(parts, fmt.Sprintf("path='%s'", f.pathPattern))
	}

	if f.HasDateRangeFilter() {
		parts = append(parts, f.dateRange.String())
	}

	if f.HasDiffFilter() {
		parts = append(parts, fmt.Sprintf("hasDiff=%t", *f.hasDiff))
	}

	return strings.Join(parts, ", ")
}
