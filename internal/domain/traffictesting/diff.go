package traffictesting

import "github.com/google/uuid"

type Diff struct {
	ID             uuid.UUID
	FromResponseID uuid.UUID
	ToResponseID   uuid.UUID
	Content        string
}

type diffOption func(*Diff)

func WithDiffID(id uuid.UUID) diffOption {
	return func(d *Diff) {
		d.ID = id
	}
}

func NewDiff(from, to uuid.UUID, content string, opts ...diffOption) (*Diff, error) {
	diff := &Diff{
		ID:             uuid.New(),
		FromResponseID: from,
		ToResponseID:   to,
		Content:        content,
	}

	for _, o := range opts {
		o(diff)
	}

	if diff.ID == uuid.Nil {
		diff.ID = uuid.New()
	}

	return diff, nil
}
