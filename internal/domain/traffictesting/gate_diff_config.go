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
}

// NewDiffConfig creates a validated DiffConfig.
func NewDiffConfig(ignoredFields, includedFields []string, floatTolerance float64) (DiffConfig, error) {
	if floatTolerance < 0 {
		return DiffConfig{}, fmt.Errorf("%w: float_tolerance must be non-negative", ErrInvalidDiffConfig)
	}

	for i, f := range ignoredFields {
		trimmed := strings.TrimSpace(f)
		if trimmed == "" {
			return DiffConfig{}, fmt.Errorf("%w: ignored_fields[%d] must not be empty", ErrInvalidDiffConfig, i)
		}
		ignoredFields[i] = trimmed
	}

	for i, f := range includedFields {
		trimmed := strings.TrimSpace(f)
		if trimmed == "" {
			return DiffConfig{}, fmt.Errorf("%w: included_fields[%d] must not be empty", ErrInvalidDiffConfig, i)
		}
		includedFields[i] = trimmed
	}

	return DiffConfig{
		IgnoredFields:  ignoredFields,
		IncludedFields: includedFields,
		FloatTolerance: floatTolerance,
	}, nil
}

// DefaultDiffConfig returns the zero-value config (no filtering, no tolerance).
func DefaultDiffConfig() DiffConfig {
	return DiffConfig{
		IgnoredFields:  []string{},
		IncludedFields: []string{},
		FloatTolerance: 0,
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

	return opts
}
