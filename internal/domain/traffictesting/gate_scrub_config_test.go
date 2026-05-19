package traffictesting_test

import (
	"testing"

	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScrubConfig_valid(t *testing.T) {
	cfg, err := traffictesting.NewScrubConfig([]string{"headers.X-Internal-Token"})

	require.NoError(t, err)
	assert.Equal(t, []string{"headers.X-Internal-Token"}, cfg.AdditionalFields)
}

func TestNewScrubConfig_trims_whitespace(t *testing.T) {
	cfg, err := traffictesting.NewScrubConfig([]string{"  headers.X-Token  "})

	require.NoError(t, err)
	assert.Equal(t, []string{"headers.X-Token"}, cfg.AdditionalFields)
}

func TestNewScrubConfig_rejects_empty_field(t *testing.T) {
	_, err := traffictesting.NewScrubConfig([]string{"headers.X-Token", ""})

	require.Error(t, err)
	assert.ErrorIs(t, err, traffictesting.ErrInvalidScrubConfig)
	assert.Contains(t, err.Error(), "scrub_fields[1]")
}

func TestNewScrubConfig_rejects_whitespace_only_field(t *testing.T) {
	_, err := traffictesting.NewScrubConfig([]string{"   "})

	require.Error(t, err)
	assert.ErrorIs(t, err, traffictesting.ErrInvalidScrubConfig)
}

func TestNewScrubConfig_empty_slice(t *testing.T) {
	cfg, err := traffictesting.NewScrubConfig([]string{})

	require.NoError(t, err)
	assert.Empty(t, cfg.AdditionalFields)
}

func TestNewScrubConfig_does_not_mutate_input(t *testing.T) {
	input := []string{"  headers.X-Token  "}
	original := input[0]

	_, _ = traffictesting.NewScrubConfig(input)

	assert.Equal(t, original, input[0], "input slice must not be mutated")
}

func TestDefaultScrubConfig(t *testing.T) {
	cfg := traffictesting.DefaultScrubConfig()

	assert.Empty(t, cfg.AdditionalFields)
	assert.NotNil(t, cfg.AdditionalFields, "should be empty slice, not nil")
}

func TestDefaultScrubFields_returns_copy(t *testing.T) {
	fields1 := traffictesting.DefaultScrubFields()
	fields2 := traffictesting.DefaultScrubFields()

	// Mutate one copy
	fields1[0] = "MUTATED"

	// Second copy should be unaffected
	assert.NotEqual(t, "MUTATED", fields2[0], "DefaultScrubFields must return independent copies")
}

func TestDefaultScrubFields_contains_expected(t *testing.T) {
	fields := traffictesting.DefaultScrubFields()

	assert.Contains(t, fields, "headers.Authorization")
	assert.Contains(t, fields, "headers.Cookie")
	assert.Contains(t, fields, "headers.Set-Cookie")
	assert.Contains(t, fields, "headers.X-Api-Key")
}

func TestScrubConfig_AllFields_merges_defaults_and_additional(t *testing.T) {
	cfg, _ := traffictesting.NewScrubConfig([]string{"headers.X-Internal"})

	all := cfg.AllFields()

	defaults := traffictesting.DefaultScrubFields()
	assert.Len(t, all, len(defaults)+1)
	for _, d := range defaults {
		assert.Contains(t, all, d)
	}
	assert.Contains(t, all, "headers.X-Internal")
}

func TestScrubConfig_AllFields_no_additional(t *testing.T) {
	cfg := traffictesting.DefaultScrubConfig()

	all := cfg.AllFields()

	assert.Equal(t, traffictesting.DefaultScrubFields(), all)
}

func TestScrubConfig_HeaderNames_extracts_header_paths(t *testing.T) {
	cfg, _ := traffictesting.NewScrubConfig([]string{"headers.X-Internal", "body.user.ssn"})

	names := cfg.HeaderNames()

	// Should include defaults + X-Internal, but NOT body.user.ssn
	assert.Contains(t, names, "Authorization")
	assert.Contains(t, names, "Cookie")
	assert.Contains(t, names, "Set-Cookie")
	assert.Contains(t, names, "X-Api-Key")
	assert.Contains(t, names, "X-Internal")
	assert.NotContains(t, names, "body.user.ssn")
	assert.NotContains(t, names, "user.ssn")
}

func TestScrubConfig_HeaderNames_no_header_paths(t *testing.T) {
	cfg, _ := traffictesting.NewScrubConfig([]string{"body.secret"})

	names := cfg.HeaderNames()

	// Should still include the default header names
	defaults := traffictesting.DefaultScrubFields()
	assert.Len(t, names, len(defaults))
}
