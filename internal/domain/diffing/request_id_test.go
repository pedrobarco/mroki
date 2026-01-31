package diffing_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/pedrobarco/mroki/internal/domain/diffing"
	"github.com/stretchr/testify/assert"
)

func TestParseRequestID_valid_uuid(t *testing.T) {
	input := "223e4567-e89b-12d3-a456-426614174000"

	id, err := diffing.ParseRequestID(input)

	assert.NoError(t, err)
	assert.Equal(t, input, id.String())
	assert.False(t, id.IsZero())
}

func TestParseRequestID_invalid_uuid(t *testing.T) {
	input := "not-a-uuid"

	id, err := diffing.ParseRequestID(input)

	assert.ErrorIs(t, err, diffing.ErrInvalidRequestID)
	assert.True(t, id.IsZero())
}

func TestNewRequestID_creates_unique_ids(t *testing.T) {
	id1 := diffing.NewRequestID()
	id2 := diffing.NewRequestID()

	assert.False(t, id1.IsZero())
	assert.False(t, id2.IsZero())
	assert.NotEqual(t, id1.String(), id2.String())
}

func TestRequestIDFromUUID_roundtrip(t *testing.T) {
	original := uuid.New()

	requestID := diffing.RequestIDFromUUID(original)

	assert.Equal(t, original, requestID.UUID())
	assert.Equal(t, original.String(), requestID.String())
}

func TestRequestID_IsZero(t *testing.T) {
	var zero diffing.RequestID
	assert.True(t, zero.IsZero())

	nonZero := diffing.NewRequestID()
	assert.False(t, nonZero.IsZero())
}
