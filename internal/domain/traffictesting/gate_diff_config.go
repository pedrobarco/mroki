package traffictesting

import (
	"fmt"
	"strings"

	"github.com/pedrobarco/mroki/pkg/diff"
)

// DiffConfig holds per-gate diff computation settings.
type DiffConfig struct {
	IgnoredFields  []string
	IncludedFields []string
	FloatTolerance float64
	SortArrays     bool
}

// NewDiffConfig creates a validated DiffConfig.
// Input slices are copied — the caller's slices are never mutated.
func NewDiffConfig(ignoredFields, includedFields []string, floatTolerance float64, sortArrays bool) (DiffConfig, error) {
	if floatTolerance < 0 {
		return DiffConfig{}, fmt.Errorf("%w: float_tolerance must be non-negative", ErrInvalidDiffConfig)
	}

	cleanedIgnored := make([]string, len(ignoredFields))
	for i, f := range ignoredFields {
		trimmed := strings.TrimSpace(f)
		if trimmed == "" {
			return DiffConfig{}, fmt.Errorf("%w: ignored_fields[%d] must not be empty", ErrInvalidDiffConfig, i)
		}
		cleanedIgnored[i] = trimmed
	}

	cleanedIncluded := make([]string, len(includedFields))
	for i, f := range includedFields {
		trimmed := strings.TrimSpace(f)
		if trimmed == "" {
			return DiffConfig{}, fmt.Errorf("%w: included_fields[%d] must not be empty", ErrInvalidDiffConfig, i)
		}
		cleanedIncluded[i] = trimmed
	}

	return DiffConfig{
		IgnoredFields:  cleanedIgnored,
		IncludedFields: cleanedIncluded,
		FloatTolerance: floatTolerance,
		SortArrays:     sortArrays,
	}, nil
}

// DefaultDiffConfig returns the zero-value config (no filtering, no tolerance, no sorting).
func DefaultDiffConfig() DiffConfig {
	return DiffConfig{
		IgnoredFields:  []string{},
		IncludedFields: []string{},
		FloatTolerance: 0,
		SortArrays:     false,
	}
}

// ToDiffOptions converts the config to pkg/diff options for use with diff.JSON().
func (c DiffConfig) ToDiffOptions() []diff.Option {
	var opts []diff.Option

	if len(c.IgnoredFields) > 0 {
		opts = append(opts, diff.WithIgnoredFields(c.IgnoredFields...))
	}

	if len(c.IncludedFields) > 0 {
		opts = append(opts, diff.WithIncludedFields(c.IncludedFields...))
	}

	if c.FloatTolerance > 0 {
		opts = append(opts, diff.WithFloatTolerance(c.FloatTolerance))
	}

	opts = append(opts, diff.WithSortArrays(c.SortArrays))

	return opts
}


