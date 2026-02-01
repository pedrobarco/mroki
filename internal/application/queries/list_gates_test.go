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

type mockGateRepositoryForListGates struct {
	getAllFn func(context.Context, *pagination.Params) (*pagination.PagedResult[*traffictesting.Gate], error)
}

func (m *mockGateRepositoryForListGates) Save(ctx context.Context, gate *traffictesting.Gate) error {
	return errors.New("not implemented")
}

func (m *mockGateRepositoryForListGates) GetByID(ctx context.Context, id traffictesting.GateID) (*traffictesting.Gate, error) {
	return nil, errors.New("not implemented")
}

func (m *mockGateRepositoryForListGates) GetAll(ctx context.Context, params *pagination.Params) (*pagination.PagedResult[*traffictesting.Gate], error) {
	if m.getAllFn != nil {
		return m.getAllFn(ctx, params)
	}
	return nil, errors.New("not implemented")
}

func TestListGatesHandler_Handle_success(t *testing.T) {
	// Arrange
	liveURL1, _ := traffictesting.ParseGateURL("https://api1.example.com")
	shadowURL1, _ := traffictesting.ParseGateURL("https://api1-staging.example.com")
	gate1, _ := traffictesting.NewGate(liveURL1, shadowURL1)

	liveURL2, _ := traffictesting.ParseGateURL("https://api2.example.com")
	shadowURL2, _ := traffictesting.ParseGateURL("https://api2-staging.example.com")
	gate2, _ := traffictesting.NewGate(liveURL2, shadowURL2)

	repo := &mockGateRepositoryForListGates{
		getAllFn: func(ctx context.Context, params *pagination.Params) (*pagination.PagedResult[*traffictesting.Gate], error) {
			return pagination.NewPagedResult([]*traffictesting.Gate{gate1, gate2}, 2, params), nil
		},
	}
	handler := NewListGatesHandler(repo)

	query := ListGatesQuery{
		Limit:  10,
		Offset: 0,
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, int64(2), result.Total)
	assert.False(t, result.HasMore)
}

func TestListGatesHandler_Handle_empty_result(t *testing.T) {
	// Arrange
	repo := &mockGateRepositoryForListGates{
		getAllFn: func(ctx context.Context, params *pagination.Params) (*pagination.PagedResult[*traffictesting.Gate], error) {
			return pagination.NewPagedResult([]*traffictesting.Gate{}, 0, params), nil
		},
	}
	handler := NewListGatesHandler(repo)

	query := ListGatesQuery{
		Limit:  10,
		Offset: 0,
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Items)
	assert.Equal(t, int64(0), result.Total)
}

func TestListGatesHandler_Handle_with_pagination(t *testing.T) {
	// Arrange
	repo := &mockGateRepositoryForListGates{
		getAllFn: func(ctx context.Context, params *pagination.Params) (*pagination.PagedResult[*traffictesting.Gate], error) {
			assert.Equal(t, 20, params.Limit())
			assert.Equal(t, 40, params.Offset())
			return pagination.NewPagedResult([]*traffictesting.Gate{}, 100, params), nil
		},
	}
	handler := NewListGatesHandler(repo)

	query := ListGatesQuery{
		Limit:  20,
		Offset: 40,
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	assert.True(t, result.HasMore) // 40 + 20 < 100
}

func TestListGatesHandler_Handle_repository_error(t *testing.T) {
	// Arrange
	expectedErr := errors.New("database connection failed")
	repo := &mockGateRepositoryForListGates{
		getAllFn: func(ctx context.Context, params *pagination.Params) (*pagination.PagedResult[*traffictesting.Gate], error) {
			return nil, expectedErr
		},
	}
	handler := NewListGatesHandler(repo)

	query := ListGatesQuery{
		Limit:  10,
		Offset: 0,
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, expectedErr)
}
