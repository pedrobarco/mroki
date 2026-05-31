package diff_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/pedrobarco/mroki/pkg/diff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSON_identical_json_returns_empty_diff(t *testing.T) {
	a := `{"key": "value", "nested": {"foo": "bar"}}`
	b := `{"key": "value", "nested": {"foo": "bar"}}`

	ops, err := diff.JSON(a, b)

	assert.NoError(t, err)
	assert.Empty(t, ops)
}

func TestJSON_different_json_returns_diff(t *testing.T) {
	a := `{"key": "value1", "count": 1}`
	b := `{"key": "value2", "count": 2}`

	ops, err := diff.JSON(a, b)

	assert.NoError(t, err)
	assert.NotEmpty(t, ops)
	assert.Len(t, ops, 2)
	for _, op := range ops {
		assert.Equal(t, "replace", op.Op)
	}
}

func TestJSON_invalid_first_input_returns_error(t *testing.T) {
	a := `{invalid json}`
	b := `{"key": "value"}`

	ops, err := diff.JSON(a, b)

	assert.Error(t, err)
	assert.Empty(t, ops)
	assert.Contains(t, err.Error(), "first input")
}

func TestJSON_invalid_second_input_returns_error(t *testing.T) {
	a := `{"key": "value"}`
	b := `{invalid json}`

	ops, err := diff.JSON(a, b)

	assert.Error(t, err)
	assert.Empty(t, ops)
	assert.Contains(t, err.Error(), "second input")
}

// Field filtering tests

func TestJSON_IgnoredFields(t *testing.T) {
	a := `{"name": "John", "age": 30, "timestamp": "2024-01-01T10:00:00Z"}`
	b := `{"name": "John", "age": 30, "timestamp": "2024-01-01T11:00:00Z"}`

	ops, err := diff.JSON(a, b, diff.WithIgnoredFields("timestamp"))

	assert.NoError(t, err)
	assert.Empty(t, ops, "should be identical when ignoring timestamp")
}

func TestJSON_IgnoredFields_StillDetectsDifferences(t *testing.T) {
	a := `{"name": "John", "age": 30, "timestamp": "2024-01-01T10:00:00Z"}`
	b := `{"name": "Jane", "age": 30, "timestamp": "2024-01-01T11:00:00Z"}`

	ops, err := diff.JSON(a, b, diff.WithIgnoredFields("timestamp"))

	assert.NoError(t, err)
	assert.Len(t, ops, 1)
	assert.Equal(t, "replace", ops[0].Op)
	assert.Equal(t, "/name", ops[0].Path)
}

func TestJSON_IncludedFields(t *testing.T) {
	a := `{"name": "John", "age": 30, "email": "john@example.com", "timestamp": "2024-01-01"}`
	b := `{"name": "John", "age": 35, "email": "john@example.com", "timestamp": "2024-01-02"}`

	ops, err := diff.JSON(a, b, diff.WithIncludedFields("name", "email"))

	assert.NoError(t, err)
	assert.Empty(t, ops, "should be identical when only comparing name and email")
}

func TestJSON_IncludedFields_DetectsDifferences(t *testing.T) {
	a := `{"name": "John", "age": 30, "email": "john@example.com"}`
	b := `{"name": "Jane", "age": 30, "email": "john@example.com"}`

	ops, err := diff.JSON(a, b, diff.WithIncludedFields("name", "email"))

	assert.NoError(t, err)
	assert.Len(t, ops, 1)
	assert.Equal(t, "/name", ops[0].Path)
}

func TestJSON_NestedFields(t *testing.T) {
	a := `{"user": {"name": "John", "profile": {"age": 30, "email": "john@example.com"}}, "timestamp": "2024-01-01"}`
	b := `{"user": {"name": "John", "profile": {"age": 35, "email": "john@example.com"}}, "timestamp": "2024-01-02"}`

	ops, err := diff.JSON(a, b, diff.WithIncludedFields("user.name", "user.profile.email"))

	assert.NoError(t, err)
	assert.Empty(t, ops, "should be identical when only comparing name and email")
}

func TestJSON_ArrayWildcard_IgnoredFields(t *testing.T) {
	a := `{"users": [{"name": "John", "created_at": "2024-01-01"}, {"name": "Jane", "created_at": "2024-01-02"}]}`
	b := `{"users": [{"name": "John", "created_at": "2024-02-01"}, {"name": "Jane", "created_at": "2024-02-02"}]}`

	ops, err := diff.JSON(a, b, diff.WithIgnoredFields("users.#.created_at"))

	assert.NoError(t, err)
	assert.Empty(t, ops, "should be identical when ignoring created_at in all array elements")
}

func TestJSON_HybridStrategy(t *testing.T) {
	a := `{"name": "John", "age": 30, "email": "john@example.com"}`
	b := `{"name": "Jane", "age": 30, "email": "jane@example.com"}`

	ops, err := diff.JSON(a, b,
		diff.WithIncludedFields("name", "email"),
		diff.WithIgnoredFields("email"),
	)

	assert.NoError(t, err)
	assert.Len(t, ops, 1, "should detect name difference only")
	assert.Equal(t, "/name", ops[0].Path)
}

func TestJSON_NoFiltering(t *testing.T) {
	a := `{"key": "value1"}`
	b := `{"key": "value2"}`

	ops, err := diff.JSON(a, b)

	assert.NoError(t, err)
	assert.Len(t, ops, 1)
	assert.Equal(t, "replace", ops[0].Op)
	assert.Equal(t, "/key", ops[0].Path)
}

func TestJSON_SortArrays_Numbers(t *testing.T) {
	a := `{"items": [1, 2, 3]}`
	b := `{"items": [3, 1, 2]}`

	ops, err := diff.JSON(a, b, diff.WithSortArrays(true))
	assert.NoError(t, err)
	assert.Empty(t, ops, "number arrays with same elements in different order should be equal")
}

func TestJSON_SortArrays_Strings(t *testing.T) {
	a := `{"tags": ["alpha", "beta", "gamma"]}`
	b := `{"tags": ["gamma", "alpha", "beta"]}`

	ops, err := diff.JSON(a, b, diff.WithSortArrays(true))
	assert.NoError(t, err)
	assert.Empty(t, ops, "string arrays with same elements in different order should be equal")
}

func TestJSON_SortArrays_Objects(t *testing.T) {
	a := `{"users": [{"name": "Alice", "age": 30}, {"name": "Bob", "age": 25}]}`
	b := `{"users": [{"name": "Bob", "age": 25}, {"name": "Alice", "age": 30}]}`

	ops, err := diff.JSON(a, b, diff.WithSortArrays(true))
	assert.NoError(t, err)
	assert.Empty(t, ops, "arrays of objects with same elements in different order should be equal")
}

func TestJSON_SortArrays_ObjectsDifferentKeyOrder(t *testing.T) {
	a := `{"users": [{"name": "Alice", "age": 30}, {"name": "Bob", "age": 25}]}`
	b := `{"users": [{"age": 25, "name": "Bob"}, {"age": 30, "name": "Alice"}]}`

	ops, err := diff.JSON(a, b, diff.WithSortArrays(true))
	assert.NoError(t, err)
	assert.Empty(t, ops, "arrays of objects with different key order and array order should be equal")
}

func TestJSON_SortArrays_NestedArrays(t *testing.T) {
	a := `{"matrix": [[1, 2], [3, 4]]}`
	b := `{"matrix": [[3, 4], [1, 2]]}`

	ops, err := diff.JSON(a, b, diff.WithSortArrays(true))
	assert.NoError(t, err)
	assert.Empty(t, ops, "nested arrays with same elements in different order should be equal")
}

func TestJSON_SortArrays_ActualDifference(t *testing.T) {
	a := `{"items": [1, 2, 3]}`
	b := `{"items": [1, 2, 4]}`

	ops, err := diff.JSON(a, b, diff.WithSortArrays(true))
	assert.NoError(t, err)
	assert.NotEmpty(t, ops, "arrays with genuinely different elements should produce a diff")
}

func TestJSON_DefaultNoSort_PositionalDiff(t *testing.T) {
	a := `{"items": [1, 2, 3]}`
	b := `{"items": [3, 1, 2]}`

	ops, err := diff.JSON(a, b)
	assert.NoError(t, err)
	assert.NotEmpty(t, ops, "without sort_arrays, reordered arrays should produce diffs (positional comparison)")
}

func TestJSON_SortArrays_IndicesMatchSortedData(t *testing.T) {
	// Verify that patch op indices correspond to sorted array positions,
	// avoiding mismatching array indices between backend and frontend.
	a := `{"items": [3, 1, 2]}`
	b := `{"items": [3, 1, 4]}`

	// Sorted: [1, 2, 3] vs [1, 3, 4] → index 1: 2→3, index 2: 3→4
	ops, err := diff.JSON(a, b, diff.WithSortArrays(true))
	assert.NoError(t, err)
	require.Len(t, ops, 2)
	assert.Equal(t, "/items/1", ops[0].Path)
	assert.Equal(t, float64(3), ops[0].Value)
	assert.Equal(t, "/items/2", ops[1].Path)
	assert.Equal(t, float64(4), ops[1].Value)
}

func TestJSON_ObjectKeyOrder_NoDiff(t *testing.T) {
	a := `{"a": 1, "b": 2, "c": 3}`
	b := `{"c": 3, "a": 1, "b": 2}`

	ops, err := diff.JSON(a, b)
	assert.NoError(t, err)
	assert.Empty(t, ops, "objects with same keys in different order should produce no diff")
}

func TestJSON_ObjectKeyOrder_Nested(t *testing.T) {
	a := `{"user": {"name": "Alice", "email": "alice@example.com"}, "meta": {"version": 1}}`
	b := `{"meta": {"version": 1}, "user": {"email": "alice@example.com", "name": "Alice"}}`

	ops, err := diff.JSON(a, b)
	assert.NoError(t, err)
	assert.Empty(t, ops, "nested objects with different key order should produce no diff")
}

func TestJSON_WithFloatTolerance(t *testing.T) {
	a := `{"value": 1.0001}`
	b := `{"value": 1.0002}`

	// Without tolerance - should detect difference
	ops, err := diff.JSON(a, b)
	assert.NoError(t, err)
	assert.NotEmpty(t, ops, "should detect difference without tolerance")

	// With tolerance - should be identical
	ops, err = diff.JSON(a, b, diff.WithFloatTolerance(0.001))
	assert.NoError(t, err)
	assert.Empty(t, ops, "should be identical with tolerance")
}

func TestJSON_MultipleOptions(t *testing.T) {
	a := `{"user": {"name": "John", "ssn": "123-45-6789", "balance": 100.001}, "timestamp": "2024-01-01"}`
	b := `{"user": {"name": "John", "ssn": "987-65-4321", "balance": 100.002}, "timestamp": "2024-01-02"}`

	ops, err := diff.JSON(a, b,
		diff.WithIncludedFields("user"),
		diff.WithIgnoredFields("user.ssn"),
		diff.WithFloatTolerance(0.01),
	)

	assert.NoError(t, err)
	assert.Empty(t, ops, "should be identical with multiple filters")
}

// Benchmark tests

func BenchmarkJSON_Baseline(b *testing.B) {
	a := `{"name":"John","age":30,"email":"john@example.com","address":"123 Main St","timestamp":"2024-01-01T10:00:00Z"}`
	same := `{"name":"John","age":30,"email":"john@example.com","address":"123 Main St","timestamp":"2024-01-01T10:00:00Z"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = diff.JSON(a, same)
	}
}

func BenchmarkJSON_IgnoredFields(b *testing.B) {
	a := `{"name":"John","age":30,"email":"john@example.com","address":"123 Main St","timestamp":"2024-01-01T10:00:00Z","request_id":"abc123"}`
	same := `{"name":"John","age":30,"email":"john@example.com","address":"123 Main St","timestamp":"2024-01-01T10:00:00Z","request_id":"abc123"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = diff.JSON(a, same, diff.WithIgnoredFields("timestamp", "request_id"))
	}
}

func BenchmarkJSON_IncludedFields(b *testing.B) {
	a := `{"name":"John","age":30,"email":"john@example.com","address":"123 Main St","timestamp":"2024-01-01T10:00:00Z","request_id":"abc123"}`
	same := `{"name":"John","age":30,"email":"john@example.com","address":"123 Main St","timestamp":"2024-01-01T10:00:00Z","request_id":"abc123"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = diff.JSON(a, same, diff.WithIncludedFields("name", "email"))
	}
}

func BenchmarkJSON_WithDifferences(b *testing.B) {
	// Benchmark with actual differences (more realistic use case)
	a := `{"name":"Alice","age":30,"email":"alice@example.com","address":"123 Main St","city":"NYC"}`
	different := `{"name":"Bob","age":25,"email":"bob@example.com","address":"456 Elm St","city":"LA"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = diff.JSON(a, different)
	}
}

func BenchmarkJSON_ComplexNested(b *testing.B) {
	// Benchmark with complex nested structure (proxy response-like)
	a := `{
		"statusCode": 200,
		"headers": {
			"Content-Type": ["application/json"],
			"X-Custom": ["value1", "value2", "value3"]
		},
		"body": {
			"status": "ok",
			"data": {
				"user": "alice",
				"age": 30,
				"tags": ["admin", "active"],
				"metadata": {"id": "123", "version": "1.0"}
			}
		}
	}`
	c := `{
		"statusCode": 500,
		"headers": {
			"Content-Type": ["application/json"],
			"X-Custom": ["value1", "value3", "value4"]
		},
		"body": {
			"status": "error",
			"data": {
				"user": "bob",
				"age": 25,
				"tags": ["admin", "inactive"],
				"metadata": {"id": "456", "version": "2.0"}
			}
		}
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = diff.JSON(a, c)
	}
}

// --- diff.Parsed tests ---

func TestParsed_identical_returns_empty_diff(t *testing.T) {
	a := map[string]any{"key": "value", "nested": map[string]any{"foo": "bar"}}
	b := map[string]any{"key": "value", "nested": map[string]any{"foo": "bar"}}

	ops, err := diff.Parsed(a, b)

	require.NoError(t, err)
	assert.Empty(t, ops)
}

func TestParsed_different_returns_diff(t *testing.T) {
	a := map[string]any{"key": "value1", "count": float64(1)}
	b := map[string]any{"key": "value2", "count": float64(2)}

	ops, err := diff.Parsed(a, b)

	require.NoError(t, err)
	assert.Len(t, ops, 2)
	for _, op := range ops {
		assert.Equal(t, "replace", op.Op)
	}
}

func TestParsed_nil_input_returns_error(t *testing.T) {
	_, err := diff.Parsed(nil, map[string]any{})
	assert.Error(t, err)

	_, err = diff.Parsed(map[string]any{}, nil)
	assert.Error(t, err)
}

func TestParsed_with_ignored_fields(t *testing.T) {
	a := map[string]any{"name": "John", "timestamp": "2024-01-01"}
	b := map[string]any{"name": "John", "timestamp": "2024-01-02"}

	ops, err := diff.Parsed(a, b, diff.WithIgnoredFields("timestamp"))

	require.NoError(t, err)
	assert.Empty(t, ops, "should have no diff when differing field is ignored")
}

func TestParsed_with_included_fields(t *testing.T) {
	a := map[string]any{"name": "John", "age": float64(30)}
	b := map[string]any{"name": "John", "age": float64(31)}

	ops, err := diff.Parsed(a, b, diff.WithIncludedFields("name"))

	require.NoError(t, err)
	assert.Empty(t, ops, "should have no diff when only matching fields are included")
}

func TestParsed_with_sort_arrays(t *testing.T) {
	a := map[string]any{"items": []any{float64(3), float64(1), float64(2)}}
	b := map[string]any{"items": []any{float64(2), float64(3), float64(1)}}

	ops, err := diff.Parsed(a, b, diff.WithSortArrays(true))

	require.NoError(t, err)
	assert.Empty(t, ops, "sorted arrays with same elements should produce no diff")
}

func TestParsed_without_sort_arrays(t *testing.T) {
	a := map[string]any{"items": []any{float64(3), float64(1), float64(2)}}
	b := map[string]any{"items": []any{float64(2), float64(3), float64(1)}}

	ops, err := diff.Parsed(a, b)

	require.NoError(t, err)
	assert.NotEmpty(t, ops, "without sort_arrays, reordered arrays should produce diffs")
}

func TestSortArraysInTree_sorts_flat_array(t *testing.T) {
	v := map[string]any{"items": []any{float64(3), float64(1), float64(2)}}
	result := diff.SortArraysInTree(v)

	sorted := result.(map[string]any)["items"].([]any)
	assert.Equal(t, []any{float64(1), float64(2), float64(3)}, sorted)
}

func TestSortArraysInTree_sorts_string_array(t *testing.T) {
	v := []any{"cherry", "apple", "banana"}
	result := diff.SortArraysInTree(v)

	assert.Equal(t, []any{"apple", "banana", "cherry"}, result)
}

func TestSortArraysInTree_sorts_nested_arrays(t *testing.T) {
	v := map[string]any{
		"matrix": []any{
			[]any{float64(3), float64(1)},
			[]any{float64(1), float64(2)},
		},
	}
	result := diff.SortArraysInTree(v)

	matrix := result.(map[string]any)["matrix"].([]any)
	// Inner arrays sorted first: [3,1]→[1,3] and [1,2]→[1,2]
	// Then outer sorted by canonical JSON key: [1,2] < [1,3]
	assert.Equal(t, []any{float64(1), float64(2)}, matrix[0])
	assert.Equal(t, []any{float64(1), float64(3)}, matrix[1])
}

func TestSortArraysInTree_preserves_non_array_values(t *testing.T) {
	v := map[string]any{
		"name":   "test",
		"count":  float64(42),
		"active": true,
		"data":   nil,
	}
	result := diff.SortArraysInTree(v)

	m := result.(map[string]any)
	assert.Equal(t, "test", m["name"])
	assert.Equal(t, float64(42), m["count"])
	assert.Equal(t, true, m["active"])
	assert.Nil(t, m["data"])
}

func TestSortArraysInTree_sorts_arrays_inside_objects_in_arrays(t *testing.T) {
	v := []any{
		map[string]any{"tags": []any{"b", "a"}},
		map[string]any{"tags": []any{"d", "c"}},
	}
	diff.SortArraysInTree(v)

	// Inner arrays should be sorted
	assert.Equal(t, []any{"a", "b"}, v[0].(map[string]any)["tags"])
	assert.Equal(t, []any{"c", "d"}, v[1].(map[string]any)["tags"])
}

// TestBuildEnvelope_MatchesJsonPath verifies that BuildEnvelope produces the
// same Go value tree as the current json.Marshal → gjson.ParseBytes path.
func TestBuildEnvelope_MatchesJsonPath(t *testing.T) {
	headers := http.Header{
		"Content-Type": {"application/json"},
		"X-Multi":      {"val1", "val2"},
	}
	bodyJSON := `{"user": {"name": "Alice"}, "count": 42}`
	var bodyParsed any
	require.NoError(t, json.Unmarshal([]byte(bodyJSON), &bodyParsed))

	envelope := diff.BuildEnvelope(200, headers, bodyParsed)

	// Verify structure
	assert.Equal(t, float64(200), envelope["statusCode"])
	assert.NotNil(t, envelope["headers"])
	assert.NotNil(t, envelope["body"])

	// Verify header values are []any (matching json.Marshal → parse shape)
	h := envelope["headers"].(map[string]any)
	ct := h["Content-Type"].([]any)
	assert.Equal(t, []any{"application/json"}, ct)
	multi := h["X-Multi"].([]any)
	assert.Equal(t, []any{"val1", "val2"}, multi)

	// Verify body is the parsed tree
	body := envelope["body"].(map[string]any)
	user := body["user"].(map[string]any)
	assert.Equal(t, "Alice", user["name"])
}

func TestBuildEnvelope_NilBody(t *testing.T) {
	envelope := diff.BuildEnvelope(204, http.Header{}, nil)

	assert.Equal(t, float64(204), envelope["statusCode"])
	assert.Nil(t, envelope["body"])
}

// TestParsed_EquivalenceWithJSON verifies that diff.Parsed produces the same
// PatchOps as diff.JSON for the same logical input.
func TestParsed_EquivalenceWithJSON(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		opts []diff.Option
	}{
		{
			name: "simple replace",
			a:    `{"name":"Alice","age":30}`,
			b:    `{"name":"Bob","age":30}`,
		},
		{
			name: "add field",
			a:    `{"name":"Alice"}`,
			b:    `{"name":"Alice","age":30}`,
		},
		{
			name: "remove field",
			a:    `{"name":"Alice","age":30}`,
			b:    `{"name":"Alice"}`,
		},
		{
			name: "nested change",
			a:    `{"user":{"name":"Alice","profile":{"age":30}}}`,
			b:    `{"user":{"name":"Alice","profile":{"age":31}}}`,
		},
		{
			name: "with ignored fields",
			a:    `{"name":"Alice","ts":"2024-01-01","age":30}`,
			b:    `{"name":"Bob","ts":"2024-01-02","age":31}`,
			opts: []diff.Option{diff.WithIgnoredFields("ts")},
		},
		{
			name: "identical",
			a:    `{"x":1,"y":2}`,
			b:    `{"x":1,"y":2}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// diff.JSON path
			jsonOps, err := diff.JSON(tt.a, tt.b, tt.opts...)
			require.NoError(t, err)

			// diff.Parsed path
			var aTree, bTree any
			require.NoError(t, json.Unmarshal([]byte(tt.a), &aTree))
			require.NoError(t, json.Unmarshal([]byte(tt.b), &bTree))
			parsedOps, err := diff.Parsed(aTree, bTree, tt.opts...)
			require.NoError(t, err)

			// Same number of ops
			assert.Equal(t, len(jsonOps), len(parsedOps), "op count mismatch")

			// Same ops (order may vary, so compare sets)
			jsonSet := opsToSet(jsonOps)
			parsedSet := opsToSet(parsedOps)
			assert.Equal(t, jsonSet, parsedSet)
		})
	}
}

func opsToSet(ops []diff.PatchOp) map[string]string {
	s := make(map[string]string, len(ops))
	for _, op := range ops {
		s[op.Path] = op.Op
	}
	return s
}

// --- Pipeline benchmarks: diff.JSON (bytes) vs BuildEnvelope + diff.Parsed (trees) ---
//
// These benchmarks compare the full diff pipeline for both approaches.
// diff.JSON path:  json.Marshal(headers) → fmt.Sprintf envelope → diff.JSON
// diff.Parsed path: json.Unmarshal body (simulating redactor) → BuildEnvelope → diff.Parsed
//
// Run with: go test ./pkg/diff/... -bench=BenchmarkPipeline -benchmem

// makeBody generates a JSON body with n top-level keys for benchmarking.
func makeBody(n int) (string, map[string]any) {
	m := make(map[string]any, n)
	for i := 0; i < n; i++ {
		m[fmt.Sprintf("field_%d", i)] = fmt.Sprintf("value_%d", i)
	}
	b, _ := json.Marshal(m)
	return string(b), m
}

// makeHeaders generates an http.Header with n entries.
func makeHeaders(n int) http.Header {
	h := make(http.Header, n)
	for i := 0; i < n; i++ {
		h.Set(fmt.Sprintf("X-Header-%d", i), fmt.Sprintf("val-%d", i))
	}
	return h
}

func jsonEnvelope(status int, headers http.Header, body string) string {
	h, _ := json.Marshal(headers)
	return fmt.Sprintf(`{"statusCode": %d, "headers": %s, "body": %s}`, status, h, body)
}

func benchmarkPipelineJSON(b *testing.B, bodyStr string, headers http.Header, opts ...diff.Option) {
	a := jsonEnvelope(200, headers, bodyStr)
	c := jsonEnvelope(500, headers, bodyStr)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = diff.JSON(a, c, opts...)
	}
}

func benchmarkPipelineParsed(b *testing.B, bodyTree map[string]any, headers http.Header, opts ...diff.Option) {
	a := diff.BuildEnvelope(200, headers, bodyTree)
	c := diff.BuildEnvelope(500, headers, bodyTree)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = diff.Parsed(a, c, opts...)
	}
}

// Small body: 10 fields
func BenchmarkPipeline_JSON_Small(b *testing.B) {
	bodyStr, _ := makeBody(10)
	benchmarkPipelineJSON(b, bodyStr, makeHeaders(5))
}

func BenchmarkPipeline_Parsed_Small(b *testing.B) {
	_, bodyTree := makeBody(10)
	benchmarkPipelineParsed(b, bodyTree, makeHeaders(5))
}

// Medium body: 100 fields
func BenchmarkPipeline_JSON_Medium(b *testing.B) {
	bodyStr, _ := makeBody(100)
	benchmarkPipelineJSON(b, bodyStr, makeHeaders(10))
}

func BenchmarkPipeline_Parsed_Medium(b *testing.B) {
	_, bodyTree := makeBody(100)
	benchmarkPipelineParsed(b, bodyTree, makeHeaders(10))
}

// Large body: 1000 fields
func BenchmarkPipeline_JSON_Large(b *testing.B) {
	bodyStr, _ := makeBody(1000)
	benchmarkPipelineJSON(b, bodyStr, makeHeaders(20))
}

func BenchmarkPipeline_Parsed_Large(b *testing.B) {
	_, bodyTree := makeBody(1000)
	benchmarkPipelineParsed(b, bodyTree, makeHeaders(20))
}

// With ignored fields (5 fields ignored)
func BenchmarkPipeline_JSON_Medium_IgnoredFields(b *testing.B) {
	bodyStr, _ := makeBody(100)
	opts := []diff.Option{diff.WithIgnoredFields("field_0", "field_10", "field_20", "field_30", "field_40")}
	benchmarkPipelineJSON(b, bodyStr, makeHeaders(10), opts...)
}

func BenchmarkPipeline_Parsed_Medium_IgnoredFields(b *testing.B) {
	_, bodyTree := makeBody(100)
	opts := []diff.Option{diff.WithIgnoredFields("field_0", "field_10", "field_20", "field_30", "field_40")}
	benchmarkPipelineParsed(b, bodyTree, makeHeaders(10), opts...)
}

// With included fields (5 fields included)
func BenchmarkPipeline_JSON_Medium_IncludedFields(b *testing.B) {
	bodyStr, _ := makeBody(100)
	opts := []diff.Option{diff.WithIncludedFields("field_0", "field_10", "field_20", "field_30", "field_40")}
	benchmarkPipelineJSON(b, bodyStr, makeHeaders(10), opts...)
}

func BenchmarkPipeline_Parsed_Medium_IncludedFields(b *testing.B) {
	_, bodyTree := makeBody(100)
	opts := []diff.Option{diff.WithIncludedFields("field_0", "field_10", "field_20", "field_30", "field_40")}
	benchmarkPipelineParsed(b, bodyTree, makeHeaders(10), opts...)
}

// With differences in body content
func BenchmarkPipeline_JSON_Medium_WithDiffs(b *testing.B) {
	bodyStrA, _ := makeBody(100)
	// Build a different body
	m := make(map[string]any, 100)
	for i := 0; i < 100; i++ {
		m[fmt.Sprintf("field_%d", i)] = fmt.Sprintf("different_%d", i)
	}
	bodyB, _ := json.Marshal(m)
	headers := makeHeaders(10)

	a := jsonEnvelope(200, headers, bodyStrA)
	c := jsonEnvelope(200, headers, string(bodyB))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = diff.JSON(a, c)
	}
}

func BenchmarkPipeline_Parsed_Medium_WithDiffs(b *testing.B) {
	_, bodyTreeA := makeBody(100)
	bodyTreeB := make(map[string]any, 100)
	for i := 0; i < 100; i++ {
		bodyTreeB[fmt.Sprintf("field_%d", i)] = fmt.Sprintf("different_%d", i)
	}
	headers := makeHeaders(10)

	a := diff.BuildEnvelope(200, headers, bodyTreeA)
	c := diff.BuildEnvelope(200, headers, bodyTreeB)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = diff.Parsed(a, c)
	}
}