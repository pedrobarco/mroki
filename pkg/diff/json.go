package diff

import (
	"fmt"

	"github.com/google/go-cmp/cmp"
	"github.com/tidwall/gjson"
)

// JSON compares two JSON strings with configurable options.
// It supports field filtering (whitelist/blacklist), array sorting, and float tolerance.
//
// The comparison process:
// 1. Normalize JSON (filter fields based on options)
// 2. Parse normalized JSON into Go values
// 3. Compare using go-cmp with configured options
// 4. Return human-readable diff string
//
// Returns empty string if inputs are equal, or a diff string describing differences.
//
// Example usage:
//
//	// No filtering (compare everything)
//	diff, err := diff.JSON(a, b)
//
//	// Ignore timestamp fields (blacklist)
//	diff, err := diff.JSON(a, b,
//	    diff.WithIgnoredFields("timestamp", "users.#.created_at"),
//	)
//
//	// Only compare specific fields (whitelist - faster)
//	diff, err := diff.JSON(a, b,
//	    diff.WithIncludedFields("name", "email"),
//	)
//
//	// Hybrid: include + exclude (with multiple options)
//	diff, err := diff.JSON(a, b,
//	    diff.WithIncludedFields("user", "metadata"),
//	    diff.WithIgnoredFields("user.ssn", "metadata.internal_id"),
//	    diff.WithSortArrays(),
//	)
func JSON(a, b string, opts ...Option) (string, error) {
	// Build options from functional options
	cfg := &options{}
	for _, opt := range opts {
		opt(cfg)
	}

	aBytes := []byte(a)
	bBytes := []byte(b)

	// Validate JSON first
	if !gjson.ValidBytes(aBytes) {
		return "", fmt.Errorf("invalid JSON in first input")
	}
	if !gjson.ValidBytes(bBytes) {
		return "", fmt.Errorf("invalid JSON in second input")
	}

	// Step 1: Normalize (filter fields)
	normalizer := NewFieldNormalizer(cfg.includedFields, cfg.ignoredFields)

	normalizedA, err := normalizer.NormalizeBytes(aBytes)
	if err != nil {
		return "", fmt.Errorf("failed to normalize first input: %w", err)
	}

	normalizedB, err := normalizer.NormalizeBytes(bBytes)
	if err != nil {
		return "", fmt.Errorf("failed to normalize second input: %w", err)
	}

	// Step 2: Parse JSON into Go values
	resultA := gjson.ParseBytes(normalizedA)
	if !resultA.IsObject() && !resultA.IsArray() {
		return "", fmt.Errorf("invalid JSON structure in first input: expected object or array")
	}

	resultB := gjson.ParseBytes(normalizedB)
	if !resultB.IsObject() && !resultB.IsArray() {
		return "", fmt.Errorf("invalid JSON structure in second input: expected object or array")
	}

	// Step 3: Compare using go-cmp
	diff := cmp.Diff(resultA.Value(), resultB.Value(), cfg.toCmpOptions()...)

	return diff, nil
}
