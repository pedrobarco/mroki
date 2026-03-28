package diff_test

import (
	"testing"

	"github.com/pedrobarco/mroki/pkg/diff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPatchReporter_simple_fields(t *testing.T) {
	a := `{"name":"alice","age":30}`
	b := `{"name":"bob","age":25}`

	ops, err := diff.JSON(a, b)

	require.NoError(t, err)
	assert.Len(t, ops, 2)

	paths := map[string]string{}
	for _, op := range ops {
		assert.Equal(t, "replace", op.Op)
		paths[op.Path] = op.Op
	}
	assert.Contains(t, paths, "/name")
	assert.Contains(t, paths, "/age")
}

func TestPatchReporter_nested_objects(t *testing.T) {
	a := `{"user":{"name":"alice","age":30},"meta":{"count":5}}`
	b := `{"user":{"name":"bob","age":25},"meta":{"count":10}}`

	ops, err := diff.JSON(a, b)

	require.NoError(t, err)
	assert.Len(t, ops, 3)

	paths := map[string]bool{}
	for _, op := range ops {
		paths[op.Path] = true
	}
	assert.True(t, paths["/user/name"])
	assert.True(t, paths["/user/age"])
	assert.True(t, paths["/meta/count"])
}

func TestPatchReporter_arrays(t *testing.T) {
	// Arrays are always sorted before comparison.
	// a sorted: ["apple","banana","cherry"]
	// b sorted: ["apple","cherry","orange"]
	// Produces 2 replacements: banana→cherry at /items/1, cherry→orange at /items/2
	a := `{"items":["apple","banana","cherry"]}`
	b := `{"items":["apple","orange","cherry"]}`

	ops, err := diff.JSON(a, b)

	require.NoError(t, err)
	assert.Len(t, ops, 2)
	assert.Equal(t, "replace", ops[0].Op)
	assert.Equal(t, "/items/1", ops[0].Path)
	assert.Equal(t, "cherry", ops[0].Value)
	assert.Equal(t, "replace", ops[1].Op)
	assert.Equal(t, "/items/2", ops[1].Path)
	assert.Equal(t, "orange", ops[1].Value)
}

func TestPatchReporter_mixed_types(t *testing.T) {
	a := `{"str":"hello","num":42,"bool":true,"null":null}`
	b := `{"str":"world","num":99,"bool":false,"null":null}`

	ops, err := diff.JSON(a, b)

	require.NoError(t, err)
	assert.Len(t, ops, 3, "null field has no difference and should not appear")

	paths := map[string]bool{}
	for _, op := range ops {
		paths[op.Path] = true
	}
	assert.True(t, paths["/str"])
	assert.True(t, paths["/num"])
	assert.True(t, paths["/bool"])
}

func TestPatchReporter_complex_nested_structure(t *testing.T) {
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

	ops, err := diff.JSON(a, b)

	require.NoError(t, err)
	assert.NotEmpty(t, ops)

	paths := map[string]bool{}
	for _, op := range ops {
		assert.Equal(t, "replace", op.Op)
		paths[op.Path] = true
	}

	assert.True(t, paths["/statusCode"])
	assert.True(t, paths["/headers/X-Custom/1"])
	assert.True(t, paths["/body/status"])
	assert.True(t, paths["/body/data/user"])
	assert.True(t, paths["/body/data/age"])
	assert.True(t, paths["/body/data/tags/1"])
}

func TestPatchReporter_empty_diff_returns_empty_ops(t *testing.T) {
	a := `{"name":"alice","age":30}`
	b := `{"name":"alice","age":30}`

	ops, err := diff.JSON(a, b)

	require.NoError(t, err)
	assert.Empty(t, ops, "identical JSON should produce empty ops")
}

func TestPatchReporter_with_ignored_fields(t *testing.T) {
	a := `{"name":"alice","timestamp":"2024-01-01T10:00:00Z"}`
	b := `{"name":"alice","timestamp":"2024-01-01T11:00:00Z"}`

	ops, err := diff.JSON(a, b, diff.WithIgnoredFields("timestamp"))

	require.NoError(t, err)
	assert.Empty(t, ops, "should be identical when ignoring timestamp")
}

func TestPatchReporter_with_included_fields(t *testing.T) {
	a := `{"name":"alice","age":30,"email":"alice@example.com"}`
	b := `{"name":"bob","age":25,"email":"alice@example.com"}`

	ops, err := diff.JSON(a, b, diff.WithIncludedFields("email"))

	require.NoError(t, err)
	assert.Empty(t, ops, "should be identical when only comparing email")
}

func TestPatchReporter_json_pointer_format(t *testing.T) {
	a := `{"user":"alice"}`
	b := `{"user":"bob"}`

	ops, err := diff.JSON(a, b)

	require.NoError(t, err)
	require.Len(t, ops, 1)
	assert.Equal(t, "replace", ops[0].Op)
	assert.Equal(t, "/user", ops[0].Path)
	assert.Equal(t, "bob", ops[0].Value)
}


func TestPatchReporter_add_operation(t *testing.T) {
	a := `{"name":"alice"}`
	b := `{"name":"alice","age":30}`

	ops, err := diff.JSON(a, b)

	require.NoError(t, err)
	require.Len(t, ops, 1)
	assert.Equal(t, "add", ops[0].Op)
	assert.Equal(t, "/age", ops[0].Path)
}

func TestPatchReporter_remove_operation(t *testing.T) {
	a := `{"name":"alice","age":30}`
	b := `{"name":"alice"}`

	ops, err := diff.JSON(a, b)

	require.NoError(t, err)
	require.Len(t, ops, 1)
	assert.Equal(t, "remove", ops[0].Op)
	assert.Equal(t, "/age", ops[0].Path)
	assert.Nil(t, ops[0].Value)
}

func TestPatchReporter_null_value(t *testing.T) {
	a := `{"name":"alice"}`
	b := `{"name":null}`

	ops, err := diff.JSON(a, b)

	require.NoError(t, err)
	require.Len(t, ops, 1)
	assert.Equal(t, "replace", ops[0].Op)
	assert.Equal(t, "/name", ops[0].Path)
	assert.Nil(t, ops[0].Value)
}

func TestFormatOps_empty(t *testing.T) {
	result := diff.FormatOps(nil)
	assert.Empty(t, result)

	result = diff.FormatOps([]diff.PatchOp{})
	assert.Empty(t, result)
}

func TestFormatOps_multiple_ops(t *testing.T) {
	ops := []diff.PatchOp{
		{Op: "replace", Path: "/name", Value: "bob"},
		{Op: "add", Path: "/age", Value: float64(30)},
		{Op: "remove", Path: "/old_field"},
	}

	result := diff.FormatOps(ops)

	assert.Contains(t, result, "replace /name")
	assert.Contains(t, result, "add /age")
	assert.Contains(t, result, "remove /old_field")
}

func TestPatchReporter_rfc6901_escaping(t *testing.T) {
	a := `{"a/b":1,"c~d":2}`
	b := `{"a/b":3,"c~d":4}`

	ops, err := diff.JSON(a, b)

	require.NoError(t, err)
	assert.Len(t, ops, 2)

	paths := map[string]bool{}
	for _, op := range ops {
		paths[op.Path] = true
	}
	assert.True(t, paths["/a~1b"], "/ in key should be escaped as ~1")
	assert.True(t, paths["/c~0d"], "~ in key should be escaped as ~0")
}