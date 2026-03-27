package diff

// Differ is an interface for comparing two values and returning a list of
// RFC 6902 JSON Patch operations describing the differences.
type Differ[T any] interface {
	Diff(a, b T) ([]PatchOp, error)
}

// DifferFunc is a function type that implements the Differ interface.
type DifferFunc[T any] func(a, b T) ([]PatchOp, error)

// Diff implements the Differ interface for DifferFunc.
func (f DifferFunc[T]) Diff(a, b T) ([]PatchOp, error) {
	return f(a, b)
}
