package diff

type Differ[T any] interface {
	Diff(a, b T) (string, error)
}

type DifferFunc[T any] func(a, b T) (string, error)
