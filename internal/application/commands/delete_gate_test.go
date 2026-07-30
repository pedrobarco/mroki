package commands

import (
	"context"
	"errors"
	"testing"

	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteGateHandler_Handle_success(t *testing.T) {
	// Arrange
	repo := &mockGateRepository{
		deleteFn: func(ctx context.Context, id traffictesting.GateID) error {
			assert.False(t, id.IsZero())
			return nil
		},
	}
	handler := NewDeleteGateHandler(repo)

	cmd := DeleteGateCommand{ID: traffictesting.NewGateID().String()}

	// Act
	err := handler.Handle(context.Background(), cmd)

	// Assert
	require.NoError(t, err)
}

func TestDeleteGateHandler_Handle_invalid_id(t *testing.T) {
	// Arrange
	repo := &mockGateRepository{}
	handler := NewDeleteGateHandler(repo)

	cmd := DeleteGateCommand{ID: "not-a-valid-id"}

	// Act
	err := handler.Handle(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid gate ID")
}

func TestDeleteGateHandler_Handle_repository_error(t *testing.T) {
	// Arrange
	expectedErr := errors.New("database connection failed")
	repo := &mockGateRepository{
		deleteFn: func(ctx context.Context, id traffictesting.GateID) error {
			return expectedErr
		},
	}
	handler := NewDeleteGateHandler(repo)

	cmd := DeleteGateCommand{ID: traffictesting.NewGateID().String()}

	// Act
	err := handler.Handle(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete gate")
	assert.ErrorIs(t, err, expectedErr)
}
