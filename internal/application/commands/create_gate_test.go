package commands

import (
	"context"
	"errors"
	"testing"

	"github.com/pedrobarco/mroki/internal/domain/pagination"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockGateRepository is a mock implementation of GateRepository for testing
type mockGateRepository struct {
	saveFn func(context.Context, *traffictesting.Gate) error
}

func (m *mockGateRepository) Save(ctx context.Context, gate *traffictesting.Gate) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, gate)
	}
	return nil
}

func (m *mockGateRepository) GetByID(ctx context.Context, id traffictesting.GateID) (*traffictesting.Gate, error) {
	return nil, errors.New("not implemented")
}

func (m *mockGateRepository) GetAll(ctx context.Context, filters traffictesting.GateFilters, sort traffictesting.GateSort, params *pagination.Params) (*pagination.PagedResult[*traffictesting.Gate], error) {
	return nil, errors.New("not implemented")
}

func TestCreateGateHandler_Handle_success(t *testing.T) {
	// Arrange
	repo := &mockGateRepository{
		saveFn: func(ctx context.Context, gate *traffictesting.Gate) error {
			assert.NotNil(t, gate)
			assert.False(t, gate.ID.IsZero())
			return nil
		},
	}
	handler := NewCreateGateHandler(repo)

	cmd := CreateGateCommand{
		LiveURL:   "https://api.example.com",
		ShadowURL: "https://api-staging.example.com",
	}

	// Act
	gate, err := handler.Handle(context.Background(), cmd)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, gate)
	assert.False(t, gate.ID.IsZero())
	assert.Equal(t, "https://api.example.com", gate.LiveURL.String())
	assert.Equal(t, "https://api-staging.example.com", gate.ShadowURL.String())
}

func TestCreateGateHandler_Handle_invalid_live_url(t *testing.T) {
	// Arrange
	repo := &mockGateRepository{}
	handler := NewCreateGateHandler(repo)

	cmd := CreateGateCommand{
		LiveURL:   "ftp://invalid-scheme.com", // Invalid scheme
		ShadowURL: "https://api-staging.example.com",
	}

	// Act
	gate, err := handler.Handle(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	assert.Nil(t, gate)
	assert.Contains(t, err.Error(), "invalid live URL")
}

func TestCreateGateHandler_Handle_invalid_shadow_url(t *testing.T) {
	// Arrange
	repo := &mockGateRepository{}
	handler := NewCreateGateHandler(repo)

	cmd := CreateGateCommand{
		LiveURL:   "https://api.example.com",
		ShadowURL: "not-a-valid-url", // Invalid URL
	}

	// Act
	gate, err := handler.Handle(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	assert.Nil(t, gate)
	assert.Contains(t, err.Error(), "invalid shadow URL")
}

func TestCreateGateHandler_Handle_repository_error(t *testing.T) {
	// Arrange
	expectedErr := errors.New("database connection failed")
	repo := &mockGateRepository{
		saveFn: func(ctx context.Context, gate *traffictesting.Gate) error {
			return expectedErr
		},
	}
	handler := NewCreateGateHandler(repo)

	cmd := CreateGateCommand{
		LiveURL:   "https://api.example.com",
		ShadowURL: "https://api-staging.example.com",
	}

	// Act
	gate, err := handler.Handle(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	assert.Nil(t, gate)
	assert.Contains(t, err.Error(), "failed to save gate")
	assert.ErrorIs(t, err, expectedErr)
}

func TestCreateGateHandler_Handle_http_urls(t *testing.T) {
	// Arrange
	repo := &mockGateRepository{
		saveFn: func(ctx context.Context, gate *traffictesting.Gate) error {
			return nil
		},
	}
	handler := NewCreateGateHandler(repo)

	cmd := CreateGateCommand{
		LiveURL:   "http://api.example.com", // HTTP should be valid
		ShadowURL: "http://api-staging.example.com",
	}

	// Act
	gate, err := handler.Handle(context.Background(), cmd)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, gate)
	assert.Equal(t, "http://api.example.com", gate.LiveURL.String())
	assert.Equal(t, "http://api-staging.example.com", gate.ShadowURL.String())
}
