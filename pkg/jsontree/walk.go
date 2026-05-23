package jsontree

import "strings"

// WalkPath navigates a map[string]any / []any tree along a dot-separated path
// and calls visitor(parent, key) at each matching leaf. The visitor receives the
// parent map and the final key so it can read, write, or delete the entry.
//
// The "#" segment is a wildcard that iterates over []any slices, applying the
// remaining path to each element.
//
// Missing intermediate keys are silently skipped. If the path leads to a
// non-existent key, the visitor is not called.
func WalkPath(tree Tree, path string, visitor func(parent map[string]any, key string)) {
	if path == "" || tree == nil {
		return
	}
	segments := strings.Split(path, ".")
	walkSegments(tree, segments, visitor)
}

// walkSegments is the recursive implementation of WalkPath.
func walkSegments(node Tree, segments []string, visitor func(parent map[string]any, key string)) {
	if len(segments) == 0 || node == nil {
		return
	}

	key := segments[0]
	rest := segments[1:]

	switch v := node.(type) {
	case map[string]any:
		if len(rest) == 0 {
			// Leaf: call visitor if key exists
			if _, ok := v[key]; ok {
				visitor(v, key)
			}
			return
		}
		// Recurse into child
		if child, ok := v[key]; ok {
			walkSegments(child, rest, visitor)
		}

	case []any:
		// "#" wildcard: apply remaining path to each array element
		if key == "#" {
			for _, elem := range v {
				if len(rest) == 0 {
					// Can't visit array elements directly (no parent map key)
					continue
				}
				walkSegments(elem, rest, visitor)
			}
		}
	}
}
