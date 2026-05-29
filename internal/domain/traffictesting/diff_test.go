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
	d, err := traffictesting.NewDiff(fromID, toID, content, traffictesting.DefaultDiffConfig())
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
	d, _ := traffictesting.NewDiff(fromID, toID, []diff.PatchOp{{Op: "replace", Path: "/a"}}, traffictesting.DefaultDiffConfig())
	assert.False(t, d.IsZero())
}

func TestDiff_Equals(t *testing.T) {
	fromID := uuid.New()
	toID := uuid.New()
	content := []diff.PatchOp{
		{Op: "replace", Path: "/name", Value: "bob"},
	}
	cfg := traffictesting.DefaultDiffConfig()

	diff1, _ := traffictesting.NewDiff(fromID, toID, content, cfg)
	diff2, _ := traffictesting.NewDiff(fromID, toID, content, cfg)

	assert.True(t, diff1.Equals(*diff2), "diffs with same values should be equal")

	t.Run("different content", func(t *testing.T) {
		other, _ := traffictesting.NewDiff(fromID, toID, []diff.PatchOp{
			{Op: "add", Path: "/age", Value: float64(30)},
		}, cfg)
		assert.False(t, diff1.Equals(*other))
	})

	t.Run("different config", func(t *testing.T) {
		sortCfg, _ := traffictesting.NewDiffConfig(nil, nil, 0, true)
		other, _ := traffictesting.NewDiff(fromID, toID, content, sortCfg)
		assert.False(t, diff1.Equals(*other))
	})

	t.Run("ignores entity metadata", func(t *testing.T) {
		other, _ := traffictesting.NewDiff(uuid.New(), uuid.New(), content, cfg)
		other.CreatedAt = diff1.CreatedAt.Add(time.Hour)
		assert.True(t, diff1.Equals(*other), "response IDs and CreatedAt are not part of value identity")
	})
}

func TestDiff_HasContent(t *testing.T) {
	t.Run("zero diff returns false", func(t *testing.T) {
		var zero traffictesting.Diff
		assert.False(t, zero.HasContent())
	})

	t.Run("empty content returns false", func(t *testing.T) {
		d, _ := traffictesting.NewDiff(uuid.New(), uuid.New(), []diff.PatchOp{}, traffictesting.DefaultDiffConfig())
		assert.False(t, d.HasContent())
	})

	t.Run("nil content returns false", func(t *testing.T) {
		d, _ := traffictesting.NewDiff(uuid.New(), uuid.New(), nil, traffictesting.DefaultDiffConfig())
		assert.False(t, d.HasContent())
	})

	t.Run("non-empty content returns true", func(t *testing.T) {
		d, _ := traffictesting.NewDiff(uuid.New(), uuid.New(), []diff.PatchOp{
			{Op: "replace", Path: "/status", Value: "different"},
		}, traffictesting.DefaultDiffConfig())
		assert.True(t, d.HasContent())
	})
}

func TestNewDiff_with_empty_content(t *testing.T) {
	fromID := uuid.New()
	toID := uuid.New()

	d, err := traffictesting.NewDiff(fromID, toID, []diff.PatchOp{}, traffictesting.DefaultDiffConfig())

	assert.NoError(t, err)
	assert.Empty(t, d.Content, "empty content should be allowed")
	assert.False(t, d.IsZero(), "diff with empty content but valid IDs is not zero")
}
