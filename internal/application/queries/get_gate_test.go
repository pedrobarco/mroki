package queries

import (
	"context"
	"errors"
	"testing"

	"github.com/pedrobarco/mroki/internal/domain/pagination"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockGateRepositoryForGetGate struct {
	getByIDFn func(context.Context, traffictesting.GateID) (*traffictesting.Gate, error)
}

func (m *mockGateRepositoryForGetGate) Save(ctx context.Context, gate *traffictesting.Gate) error {
	return errors.New("not implemented")
}

func (m *mockGateRepositoryForGetGate) GetByID(ctx context.Context, id traffictesting.GateID) (*traffictesting.Gate, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockGateRepositoryForGetGate) GetAll(ctx context.Context, filters traffictesting.GateFilters, sort traffictesting.GateSort, params *pagination.Params) (*pagination.PagedResult[*traffictesting.Gate], error) {
	return nil, errors.New("not implemented")
}

func TestGetGateHandler_Handle_success(t *testing.T) {
	// Arrange
	name, _ := traffictesting.ParseGateName("test-gate")
	liveURL, _ := traffictesting.ParseGateURL("https://api.example.com")
	shadowURL, _ := traffictesting.ParseGateURL("https://api-staging.example.com")
	expectedGate, _ := traffictesting.NewGate(name, liveURL, shadowURL)

	repo := &mockGateRepositoryForGetGate{
		getByIDFn: func(ctx context.Context, id traffictesting.GateID) (*traffictesting.Gate, error) {
			return expectedGate, nil
		},
	}
	handler := NewGetGateHandler(repo)

	query := GetGateQuery{
		ID: expectedGate.ID.String(),
	}

	// Act
	gate, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, gate)
	assert.Equal(t, expectedGate.ID, gate.ID)
}

func TestGetGateHandler_Handle_invalid_id(t *testing.T) {
	// Arrange
	repo := &mockGateRepositoryForGetGate{}
	handler := NewGetGateHandler(repo)

	query := GetGateQuery{
		ID: "invalid-uuid",
	}

	// Act
	gate, err := handler.Handle(context.Background(), query)

	// Assert
	require.Error(t, err)
	assert.Nil(t, gate)
}

func TestGetGateHandler_Handle_not_found(t *testing.T) {
	// Arrange
	repo := &mockGateRepositoryForGetGate{
		getByIDFn: func(ctx context.Context, id traffictesting.GateID) (*traffictesting.Gate, error) {
			return nil, traffictesting.ErrGateNotFound
		},
	}
	handler := NewGetGateHandler(repo)

	gateID := traffictesting.NewGateID()
	query := GetGateQuery{
		ID: gateID.String(),
	}

	// Act
	gate, err := handler.Handle(context.Background(), query)

	// Assert
	require.Error(t, err)
	assert.Nil(t, gate)
	assert.ErrorIs(t, err, traffictesting.ErrGateNotFound)
}

func TestGetGateHandler_Handle_repository_error(t *testing.T) {
	// Arrange
	expectedErr := errors.New("database connection failed")
	repo := &mockGateRepositoryForGetGate{
		getByIDFn: func(ctx context.Context, id traffictesting.GateID) (*traffictesting.Gate, error) {
			return nil, expectedErr
		},
	}
	handler := NewGetGateHandler(repo)

	gateID := traffictesting.NewGateID()
	query := GetGateQuery{
		ID: gateID.String(),
	}

	// Act
	gate, err := handler.Handle(context.Background(), query)

	// Assert
	require.Error(t, err)
	assert.Nil(t, gate)
	assert.ErrorIs(t, err, expectedErr)
}
