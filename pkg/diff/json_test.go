package diff_test

import (
	"testing"

	"github.com/pedrobarco/mroki/pkg/diff"
	"github.com/stretchr/testify/assert"
)

func TestJSON_identical_json_returns_empty_diff(t *testing.T) {
	a := `{"key": "value", "nested": {"foo": "bar"}}`
	b := `{"key": "value", "nested": {"foo": "bar"}}`

	result, err := diff.JSON(a, b)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestJSON_different_json_returns_diff(t *testing.T) {
	a := `{"key": "value1", "count": 1}`
	b := `{"key": "value2", "count": 2}`

	result, err := diff.JSON(a, b)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "value1")
	assert.Contains(t, result, "value2")
}

func TestJSON_invalid_first_input_returns_error(t *testing.T) {
	a := `{invalid json}`
	b := `{"key": "value"}`

	result, err := diff.JSON(a, b)

	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "first input")
}

func TestJSON_invalid_second_input_returns_error(t *testing.T) {
	a := `{"key": "value"}`
	b := `{invalid json}`

	result, err := diff.JSON(a, b)

	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "second input")
}

// Field filtering tests

func TestJSON_IgnoredFields(t *testing.T) {
	// Two JSONs that differ only in timestamp field
	a := `{"name": "John", "age": 30, "timestamp": "2024-01-01T10:00:00Z"}`
	b := `{"name": "John", "age": 30, "timestamp": "2024-01-01T11:00:00Z"}`

	result, err := diff.JSON(a, b, diff.WithIgnoredFields("timestamp"))

	assert.NoError(t, err)
	assert.Empty(t, result, "should be identical when ignoring timestamp")
}

func TestJSON_IgnoredFields_StillDetectsDifferences(t *testing.T) {
	// Two JSONs that differ in both timestamp (ignored) and name (not ignored)
	a := `{"name": "John", "age": 30, "timestamp": "2024-01-01T10:00:00Z"}`
	b := `{"name": "Jane", "age": 30, "timestamp": "2024-01-01T11:00:00Z"}`

	result, err := diff.JSON(a, b, diff.WithIgnoredFields("timestamp"))

	assert.NoError(t, err)
	assert.NotEmpty(t, result, "should detect difference in name field")
	assert.Contains(t, result, "John")
	assert.Contains(t, result, "Jane")
}

func TestJSON_IncludedFields(t *testing.T) {
	// Two JSONs with multiple fields, but we only care about name and email
	a := `{"name": "John", "age": 30, "email": "john@example.com", "timestamp": "2024-01-01"}`
	b := `{"name": "John", "age": 35, "email": "john@example.com", "timestamp": "2024-01-02"}`

	result, err := diff.JSON(a, b, diff.WithIncludedFields("name", "email"))

	assert.NoError(t, err)
	assert.Empty(t, result, "should be identical when only comparing name and email")
}

func TestJSON_IncludedFields_DetectsDifferences(t *testing.T) {
	// Two JSONs where included field differs
	a := `{"name": "John", "age": 30, "email": "john@example.com"}`
	b := `{"name": "Jane", "age": 30, "email": "john@example.com"}`

	result, err := diff.JSON(a, b, diff.WithIncludedFields("name", "email"))

	assert.NoError(t, err)
	assert.NotEmpty(t, result, "should detect difference in name field")
	assert.Contains(t, result, "John")
	assert.Contains(t, result, "Jane")
}

func TestJSON_NestedFields(t *testing.T) {
	a := `{"user": {"name": "John", "profile": {"age": 30, "email": "john@example.com"}}, "timestamp": "2024-01-01"}`
	b := `{"user": {"name": "John", "profile": {"age": 35, "email": "john@example.com"}}, "timestamp": "2024-01-02"}`

	result, err := diff.JSON(a, b, diff.WithIncludedFields("user.name", "user.profile.email"))

	assert.NoError(t, err)
	assert.Empty(t, result, "should be identical when only comparing name and email")
}

func TestJSON_ArrayWildcard_IgnoredFields(t *testing.T) {
	a := `{"users": [{"name": "John", "created_at": "2024-01-01"}, {"name": "Jane", "created_at": "2024-01-02"}]}`
	b := `{"users": [{"name": "John", "created_at": "2024-02-01"}, {"name": "Jane", "created_at": "2024-02-02"}]}`

	result, err := diff.JSON(a, b, diff.WithIgnoredFields("users.#.created_at"))

	assert.NoError(t, err)
	assert.Empty(t, result, "should be identical when ignoring created_at in all array elements")
}

func TestJSON_HybridStrategy(t *testing.T) {
	// Hybrid: include + exclude
	// Use case: Include entire user object, but exclude sensitive fields
	a := `{"name": "John", "age": 30, "email": "john@example.com"}`
	b := `{"name": "Jane", "age": 30, "email": "jane@example.com"}`

	result, err := diff.JSON(a, b,
		diff.WithIncludedFields("name", "email"),
		diff.WithIgnoredFields("email"),
	)

	assert.NoError(t, err)
	assert.NotEmpty(t, result, "should detect name difference")
	assert.Contains(t, result, "John")
	assert.Contains(t, result, "Jane")
	// Email should not be in diff (was excluded after include)
	assert.NotContains(t, result, "john@example.com")
	assert.NotContains(t, result, "jane@example.com")
}

func TestJSON_NoFiltering(t *testing.T) {
	a := `{"key": "value1"}`
	b := `{"key": "value2"}`

	result, err := diff.JSON(a, b)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "value1")
	assert.Contains(t, result, "value2")
}

func TestJSON_WithSortArrays(t *testing.T) {
	// Same elements, different order
	a := `{"items": [1, 2, 3]}`
	b := `{"items": [3, 1, 2]}`

	// Without sorting - should detect difference
	result, err := diff.JSON(a, b)
	assert.NoError(t, err)
	assert.NotEmpty(t, result, "should detect difference without sorting")

	// With sorting - should be identical
	result, err = diff.JSON(a, b, diff.WithSortArrays())
	assert.NoError(t, err)
	assert.Empty(t, result, "should be identical with sorting")
}

func TestJSON_WithFloatTolerance(t *testing.T) {
	a := `{"value": 1.0001}`
	b := `{"value": 1.0002}`

	// Without tolerance - should detect difference
	result, err := diff.JSON(a, b)
	assert.NoError(t, err)
	assert.NotEmpty(t, result, "should detect difference without tolerance")

	// With tolerance - should be identical
	result, err = diff.JSON(a, b, diff.WithFloatTolerance(0.001))
	assert.NoError(t, err)
	assert.Empty(t, result, "should be identical with tolerance")
}

func TestJSON_MultipleOptions(t *testing.T) {
	a := `{"user": {"name": "John", "ssn": "123-45-6789", "balance": 100.001}, "timestamp": "2024-01-01"}`
	b := `{"user": {"name": "John", "ssn": "987-65-4321", "balance": 100.002}, "timestamp": "2024-01-02"}`

	result, err := diff.JSON(a, b,
		diff.WithIncludedFields("user"),
		diff.WithIgnoredFields("user.ssn"),
		diff.WithFloatTolerance(0.01),
	)

	assert.NoError(t, err)
	assert.Empty(t, result, "should be identical with multiple filters")
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
