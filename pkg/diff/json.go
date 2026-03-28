package diff

import (
	"fmt"

	"github.com/google/go-cmp/cmp"
	"github.com/tidwall/gjson"
)

// JSON compares two JSON strings with configurable options and returns
// a list of RFC 6902 JSON Patch operations describing the differences.
//
// The comparison process:
// 1. Normalize JSON (filter fields based on options)
// 2. Parse normalized JSON into Go values
// 3. Compare using go-cmp with a patch reporter
// 4. Return []PatchOp (empty slice if inputs are equal)
//
// Example usage:
//
//	// No filtering (compare everything)
//	ops, err := diff.JSON(a, b)
//
//	// Ignore timestamp fields (blacklist)
//	ops, err := diff.JSON(a, b,
//	    diff.WithIgnoredFields("timestamp", "users.#.created_at"),
//	)
//
//	// Only compare specific fields (whitelist - faster)
//	ops, err := diff.JSON(a, b,
//	    diff.WithIncludedFields("name", "email"),
//	)
//
//	// Hybrid: include + exclude (with multiple options)
//	ops, err := diff.JSON(a, b,
//	    diff.WithIncludedFields("user", "metadata"),
//	    diff.WithIgnoredFields("user.ssn", "metadata.internal_id"),
//	)
func JSON(a, b string, opts ...Option) ([]PatchOp, error) {
	// Build options from functional options
	cfg := &options{}
	for _, opt := range opts {
		opt(cfg)
	}

	aBytes := []byte(a)
	bBytes := []byte(b)

	// Validate JSON first
	if !gjson.ValidBytes(aBytes) {
		return nil, fmt.Errorf("invalid JSON in first input")
	}
	if !gjson.ValidBytes(bBytes) {
		return nil, fmt.Errorf("invalid JSON in second input")
	}

	// Step 1: Normalize (filter fields)
	normalizer := NewFieldNormalizer(cfg.includedFields, cfg.ignoredFields)

	normalizedA, err := normalizer.NormalizeBytes(aBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize first input: %w", err)
	}

	normalizedB, err := normalizer.NormalizeBytes(bBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize second input: %w", err)
	}

	// Step 2: Parse JSON into Go values
	resultA := gjson.ParseBytes(normalizedA)
	if !resultA.IsObject() && !resultA.IsArray() {
		return nil, fmt.Errorf("invalid JSON structure in first input: expected object or array")
	}

	resultB := gjson.ParseBytes(normalizedB)
	if !resultB.IsObject() && !resultB.IsArray() {
		return nil, fmt.Errorf("invalid JSON structure in second input: expected object or array")
	}

	// Step 3: Compare using go-cmp with patch reporter
	reporter := &patchReporter{}
	cmpOpts := append(cfg.toCmpOptions(), cmp.Reporter(reporter))
	cmp.Equal(resultA.Value(), resultB.Value(), cmpOpts...)

	ops := reporter.Ops()
	if ops == nil {
		ops = []PatchOp{}
	}
	return ops, nil
}
