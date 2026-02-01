package traffictesting

import "github.com/google/uuid"

type Diff struct {
	FromResponseID uuid.UUID
	ToResponseID   uuid.UUID
	Content        string
}

type diffOption func(*Diff)

func NewDiff(from, to uuid.UUID, content string, opts ...diffOption) (*Diff, error) {
	diff := &Diff{
		FromResponseID: from,
		ToResponseID:   to,
		Content:        content,
	}

	for _, o := range opts {
		o(diff)
	}

	return diff, nil
}

// IsZero returns true if the Diff is the zero value.
// A zero Diff has nil FromResponseID and ToResponseID.
func (d Diff) IsZero() bool {
	return d.FromResponseID == uuid.Nil && d.ToResponseID == uuid.Nil
}

// Equals compares two Diff value objects for equality.
// Value objects are equal if all their attributes are equal.
func (d Diff) Equals(other Diff) bool {
	return d.FromResponseID == other.FromResponseID &&
		d.ToResponseID == other.ToResponseID &&
		d.Content == other.Content
}
