package jsontree_test

import (
	"testing"

	"github.com/pedrobarco/mroki/pkg/jsontree"
	"github.com/stretchr/testify/assert"
)

func TestDeletePaths_top_level_field(t *testing.T) {
	tree := map[string]any{"name": "Alice", "age": float64(30)}
	jsontree.DeletePaths(tree, []string{"name"})
	assert.Equal(t, map[string]any{"age": float64(30)}, tree)
}

func TestDeletePaths_nested_field(t *testing.T) {
	tree := map[string]any{
		"user": map[string]any{
			"name":  "Alice",
			"email": "alice@example.com",
		},
	}
	jsontree.DeletePaths(tree, []string{"user.email"})
	assert.Equal(t, map[string]any{
		"user": map[string]any{"name": "Alice"},
	}, tree)
}

func TestDeletePaths_array_wildcard(t *testing.T) {
	tree := map[string]any{
		"users": []any{
			map[string]any{"name": "Alice", "secret": "pw1"},
			map[string]any{"name": "Bob", "secret": "pw2"},
		},
	}
	jsontree.DeletePaths(tree, []string{"users.#.secret"})
	assert.Equal(t, map[string]any{
		"users": []any{
			map[string]any{"name": "Alice"},
			map[string]any{"name": "Bob"},
		},
	}, tree)
}

func TestDeletePaths_multiple_paths(t *testing.T) {
	tree := map[string]any{"a": float64(1), "b": float64(2), "c": float64(3)}
	jsontree.DeletePaths(tree, []string{"a", "c"})
	assert.Equal(t, map[string]any{"b": float64(2)}, tree)
}

func TestDeletePaths_non_existent_path(t *testing.T) {
	tree := map[string]any{"name": "Alice"}
	jsontree.DeletePaths(tree, []string{"nonexistent"})
	assert.Equal(t, map[string]any{"name": "Alice"}, tree)
}

func TestDeletePaths_nil_tree(t *testing.T) {
	assert.NotPanics(t, func() {
		jsontree.DeletePaths(nil, []string{"a"})
	})
}

func TestDeletePaths_non_map_tree(t *testing.T) {
	assert.NotPanics(t, func() {
		jsontree.DeletePaths("scalar", []string{"a"})
	})
}

func TestDeletePaths_empty_paths(t *testing.T) {
	tree := map[string]any{"name": "Alice"}
	jsontree.DeletePaths(tree, []string{})
	assert.Equal(t, map[string]any{"name": "Alice"}, tree)
}

func TestDeletePaths_in_place_mutation(t *testing.T) {
	tree := map[string]any{"keep": "yes", "remove": "no"}
	jsontree.DeletePaths(tree, []string{"remove"})
	_, exists := tree["remove"]
	assert.False(t, exists, "original tree should be mutated in-place")
	assert.Equal(t, "yes", tree["keep"])
}
