package diffing_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/pedrobarco/mroki/internal/domain/diffing"
	"github.com/stretchr/testify/assert"
)

func TestParseGateID_valid_uuid(t *testing.T) {
	input := "123e4567-e89b-12d3-a456-426614174000"

	id, err := diffing.ParseGateID(input)

	assert.NoError(t, err)
	assert.Equal(t, input, id.String())
	assert.False(t, id.IsZero())
}

func TestParseGateID_invalid_uuid(t *testing.T) {
	input := "not-a-uuid"

	id, err := diffing.ParseGateID(input)

	assert.ErrorIs(t, err, diffing.ErrInvalidGateID)
	assert.True(t, id.IsZero())
}

func TestNewGateID_creates_unique_ids(t *testing.T) {
	id1 := diffing.NewGateID()
	id2 := diffing.NewGateID()

	assert.False(t, id1.IsZero())
	assert.False(t, id2.IsZero())
	assert.NotEqual(t, id1.String(), id2.String())
}

func TestGateIDFromUUID_roundtrip(t *testing.T) {
	original := uuid.New()

	gateID := diffing.GateIDFromUUID(original)

	assert.Equal(t, original, gateID.UUID())
	assert.Equal(t, original.String(), gateID.String())
}

func TestGateID_IsZero(t *testing.T) {
	var zero diffing.GateID
	assert.True(t, zero.IsZero())

	nonZero := diffing.NewGateID()
	assert.False(t, nonZero.IsZero())
}
