package jsontree_test

import (
	"testing"

	"github.com/pedrobarco/mroki/pkg/jsontree"
	"github.com/stretchr/testify/assert"
)

func TestPickPaths_single_top_level_field(t *testing.T) {
	tree := map[string]any{"name": "Alice", "age": float64(30)}
	result := jsontree.PickPaths(tree, []string{"name"})
	assert.Equal(t, map[string]any{"name": "Alice"}, result)
}

func TestPickPaths_nested_field(t *testing.T) {
	tree := map[string]any{
		"user": map[string]any{
			"name":  "Alice",
			"email": "alice@example.com",
		},
	}
	result := jsontree.PickPaths(tree, []string{"user.name"})
	assert.Equal(t, map[string]any{
		"user": map[string]any{"name": "Alice"},
	}, result)
}

func TestPickPaths_array_wildcard(t *testing.T) {
	tree := map[string]any{
		"items": []any{
			map[string]any{"id": float64(1), "label": "a"},
			map[string]any{"id": float64(2), "label": "b"},
		},
	}
	result := jsontree.PickPaths(tree, []string{"items.#.id"})
	assert.Equal(t, map[string]any{
		"items": []any{
			map[string]any{"id": float64(1)},
			map[string]any{"id": float64(2)},
		},
	}, result)
}

func TestPickPaths_multiple_paths(t *testing.T) {
	tree := map[string]any{
		"name": "Alice",
		"age":  float64(30),
		"city": "NYC",
	}
	result := jsontree.PickPaths(tree, []string{"name", "age"})
	assert.Equal(t, map[string]any{"name": "Alice", "age": float64(30)}, result)
}

func TestPickPaths_no_matching_paths(t *testing.T) {
	tree := map[string]any{"name": "Alice"}
	result := jsontree.PickPaths(tree, []string{"nonexistent"})
	assert.Equal(t, map[string]any{}, result)
}

func TestPickPaths_nil_tree(t *testing.T) {
	result := jsontree.PickPaths(nil, []string{"a"})
	assert.Equal(t, map[string]any{}, result)
}

func TestPickPaths_non_map_tree(t *testing.T) {
	result := jsontree.PickPaths("scalar", []string{"a"})
	assert.Equal(t, map[string]any{}, result)
}

func TestPickPaths_empty_paths(t *testing.T) {
	tree := map[string]any{"name": "Alice"}
	result := jsontree.PickPaths(tree, []string{})
	assert.Equal(t, map[string]any{}, result)
}

func TestPickPaths_missing_intermediate_key(t *testing.T) {
	tree := map[string]any{"user": map[string]any{"name": "Alice"}}
	result := jsontree.PickPaths(tree, []string{"user.profile.email"})
	assert.Equal(t, map[string]any{}, result)
}

func TestPickPaths_nested_array_wildcard(t *testing.T) {
	tree := map[string]any{
		"data": []any{
			map[string]any{
				"tags": []any{
					map[string]any{"name": "go", "count": float64(5)},
					map[string]any{"name": "rust", "count": float64(3)},
				},
			},
			map[string]any{
				"tags": []any{
					map[string]any{"name": "python", "count": float64(8)},
				},
			},
		},
	}
	result := jsontree.PickPaths(tree, []string{"data.#.tags.#.name"})
	assert.Equal(t, map[string]any{
		"data": []any{
			map[string]any{
				"tags": []any{
					map[string]any{"name": "go"},
					map[string]any{"name": "rust"},
				},
			},
			map[string]any{
				"tags": []any{
					map[string]any{"name": "python"},
				},
			},
		},
	}, result)
}
