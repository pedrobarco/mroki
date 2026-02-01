package traffictesting_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/stretchr/testify/assert"
)

func TestParseRequestID_valid_uuid(t *testing.T) {
	input := "223e4567-e89b-12d3-a456-426614174000"

	id, err := traffictesting.ParseRequestID(input)

	assert.NoError(t, err)
	assert.Equal(t, input, id.String())
	assert.False(t, id.IsZero())
}

func TestParseRequestID_invalid_uuid(t *testing.T) {
	input := "not-a-uuid"

	id, err := traffictesting.ParseRequestID(input)

	assert.ErrorIs(t, err, traffictesting.ErrInvalidRequestID)
	assert.True(t, id.IsZero())
}

func TestNewRequestID_creates_unique_ids(t *testing.T) {
	id1 := traffictesting.NewRequestID()
	id2 := traffictesting.NewRequestID()

	assert.False(t, id1.IsZero())
	assert.False(t, id2.IsZero())
	assert.NotEqual(t, id1.String(), id2.String())
}

func TestRequestIDFromUUID_roundtrip(t *testing.T) {
	original := uuid.New()

	requestID := traffictesting.RequestIDFromUUID(original)

	assert.Equal(t, original, requestID.UUID())
	assert.Equal(t, original.String(), requestID.String())
}

func TestRequestID_IsZero(t *testing.T) {
	var zero traffictesting.RequestID
	assert.True(t, zero.IsZero())

	nonZero := traffictesting.NewRequestID()
	assert.False(t, nonZero.IsZero())
}
