package diff_test

import (
	"testing"

	"github.com/pedrobarco/mroki/pkg/diff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanReporter_simple_fields(t *testing.T) {
	a := `{"name":"alice","age":30}`
	b := `{"name":"bob","age":25}`

	result, err := diff.JSON(a, b)

	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// Should not contain Go type annotations
	assert.NotContains(t, result, "float64(")
	assert.NotContains(t, result, "string(")
	assert.NotContains(t, result, "map[string]any")

	// Should contain field names and values
	assert.Contains(t, result, "age")
	assert.Contains(t, result, "name")
	assert.Contains(t, result, "30")
	assert.Contains(t, result, "25")
	assert.Contains(t, result, `"alice"`)
	assert.Contains(t, result, `"bob"`)

	// Should use - and + for diffs
	assert.Contains(t, result, "- ")
	assert.Contains(t, result, "+ ")
}

func TestCleanReporter_nested_objects(t *testing.T) {
	a := `{"user":{"name":"alice","age":30},"meta":{"count":5}}`
	b := `{"user":{"name":"bob","age":25},"meta":{"count":10}}`

	result, err := diff.JSON(a, b)

	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// Should not contain Go type annotations
	assert.NotContains(t, result, "float64(")
	assert.NotContains(t, result, "string(")
	assert.NotContains(t, result, "map[string]any")

	// Should show dotted paths for nested fields
	assert.Contains(t, result, "user.age")
	assert.Contains(t, result, "user.name")
	assert.Contains(t, result, "meta.count")
}

func TestCleanReporter_arrays(t *testing.T) {
	a := `{"items":["apple","banana","cherry"]}`
	b := `{"items":["apple","orange","cherry"]}`

	result, err := diff.JSON(a, b)

	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// Should not contain Go type annotations
	assert.NotContains(t, result, "string(")
	assert.NotContains(t, result, "[]any")

	// Should show array index notation
	assert.Contains(t, result, "items[1]")
	assert.Contains(t, result, `"banana"`)
	assert.Contains(t, result, `"orange"`)
}

func TestCleanReporter_mixed_types(t *testing.T) {
	a := `{"str":"hello","num":42,"bool":true,"null":null}`
	b := `{"str":"world","num":99,"bool":false,"null":null}`

	result, err := diff.JSON(a, b)

	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// Should not contain Go type annotations
	assert.NotContains(t, result, "string(")
	assert.NotContains(t, result, "float64(")
	assert.NotContains(t, result, "bool(")

	// Should contain clean values
	assert.Contains(t, result, `"hello"`)
	assert.Contains(t, result, `"world"`)
	assert.Contains(t, result, "42")
	assert.Contains(t, result, "99")
	assert.Contains(t, result, "true")
	assert.Contains(t, result, "false")

	// null field shouldn't appear (no difference)
	assert.NotContains(t, result, "null")
}

func TestCleanReporter_complex_nested_structure(t *testing.T) {
	a := `{
		"statusCode": 200,
		"headers": {
			"Content-Type": ["application/json"],
			"X-Custom": ["value1", "value2"]
		},
		"body": {
			"status": "ok",
			"data": {
				"user": "alice",
				"age": 30,
				"tags": ["admin", "active"]
			}
		}
	}`

	b := `{
		"statusCode": 500,
		"headers": {
			"Content-Type": ["application/json"],
			"X-Custom": ["value1", "value3"]
		},
		"body": {
			"status": "error",
			"data": {
				"user": "bob",
				"age": 25,
				"tags": ["admin", "inactive"]
			}
		}
	}`

	result, err := diff.JSON(a, b)

	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// Should not contain any Go type annotations
	assert.NotContains(t, result, "float64(")
	assert.NotContains(t, result, "string(")
	assert.NotContains(t, result, "map[string]any")
	assert.NotContains(t, result, "[]any")

	// Should show proper paths for all differences
	assert.Contains(t, result, "statusCode")
	assert.Contains(t, result, "headers.X-Custom[1]")
	assert.Contains(t, result, "body.status")
	assert.Contains(t, result, "body.data.user")
	assert.Contains(t, result, "body.data.age")
	assert.Contains(t, result, "body.data.tags[1]")

	// Should show clean values
	assert.Contains(t, result, "200")
	assert.Contains(t, result, "500")
	assert.Contains(t, result, `"value2"`)
	assert.Contains(t, result, `"value3"`)
	assert.Contains(t, result, `"ok"`)
	assert.Contains(t, result, `"error"`)
	assert.Contains(t, result, `"alice"`)
	assert.Contains(t, result, `"bob"`)
	assert.Contains(t, result, "30")
	assert.Contains(t, result, "25")
	assert.Contains(t, result, `"active"`)
	assert.Contains(t, result, `"inactive"`)
}

func TestCleanReporter_floats_formatted_cleanly(t *testing.T) {
	// Whole number floats should render without decimals
	a := `{"price":10.0,"tax":1.5}`
	b := `{"price":20.0,"tax":2.7}`

	result, err := diff.JSON(a, b)

	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// Should not contain float64() wrapper
	assert.NotContains(t, result, "float64(")

	// Whole numbers should appear without .0
	assert.Contains(t, result, "- 10\n")
	assert.Contains(t, result, "+ 20\n")

	// Decimals should be preserved
	assert.Contains(t, result, "1.5")
	assert.Contains(t, result, "2.7")
}

func TestCleanReporter_empty_diff_returns_empty_string(t *testing.T) {
	a := `{"name":"alice","age":30}`
	b := `{"name":"alice","age":30}`

	result, err := diff.JSON(a, b)

	require.NoError(t, err)
	assert.Empty(t, result, "identical JSON should produce empty diff")
}

func TestCleanReporter_with_ignored_fields(t *testing.T) {
	a := `{"name":"alice","timestamp":"2024-01-01T10:00:00Z"}`
	b := `{"name":"alice","timestamp":"2024-01-01T11:00:00Z"}`

	result, err := diff.JSON(a, b, diff.WithIgnoredFields("timestamp"))

	require.NoError(t, err)
	assert.Empty(t, result, "should be identical when ignoring timestamp")
}

func TestCleanReporter_with_included_fields(t *testing.T) {
	a := `{"name":"alice","age":30,"email":"alice@example.com"}`
	b := `{"name":"bob","age":25,"email":"alice@example.com"}`

	result, err := diff.JSON(a, b, diff.WithIncludedFields("email"))

	require.NoError(t, err)
	assert.Empty(t, result, "should be identical when only comparing email")
}

func TestCleanReporter_output_format_readable(t *testing.T) {
	a := `{"user":"alice"}`
	b := `{"user":"bob"}`

	result, err := diff.JSON(a, b)

	require.NoError(t, err)

	// Output should be formatted with indentation and clear diff markers
	lines := splitLines(result)
	assert.GreaterOrEqual(t, len(lines), 3, "should have at least 3 lines (path, -, +)")

	// Check that lines are indented
	for _, line := range lines {
		if line != "" {
			assert.True(t, line[:2] == "  ", "lines should be indented with 2 spaces")
		}
	}
}

// Helper function to split string into lines
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
