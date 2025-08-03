package diffing

import "github.com/google/uuid"

type Diff struct {
	ID      uuid.UUID
	Content string
}

func NewDiff(content string) (*Diff, error) {
	return &Diff{
		ID:      uuid.New(),
		Content: content,
	}, nil
}
