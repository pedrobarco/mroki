package traffictesting

import (
	"time"

	"github.com/google/uuid"
	"github.com/pedrobarco/mroki/pkg/diff"
)

type Diff struct {
	FromResponseID uuid.UUID
	ToResponseID   uuid.UUID
	Content        []diff.PatchOp
	CreatedAt      time.Time
}

type diffOption func(*Diff)

func NewDiff(from, to uuid.UUID, content []diff.PatchOp, opts ...diffOption) (*Diff, error) {
	diff := &Diff{
		FromResponseID: from,
		ToResponseID:   to,
		Content:        content,
		CreatedAt:      time.Now(),
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
	if d.FromResponseID != other.FromResponseID || d.ToResponseID != other.ToResponseID {
		return false
	}
	if len(d.Content) != len(other.Content) {
		return false
	}
	for i, op := range d.Content {
		if op.Op != other.Content[i].Op || op.Path != other.Content[i].Path {
			return false
		}
	}
	return true
}
