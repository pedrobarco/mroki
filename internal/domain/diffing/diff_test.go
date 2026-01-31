package diffing_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/pedrobarco/mroki/internal/domain/diffing"
	"github.com/stretchr/testify/assert"
)

func TestNewDiff_creates_diff_with_auto_generated_id(t *testing.T) {
	fromID := uuid.New()
	toID := uuid.New()
	content := `{"status": "different"}`

	diff, err := diffing.NewDiff(fromID, toID, content)

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, diff.ID)
	assert.Equal(t, fromID, diff.FromResponseID)
	assert.Equal(t, toID, diff.ToResponseID)
	assert.Equal(t, content, diff.Content)
}

func TestNewDiff_with_custom_id(t *testing.T) {
	fromID := uuid.New()
	toID := uuid.New()
	customID := uuid.New()
	content := `{"field": "changed"}`

	diff, err := diffing.NewDiff(fromID, toID, content, diffing.WithDiffID(customID))

	assert.NoError(t, err)
	assert.Equal(t, customID, diff.ID)
}
