package traffictesting_test

import (
	"testing"

	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDiffConfig_valid(t *testing.T) {
	cfg, err := traffictesting.NewDiffConfig(
		[]string{"timestamp"},
		[]string{"body.user"},
		0.01,
		false,
	)

	require.NoError(t, err)
	assert.Equal(t, []string{"timestamp"}, cfg.IgnoredFields)
	assert.Equal(t, []string{"body.user"}, cfg.IncludedFields)
	assert.Equal(t, 0.01, cfg.FloatTolerance)
	assert.False(t, cfg.SortArrays)
}

func TestNewDiffConfig_trims_whitespace(t *testing.T) {
	cfg, err := traffictesting.NewDiffConfig(
		[]string{"  timestamp  "},
		[]string{"  body.user  "},
		0,
		false,
	)

	require.NoError(t, err)
	assert.Equal(t, []string{"timestamp"}, cfg.IgnoredFields)
	assert.Equal(t, []string{"body.user"}, cfg.IncludedFields)
}

func TestNewDiffConfig_rejects_empty_ignored_field(t *testing.T) {
	_, err := traffictesting.NewDiffConfig([]string{""}, []string{}, 0, false)

	require.Error(t, err)
	assert.ErrorIs(t, err, traffictesting.ErrInvalidDiffConfig)
	assert.Contains(t, err.Error(), "ignored_fields[0]")
}

func TestNewDiffConfig_rejects_empty_included_field(t *testing.T) {
	_, err := traffictesting.NewDiffConfig([]string{}, []string{"valid", ""}, 0, false)

	require.Error(t, err)
	assert.ErrorIs(t, err, traffictesting.ErrInvalidDiffConfig)
	assert.Contains(t, err.Error(), "included_fields[1]")
}

func TestNewDiffConfig_rejects_negative_tolerance(t *testing.T) {
	_, err := traffictesting.NewDiffConfig([]string{}, []string{}, -1, false)

	require.Error(t, err)
	assert.ErrorIs(t, err, traffictesting.ErrInvalidDiffConfig)
	assert.Contains(t, err.Error(), "float_tolerance")
}

func TestNewDiffConfig_does_not_mutate_input(t *testing.T) {
	ignored := []string{"  timestamp  "}
	included := []string{"  body  "}
	origIgnored := ignored[0]
	origIncluded := included[0]

	_, _ = traffictesting.NewDiffConfig(ignored, included, 0, false)

	assert.Equal(t, origIgnored, ignored[0], "ignored input slice must not be mutated")
	assert.Equal(t, origIncluded, included[0], "included input slice must not be mutated")
}

func TestNewDiffConfig_empty_slices(t *testing.T) {
	cfg, err := traffictesting.NewDiffConfig([]string{}, []string{}, 0, false)

	require.NoError(t, err)
	assert.Empty(t, cfg.IgnoredFields)
	assert.Empty(t, cfg.IncludedFields)
	assert.Equal(t, 0.0, cfg.FloatTolerance)
}

func TestNewDiffConfig_with_sort_arrays_true(t *testing.T) {
	cfg, err := traffictesting.NewDiffConfig(nil, nil, 0, true)

	require.NoError(t, err)
	assert.True(t, cfg.SortArrays)
}

func TestDefaultDiffConfig(t *testing.T) {
	cfg := traffictesting.DefaultDiffConfig()

	assert.Empty(t, cfg.IgnoredFields)
	assert.Empty(t, cfg.IncludedFields)
	assert.Equal(t, 0.0, cfg.FloatTolerance)
	assert.False(t, cfg.SortArrays)
}

func TestDiffConfig_ToDiffOptions(t *testing.T) {
	cfg, err := traffictesting.NewDiffConfig(nil, nil, 0, true)
	require.NoError(t, err)

	opts := cfg.ToDiffOptions()

	assert.NotEmpty(t, opts, "ToDiffOptions should return at least the sort arrays option")
}
