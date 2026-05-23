package jsontree_test

import (
	"testing"

	"github.com/pedrobarco/mroki/pkg/jsontree"
	"github.com/stretchr/testify/assert"
)

func TestWalkPath_nil_tree(t *testing.T) {
	called := false
	jsontree.WalkPath(nil, "a.b", func(parent map[string]any, key string) {
		called = true
	})
	assert.False(t, called)
}

func TestWalkPath_empty_path(t *testing.T) {
	called := false
	jsontree.WalkPath(map[string]any{"a": 1}, "", func(parent map[string]any, key string) {
		called = true
	})
	assert.False(t, called)
}

func TestWalkPath_top_level_key(t *testing.T) {
	tree := map[string]any{"name": "Alice", "age": float64(30)}
	var visited string
	jsontree.WalkPath(tree, "name", func(parent map[string]any, key string) {
		visited = key
	})
	assert.Equal(t, "name", visited)
}

func TestWalkPath_missing_key(t *testing.T) {
	tree := map[string]any{"name": "Alice"}
	called := false
	jsontree.WalkPath(tree, "nonexistent", func(parent map[string]any, key string) {
		called = true
	})
	assert.False(t, called)
}

func TestWalkPath_nested_key(t *testing.T) {
	tree := map[string]any{
		"user": map[string]any{
			"profile": map[string]any{
				"email": "alice@example.com",
			},
		},
	}
	var visited string
	jsontree.WalkPath(tree, "user.profile.email", func(parent map[string]any, key string) {
		visited = parent[key].(string)
	})
	assert.Equal(t, "alice@example.com", visited)
}

func TestWalkPath_missing_intermediate(t *testing.T) {
	tree := map[string]any{"user": map[string]any{"name": "Alice"}}
	called := false
	jsontree.WalkPath(tree, "user.profile.email", func(parent map[string]any, key string) {
		called = true
	})
	assert.False(t, called)
}

func TestWalkPath_array_wildcard(t *testing.T) {
	tree := map[string]any{
		"users": []any{
			map[string]any{"name": "Alice", "age": float64(30)},
			map[string]any{"name": "Bob", "age": float64(25)},
		},
	}
	var names []string
	jsontree.WalkPath(tree, "users.#.name", func(parent map[string]any, key string) {
		names = append(names, parent[key].(string))
	})
	assert.Equal(t, []string{"Alice", "Bob"}, names)
}

func TestWalkPath_array_wildcard_missing_field(t *testing.T) {
	tree := map[string]any{
		"users": []any{
			map[string]any{"name": "Alice"},
			map[string]any{"age": float64(25)},
		},
	}
	var names []string
	jsontree.WalkPath(tree, "users.#.name", func(parent map[string]any, key string) {
		names = append(names, parent[key].(string))
	})
	assert.Equal(t, []string{"Alice"}, names, "should only visit elements where key exists")
}

func TestWalkPath_array_wildcard_no_remaining_path(t *testing.T) {
	tree := map[string]any{
		"tags": []any{"a", "b", "c"},
	}
	called := false
	jsontree.WalkPath(tree, "tags.#", func(parent map[string]any, key string) {
		called = true
	})
	assert.False(t, called, "cannot visit array elements directly (no parent map key)")
}

func TestWalkPath_nested_array_wildcard(t *testing.T) {
	tree := map[string]any{
		"company": map[string]any{
			"departments": []any{
				map[string]any{"name": "Engineering", "head": "Alice"},
				map[string]any{"name": "Sales", "head": "Bob"},
			},
		},
	}
	var heads []string
	jsontree.WalkPath(tree, "company.departments.#.head", func(parent map[string]any, key string) {
		heads = append(heads, parent[key].(string))
	})
	assert.Equal(t, []string{"Alice", "Bob"}, heads)
}

func TestWalkPath_visitor_can_mutate(t *testing.T) {
	tree := map[string]any{"secret": "password123"}
	jsontree.WalkPath(tree, "secret", func(parent map[string]any, key string) {
		parent[key] = "[REDACTED]"
	})
	assert.Equal(t, "[REDACTED]", tree["secret"])
}

func TestWalkPath_visitor_can_delete(t *testing.T) {
	tree := map[string]any{"keep": "yes", "remove": "no"}
	jsontree.WalkPath(tree, "remove", func(parent map[string]any, key string) {
		delete(parent, key)
	})
	assert.Equal(t, map[string]any{"keep": "yes"}, tree)
}

func TestWalkPath_non_map_intermediate(t *testing.T) {
	tree := map[string]any{"user": "not-a-map"}
	called := false
	jsontree.WalkPath(tree, "user.name", func(parent map[string]any, key string) {
		called = true
	})
	assert.False(t, called, "should not recurse into non-map values")
}
