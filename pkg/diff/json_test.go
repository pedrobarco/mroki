package diff_test

import (
	"testing"

	"github.com/pedrobarco/mroki/pkg/diff"
	"github.com/stretchr/testify/assert"
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

	ops, err := diff.JSON(a, b)
	assert.NoError(t, err)
	assert.Empty(t, ops, "number arrays with same elements in different order should be equal")
}

func TestJSON_SortArrays_Strings(t *testing.T) {
	a := `{"tags": ["alpha", "beta", "gamma"]}`
	b := `{"tags": ["gamma", "alpha", "beta"]}`

	ops, err := diff.JSON(a, b)
	assert.NoError(t, err)
	assert.Empty(t, ops, "string arrays with same elements in different order should be equal")
}

func TestJSON_SortArrays_Objects(t *testing.T) {
	a := `{"users": [{"name": "Alice", "age": 30}, {"name": "Bob", "age": 25}]}`
	b := `{"users": [{"name": "Bob", "age": 25}, {"name": "Alice", "age": 30}]}`

	ops, err := diff.JSON(a, b)
	assert.NoError(t, err)
	assert.Empty(t, ops, "arrays of objects with same elements in different order should be equal")
}

func TestJSON_SortArrays_ObjectsDifferentKeyOrder(t *testing.T) {
	a := `{"users": [{"name": "Alice", "age": 30}, {"name": "Bob", "age": 25}]}`
	b := `{"users": [{"age": 25, "name": "Bob"}, {"age": 30, "name": "Alice"}]}`

	ops, err := diff.JSON(a, b)
	assert.NoError(t, err)
	assert.Empty(t, ops, "arrays of objects with different key order and array order should be equal")
}

func TestJSON_SortArrays_NestedArrays(t *testing.T) {
	a := `{"matrix": [[1, 2], [3, 4]]}`
	b := `{"matrix": [[3, 4], [1, 2]]}`

	ops, err := diff.JSON(a, b)
	assert.NoError(t, err)
	assert.Empty(t, ops, "nested arrays with same elements in different order should be equal")
}

func TestJSON_SortArrays_ActualDifference(t *testing.T) {
	a := `{"items": [1, 2, 3]}`
	b := `{"items": [1, 2, 4]}`

	ops, err := diff.JSON(a, b)
	assert.NoError(t, err)
	assert.NotEmpty(t, ops, "arrays with genuinely different elements should produce a diff")
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
