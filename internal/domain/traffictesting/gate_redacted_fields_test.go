package traffictesting_test

import (
	"testing"

	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRedactedFields_valid(t *testing.T) {
	rf, err := traffictesting.NewRedactedFields([]string{"headers.X-Internal-Token"})

	require.NoError(t, err)
	assert.Equal(t, []string{"headers.X-Internal-Token"}, rf.AdditionalFields)
}

func TestNewRedactedFields_trims_whitespace(t *testing.T) {
	rf, err := traffictesting.NewRedactedFields([]string{"  headers.X-Token  "})

	require.NoError(t, err)
	assert.Equal(t, []string{"headers.X-Token"}, rf.AdditionalFields)
}

func TestNewRedactedFields_rejects_empty_field(t *testing.T) {
	_, err := traffictesting.NewRedactedFields([]string{"headers.X-Token", ""})

	require.Error(t, err)
	assert.ErrorIs(t, err, traffictesting.ErrInvalidRedactedFields)
	assert.Contains(t, err.Error(), "redacted_fields[1]")
}

func TestNewRedactedFields_rejects_whitespace_only_field(t *testing.T) {
	_, err := traffictesting.NewRedactedFields([]string{"   "})

	require.Error(t, err)
	assert.ErrorIs(t, err, traffictesting.ErrInvalidRedactedFields)
}

func TestNewRedactedFields_rejects_field_without_prefix(t *testing.T) {
	_, err := traffictesting.NewRedactedFields([]string{"Authorization"})

	require.Error(t, err)
	assert.ErrorIs(t, err, traffictesting.ErrInvalidRedactedFields)
	assert.Contains(t, err.Error(), "must start with")
}

func TestNewRedactedFields_accepts_body_prefix(t *testing.T) {
	rf, err := traffictesting.NewRedactedFields([]string{"body.user.password"})

	require.NoError(t, err)
	assert.Equal(t, []string{"body.user.password"}, rf.AdditionalFields)
}

func TestNewRedactedFields_rejects_bare_headers_prefix(t *testing.T) {
	_, err := traffictesting.NewRedactedFields([]string{"headers."})

	require.Error(t, err)
	assert.ErrorIs(t, err, traffictesting.ErrInvalidRedactedFields)
	assert.Contains(t, err.Error(), "must specify a field name")
}

func TestNewRedactedFields_rejects_bare_body_prefix(t *testing.T) {
	_, err := traffictesting.NewRedactedFields([]string{"body."})

	require.Error(t, err)
	assert.ErrorIs(t, err, traffictesting.ErrInvalidRedactedFields)
	assert.Contains(t, err.Error(), "must specify a field path")
}

func TestNewRedactedFields_empty_slice(t *testing.T) {
	rf, err := traffictesting.NewRedactedFields([]string{})

	require.NoError(t, err)
	assert.Empty(t, rf.AdditionalFields)
}

func TestNewRedactedFields_does_not_mutate_input(t *testing.T) {
	input := []string{"  headers.X-Token  "}
	original := input[0]

	_, _ = traffictesting.NewRedactedFields(input)

	assert.Equal(t, original, input[0], "input slice must not be mutated")
}

func TestDefaultRedactedFields(t *testing.T) {
	rf := traffictesting.DefaultRedactedFields()

	assert.Empty(t, rf.AdditionalFields)
	assert.NotNil(t, rf.AdditionalFields, "should be empty slice, not nil")
}

func TestDefaultRedactedFieldList_returns_copy(t *testing.T) {
	fields1 := traffictesting.DefaultRedactedFieldList()
	fields2 := traffictesting.DefaultRedactedFieldList()

	// Mutate one copy
	fields1[0] = "MUTATED"

	// Second copy should be unaffected
	assert.NotEqual(t, "MUTATED", fields2[0], "DefaultRedactedFieldList must return independent copies")
}

func TestDefaultRedactedFieldList_contains_expected(t *testing.T) {
	fields := traffictesting.DefaultRedactedFieldList()

	assert.Contains(t, fields, "headers.Authorization")
	assert.Contains(t, fields, "headers.Cookie")
	assert.Contains(t, fields, "headers.Set-Cookie")
	assert.Contains(t, fields, "headers.X-Api-Key")
}

func TestRedactedFields_AllFields_merges_defaults_and_additional(t *testing.T) {
	rf, _ := traffictesting.NewRedactedFields([]string{"headers.X-Internal"})

	all := rf.AllFields()

	defaults := traffictesting.DefaultRedactedFieldList()
	assert.Len(t, all, len(defaults)+1)
	for _, d := range defaults {
		assert.Contains(t, all, d)
	}
	assert.Contains(t, all, "headers.X-Internal")
}

func TestRedactedFields_AllFields_no_additional(t *testing.T) {
	rf := traffictesting.DefaultRedactedFields()

	all := rf.AllFields()

	assert.Equal(t, traffictesting.DefaultRedactedFieldList(), all)
}

func TestRedactedFields_AllFields_deduplicates_header_case_insensitive(t *testing.T) {
	// "headers.authorization" should be deduplicated against default "headers.Authorization"
	rf, err := traffictesting.NewRedactedFields([]string{"headers.authorization"})
	require.NoError(t, err)

	all := rf.AllFields()

	// Should NOT contain both — the default wins, the additional is skipped
	defaults := traffictesting.DefaultRedactedFieldList()
	assert.Len(t, all, len(defaults), "case-insensitive duplicate should be skipped")
	assert.Contains(t, all, "headers.Authorization")
}

func TestRedactedFields_AllFields_body_dedup_is_case_sensitive(t *testing.T) {
	// body paths should be exact-match — "body.User" and "body.user" are different
	rf, err := traffictesting.NewRedactedFields([]string{"body.User", "body.user"})
	require.NoError(t, err)

	all := rf.AllFields()

	defaults := traffictesting.DefaultRedactedFieldList()
	assert.Len(t, all, len(defaults)+2, "body paths are case-sensitive, both should be kept")
}
