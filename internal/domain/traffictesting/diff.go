package traffictesting

import (
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/pedrobarco/mroki/pkg/diff"
)

type Diff struct {
	FromResponseID uuid.UUID
	ToResponseID   uuid.UUID
	Content        []diff.PatchOp
	Config         DiffConfig
	CreatedAt      time.Time
}

type diffOption func(*Diff)

func NewDiff(from, to uuid.UUID, content []diff.PatchOp, config DiffConfig, opts ...diffOption) (*Diff, error) {
	d := &Diff{
		FromResponseID: from,
		ToResponseID:   to,
		Content:        content,
		Config:         config,
		CreatedAt:      time.Now(),
	}

	for _, o := range opts {
		o(d)
	}

	return d, nil
}

// IsZero returns true if the Diff is the zero value.
// A zero Diff has nil FromResponseID and ToResponseID.
func (d Diff) IsZero() bool {
	return d.FromResponseID == uuid.Nil && d.ToResponseID == uuid.Nil
}

// HasContent returns true if the Diff contains actual differences.
// A Diff with empty Content means the compared responses were identical.
func (d Diff) HasContent() bool {
	return len(d.Content) > 0
}

// Equals checks value equality by comparing Content and Config.
// Fields like FromResponseID, ToResponseID, and CreatedAt are
// entity/persistence concerns and not part of the value identity.
func (d Diff) Equals(other Diff) bool {
	if !reflect.DeepEqual(d.Content, other.Content) {
		return false
	}
	return reflect.DeepEqual(d.Config, other.Config)
}
