package diff

// Differ is an interface for comparing two values and returning a diff string.
type Differ[T any] interface {
	Diff(a, b T) (string, error)
}

// DifferFunc is a function type that implements the Differ interface.
type DifferFunc[T any] func(a, b T) (string, error)

// Diff implements the Differ interface for DifferFunc.
func (f DifferFunc[T]) Diff(a, b T) (string, error) {
	return f(a, b)
}
