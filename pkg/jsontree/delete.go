package jsontree

// DeletePaths deletes the specified dot-separated paths from the tree in-place.
// Supports "#" as a wildcard segment for iterating array elements.
// The caller is responsible for copying the tree first if mutation is not desired.
func DeletePaths(tree Tree, paths []string) {
	for _, path := range paths {
		WalkPath(tree, path, func(parent map[string]any, key string) {
			delete(parent, key)
		})
	}
}
