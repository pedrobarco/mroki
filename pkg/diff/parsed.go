package diff

import (
	"fmt"

	"github.com/google/go-cmp/cmp"
	"github.com/pedrobarco/mroki/pkg/jsontree"
)

// Parsed compares two pre-parsed value trees and returns a list of
// RFC 6902 JSON Patch operations describing the differences.
//
// This is the optimized entry point that skips JSON parsing/validation.
// Trees should be the shape produced by json.Unmarshal: map[string]any,
// []any, string, float64, bool, or nil.
//
// Use diff.JSON() for raw JSON strings. Use diff.Parsed() when you already
// have in-memory Go values (e.g., from the redactor's BodyParsed).
//
// Example usage:
//
//	liveEnvelope := diff.BuildEnvelope(200, liveHeaders, liveBodyParsed)
//	shadowEnvelope := diff.BuildEnvelope(200, shadowHeaders, shadowBodyParsed)
//	ops, err := diff.Parsed(liveEnvelope, shadowEnvelope,
//	    diff.WithIgnoredFields("headers.Date"),
//	)
func Parsed(a, b jsontree.Tree, opts ...Option) ([]PatchOp, error) {
	cfg := &options{}
	for _, opt := range opts {
		opt(cfg)
	}

	// Validate inputs are objects or arrays (same constraint as diff.JSON)
	if err := validateTree(a, "first"); err != nil {
		return nil, err
	}
	if err := validateTree(b, "second"); err != nil {
		return nil, err
	}

	// Normalize (filter fields) on the tree directly
	normalizer := NewFieldNormalizer(cfg.includedFields, cfg.ignoredFields)
	normalizedA := normalizer.NormalizeTree(a)
	normalizedB := normalizer.NormalizeTree(b)

	// Compare using go-cmp with patch reporter
	reporter := &patchReporter{}
	cmpOpts := append(cfg.toCmpOptions(), cmp.Reporter(reporter))
	cmp.Equal(normalizedA, normalizedB, cmpOpts...)

	ops := reporter.Ops()
	if ops == nil {
		ops = []PatchOp{}
	}
	return ops, nil
}

// validateTree checks that a value is a valid root for diffing (object or array).
func validateTree(v jsontree.Tree, label string) error {
	if v == nil {
		return fmt.Errorf("invalid tree in %s input: nil", label)
	}
	switch v.(type) {
	case map[string]any, []any:
		return nil
	default:
		return fmt.Errorf("invalid tree in %s input: expected object or array, got %T", label, v)
	}
}
