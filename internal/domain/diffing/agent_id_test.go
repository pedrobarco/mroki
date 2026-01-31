package diffing_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/pedrobarco/mroki/internal/domain/diffing"
	"github.com/stretchr/testify/assert"
)

func TestParseAgentID_valid_uuid(t *testing.T) {
	input := "323e4567-e89b-12d3-a456-426614174000"

	id, err := diffing.ParseAgentID(input)

	assert.NoError(t, err)
	assert.Equal(t, input, id.String())
	assert.False(t, id.IsZero())
}

func TestParseAgentID_invalid_uuid(t *testing.T) {
	input := "not-a-uuid"

	id, err := diffing.ParseAgentID(input)

	assert.ErrorIs(t, err, diffing.ErrInvalidAgentID)
	assert.True(t, id.IsZero())
}

func TestNewAgentID_creates_unique_ids(t *testing.T) {
	id1 := diffing.NewAgentID()
	id2 := diffing.NewAgentID()

	assert.False(t, id1.IsZero())
	assert.False(t, id2.IsZero())
	assert.NotEqual(t, id1.String(), id2.String())
}

func TestAgentIDFromUUID_roundtrip(t *testing.T) {
	original := uuid.New()

	agentID := diffing.AgentIDFromUUID(original)

	assert.Equal(t, original, agentID.UUID())
	assert.Equal(t, original.String(), agentID.String())
}

func TestAgentID_IsZero(t *testing.T) {
	var zero diffing.AgentID
	assert.True(t, zero.IsZero())

	nonZero := diffing.NewAgentID()
	assert.False(t, nonZero.IsZero())
}
