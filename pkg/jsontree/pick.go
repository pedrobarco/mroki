package jsontree

import "strings"

// PickPaths builds a new tree containing only the values at the specified
// dot-separated paths. The source tree is never mutated.
// Supports "#" as a wildcard segment for iterating array elements.
// Returns an empty map (not nil) when no paths match — this is intentional
// so the result is always a valid tree for downstream operations.
func PickPaths(tree Tree, paths []string) Tree {
	result := make(map[string]any)
	for _, path := range paths {
		segments := strings.Split(path, ".")
		pickPath(tree, result, segments)
	}
	return result
}

// pickPath copies a value from src to dst along the given path segments.
// Creates intermediate maps as needed. Handles "#" wildcard for arrays.
func pickPath(src any, dst map[string]any, segments []string) {
	if len(segments) == 0 || src == nil {
		return
	}

	key := segments[0]
	rest := segments[1:]

	switch v := src.(type) {
	case map[string]any:
		if len(rest) == 0 {
			// Leaf: copy value if it exists
			if val, ok := v[key]; ok {
				dst[key] = DeepCopy(val)
			}
			return
		}

		// Handle "#" wildcard in next segment
		if rest[0] == "#" {
			if child, ok := v[key]; ok {
				if arr, isArr := child.([]any); isArr {
					pickArrayPath(arr, dst, key, rest[1:])
				}
			}
			return
		}

		// Recurse into child
		if child, ok := v[key]; ok {
			childDst, ok := dst[key].(map[string]any)
			if !ok {
				childDst = make(map[string]any)
			}
			pickPath(child, childDst, rest)
			// Only set the intermediate if the recursion produced something
			if len(childDst) > 0 {
				dst[key] = childDst
			}
		}
	}
}

// pickArrayPath copies fields from each array element matching the remaining path.
func pickArrayPath(arr []any, dst map[string]any, arrayKey string, remainingSegments []string) {
	if len(remainingSegments) == 0 {
		return
	}

	var result []any
	for _, elem := range arr {
		elemMap, ok := elem.(map[string]any)
		if !ok {
			continue
		}
		newElem := make(map[string]any)
		pickPath(elemMap, newElem, remainingSegments)
		if len(newElem) > 0 {
			result = append(result, newElem)
		}
	}
	if len(result) > 0 {
		dst[arrayKey] = result
	}
}
