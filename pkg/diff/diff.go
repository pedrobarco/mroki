package diff

type Differ[T any] interface {
	Diff(a, b T) error
}

func NewNop[T any]() *nopDiffer[T] {
	return &nopDiffer[T]{}
}

type nopDiffer[T any] struct{}

func (nopDiffer[T]) Diff(a, b T) error {
	return nil
}
