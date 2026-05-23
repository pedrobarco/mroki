package jsontree

// DeepCopy creates a deep copy of a JSON value tree to avoid aliasing.
// Maps and slices are recursively copied; scalars (string, float64, bool, nil)
// are returned as-is since they are immutable.
func DeepCopy(v Tree) Tree {
	switch val := v.(type) {
	case map[string]any:
		cp := make(map[string]any, len(val))
		for k, v := range val {
			cp[k] = DeepCopy(v)
		}
		return cp
	case []any:
		cp := make([]any, len(val))
		for i, v := range val {
			cp[i] = DeepCopy(v)
		}
		return cp
	default:
		return v // scalars are immutable
	}
}
