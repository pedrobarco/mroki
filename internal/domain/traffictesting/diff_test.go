package traffictesting_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/stretchr/testify/assert"
)

func TestNewDiff_creates_diff_with_values(t *testing.T) {
	fromID := uuid.New()
	toID := uuid.New()
	content := `{"status": "different"}`

	diff, err := traffictesting.NewDiff(fromID, toID, content)

	assert.NoError(t, err)
	assert.Equal(t, fromID, diff.FromResponseID)
	assert.Equal(t, toID, diff.ToResponseID)
	assert.Equal(t, content, diff.Content)
}

func TestDiff_IsZero(t *testing.T) {
	var zero traffictesting.Diff
	assert.True(t, zero.IsZero())

	fromID := uuid.New()
	toID := uuid.New()
	diff, _ := traffictesting.NewDiff(fromID, toID, "content")
	assert.False(t, diff.IsZero())
}

func TestDiff_Equals(t *testing.T) {
	fromID := uuid.New()
	toID := uuid.New()
	content := "test content"

	diff1, _ := traffictesting.NewDiff(fromID, toID, content)
	diff2, _ := traffictesting.NewDiff(fromID, toID, content)

	assert.True(t, diff1.Equals(*diff2), "diffs with same values should be equal")

	diff3, _ := traffictesting.NewDiff(uuid.New(), toID, content)
	assert.False(t, diff1.Equals(*diff3), "diffs with different FromResponseID should not be equal")
}

func TestNewDiff_with_empty_content(t *testing.T) {
	fromID := uuid.New()
	toID := uuid.New()

	diff, err := traffictesting.NewDiff(fromID, toID, "")

	assert.NoError(t, err)
	assert.Equal(t, "", diff.Content, "empty content should be allowed")
	assert.False(t, diff.IsZero(), "diff with empty content but valid IDs is not zero")
}
