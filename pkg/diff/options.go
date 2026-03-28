package diff

import (
	"encoding/json"
	"fmt"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// options holds internal configuration for JSON comparison.
// Use With* functions to configure comparison behavior.
type options struct {
	includedFields []string
	ignoredFields  []string
	floatTolerance float64
}

// Option is a functional option for configuring JSON comparison.
type Option func(*options)

// WithIncludedFields sets a whitelist of fields to include in comparison.
// Uses gjson path syntax (e.g., "name", "users.#.email").
// Can be combined with WithIgnoredFields for hybrid filtering.
//
// Example:
//
//	diff.JSON(a, b, diff.WithIncludedFields("name", "email"))
func WithIncludedFields(fields ...string) Option {
	return func(o *options) {
		o.includedFields = fields
	}
}

// WithIgnoredFields sets a blacklist of fields to exclude from comparison.
// Uses gjson path syntax (e.g., "timestamp", "users.#.created_at").
// Can be combined with WithIncludedFields for hybrid filtering.
//
// Example:
//
//	diff.JSON(a, b, diff.WithIgnoredFields("timestamp", "request_id"))
func WithIgnoredFields(fields ...string) Option {
	return func(o *options) {
		o.ignoredFields = fields
	}
}

// WithFloatTolerance sets the acceptable difference for floating-point comparisons.
// Values closer than this tolerance are considered equal.
//
// Example:
//
//	diff.JSON(a, b, diff.WithFloatTolerance(0.0001))
func WithFloatTolerance(tolerance float64) Option {
	return func(o *options) {
		o.floatTolerance = tolerance
	}
}

// Field filtering strategies:
// - WithIncludedFields only: Whitelist (keep only these fields)
// - WithIgnoredFields only: Blacklist (remove these fields)
// - Both: Hybrid (include fields first, then exclude from included)
//
// Example hybrid usage:
//
//	diff.JSON(a, b,
//	    diff.WithIncludedFields("user", "metadata"),
//	    diff.WithIgnoredFields("user.ssn", "metadata.internal_id"),
//	)
//	// Result: Compares user (except ssn) and metadata (except internal_id)

// toCmpOptions converts options to go-cmp options.
func (o *options) toCmpOptions() []cmp.Option {
	var opts []cmp.Option

	// Always sort slices to avoid false positives from array ordering
	opts = append(opts, cmpopts.SortSlices(func(a, b any) bool {
		return toSortKey(a) < toSortKey(b)
	}))

	// Apply float tolerance if specified
	if o.floatTolerance > 0 {
		opts = append(opts, cmpopts.EquateApprox(0, o.floatTolerance))
	}

	return opts
}

// toSortKey converts a value to a deterministic string for sorting.
// Handles all JSON-decoded types: string, float64, bool, nil,
// map[string]interface{}, and []interface{}.
func toSortKey(v any) string {
	switch val := v.(type) {
	case string:
		return "s:" + val
	case float64:
		return fmt.Sprintf("n:%g", val)
	case bool:
		return fmt.Sprintf("b:%t", val)
	case nil:
		return "z:null"
	default:
		// For complex types (maps, slices), produce a canonical JSON string.
		// json.Marshal sorts map keys, giving deterministic output.
		b, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("x:%v", val)
		}
		return "j:" + string(b)
	}
}
