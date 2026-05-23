package jsontree_test

import (
	"testing"

	"github.com/pedrobarco/mroki/pkg/jsontree"
	"github.com/stretchr/testify/assert"
)

func TestDeepCopy_map_with_nested_maps(t *testing.T) {
	original := map[string]any{
		"user": map[string]any{
			"profile": map[string]any{
				"name": "Alice",
			},
		},
	}
	copied := jsontree.DeepCopy(original)
	assert.Equal(t, original, copied)
}

func TestDeepCopy_with_arrays(t *testing.T) {
	original := map[string]any{
		"items": []any{float64(1), float64(2), float64(3)},
	}
	copied := jsontree.DeepCopy(original)
	assert.Equal(t, original, copied)
}

func TestDeepCopy_mixed_types(t *testing.T) {
	original := map[string]any{
		"name":   "Alice",
		"age":    float64(30),
		"active": true,
		"extra":  nil,
	}
	copied := jsontree.DeepCopy(original)
	assert.Equal(t, original, copied)
}

func TestDeepCopy_nil_input(t *testing.T) {
	result := jsontree.DeepCopy(nil)
	assert.Nil(t, result)
}

func TestDeepCopy_mutation_independence(t *testing.T) {
	original := map[string]any{
		"nested": map[string]any{"key": "original"},
		"arr":    []any{"a", "b"},
	}
	copied := jsontree.DeepCopy(original)

	// Mutate the copy
	copiedMap := copied.(map[string]any)
	copiedMap["nested"].(map[string]any)["key"] = "modified"
	copiedMap["arr"].([]any)[0] = "x"

	// Original must be unchanged
	assert.Equal(t, "original", original["nested"].(map[string]any)["key"])
	assert.Equal(t, "a", original["arr"].([]any)[0])
}

func TestDeepCopy_scalar_input(t *testing.T) {
	assert.Equal(t, "hello", jsontree.DeepCopy("hello"))
	assert.Equal(t, float64(42), jsontree.DeepCopy(float64(42)))
	assert.Equal(t, true, jsontree.DeepCopy(true))
}

func TestDeepCopy_empty_map(t *testing.T) {
	original := map[string]any{}
	copied := jsontree.DeepCopy(original)
	assert.Equal(t, original, copied)
	assert.NotNil(t, copied)
}

func TestDeepCopy_empty_array(t *testing.T) {
	original := []any{}
	copied := jsontree.DeepCopy(original)
	assert.Equal(t, original, copied)
	assert.NotNil(t, copied)
}
