package traffictesting_test

import (
	"testing"

	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/pedrobarco/mroki/pkg/diff"
	"github.com/stretchr/testify/assert"
)

func TestNewDiff_creates_diff_with_values(t *testing.T) {
	content := []diff.PatchOp{
		{Op: "replace", Path: "/status", Value: "different"},
	}

	d, err := traffictesting.NewDiff(content, traffictesting.DefaultDiffConfig())

	assert.NoError(t, err)
	assert.Equal(t, content, d.Content)
	assert.Equal(t, traffictesting.DefaultDiffConfig(), d.Config)
}

func TestDiff_IsZero(t *testing.T) {
	var zero traffictesting.Diff
	assert.True(t, zero.IsZero())

	d, _ := traffictesting.NewDiff([]diff.PatchOp{{Op: "replace", Path: "/a"}}, traffictesting.DefaultDiffConfig())
	assert.False(t, d.IsZero())
}

func TestDiff_Equals(t *testing.T) {
	content := []diff.PatchOp{
		{Op: "replace", Path: "/name", Value: "bob"},
	}
	cfg := traffictesting.DefaultDiffConfig()

	diff1, _ := traffictesting.NewDiff(content, cfg)
	diff2, _ := traffictesting.NewDiff(content, cfg)

	assert.True(t, diff1.Equals(*diff2), "diffs with same values should be equal")

	t.Run("different content", func(t *testing.T) {
		other, _ := traffictesting.NewDiff([]diff.PatchOp{
			{Op: "add", Path: "/age", Value: float64(30)},
		}, cfg)
		assert.False(t, diff1.Equals(*other))
	})

	t.Run("different config", func(t *testing.T) {
		sortCfg, _ := traffictesting.NewDiffConfig(nil, nil, 0, true)
		other, _ := traffictesting.NewDiff(content, sortCfg)
		assert.False(t, diff1.Equals(*other))
	})
}

func TestDiff_HasContent(t *testing.T) {
	t.Run("zero diff returns false", func(t *testing.T) {
		var zero traffictesting.Diff
		assert.False(t, zero.HasContent())
	})

	t.Run("empty content returns false", func(t *testing.T) {
		d, _ := traffictesting.NewDiff([]diff.PatchOp{}, traffictesting.DefaultDiffConfig())
		assert.False(t, d.HasContent())
	})

	t.Run("nil content returns false", func(t *testing.T) {
		d, _ := traffictesting.NewDiff(nil, traffictesting.DefaultDiffConfig())
		assert.False(t, d.HasContent())
	})

	t.Run("non-empty content returns true", func(t *testing.T) {
		d, _ := traffictesting.NewDiff([]diff.PatchOp{
			{Op: "replace", Path: "/status", Value: "different"},
		}, traffictesting.DefaultDiffConfig())
		assert.True(t, d.HasContent())
	})
}

func TestNewDiff_with_empty_content(t *testing.T) {
	d, err := traffictesting.NewDiff([]diff.PatchOp{}, traffictesting.DefaultDiffConfig())

	assert.NoError(t, err)
	assert.Empty(t, d.Content, "empty content should be allowed")
	assert.False(t, d.IsZero(), "diff with empty (non-nil) content is not zero")
}
