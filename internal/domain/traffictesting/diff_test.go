package traffictesting_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/pedrobarco/mroki/pkg/diff"
	"github.com/stretchr/testify/assert"
)

func TestNewDiff_creates_diff_with_values(t *testing.T) {
	fromID := uuid.New()
	toID := uuid.New()
	content := []diff.PatchOp{
		{Op: "replace", Path: "/status", Value: "different"},
	}

	before := time.Now()
	d, err := traffictesting.NewDiff(fromID, toID, content)
	after := time.Now()

	assert.NoError(t, err)
	assert.Equal(t, fromID, d.FromResponseID)
	assert.Equal(t, toID, d.ToResponseID)
	assert.Equal(t, content, d.Content)
	assert.False(t, d.CreatedAt.IsZero(), "CreatedAt should be set automatically")
	assert.True(t, !d.CreatedAt.Before(before) && !d.CreatedAt.After(after),
		"CreatedAt should be approximately now")
}

func TestDiff_IsZero(t *testing.T) {
	var zero traffictesting.Diff
	assert.True(t, zero.IsZero())

	fromID := uuid.New()
	toID := uuid.New()
	d, _ := traffictesting.NewDiff(fromID, toID, []diff.PatchOp{{Op: "replace", Path: "/a"}})
	assert.False(t, d.IsZero())
}

func TestDiff_Equals(t *testing.T) {
	fromID := uuid.New()
	toID := uuid.New()
	content := []diff.PatchOp{
		{Op: "replace", Path: "/name", Value: "bob"},
	}

	diff1, _ := traffictesting.NewDiff(fromID, toID, content)
	diff2, _ := traffictesting.NewDiff(fromID, toID, content)

	assert.True(t, diff1.Equals(*diff2), "diffs with same values should be equal")

	diff3, _ := traffictesting.NewDiff(uuid.New(), toID, content)
	assert.False(t, diff1.Equals(*diff3), "diffs with different FromResponseID should not be equal")
}

func TestNewDiff_with_empty_content(t *testing.T) {
	fromID := uuid.New()
	toID := uuid.New()

	d, err := traffictesting.NewDiff(fromID, toID, []diff.PatchOp{})

	assert.NoError(t, err)
	assert.Empty(t, d.Content, "empty content should be allowed")
	assert.False(t, d.IsZero(), "diff with empty content but valid IDs is not zero")
}
