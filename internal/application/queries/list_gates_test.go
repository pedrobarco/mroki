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
	getAllFn func(context.Context, traffictesting.GateFilters, traffictesting.GateSort, *pagination.Params) (*pagination.PagedResult[*traffictesting.Gate], error)
}

func (m *mockGateRepositoryForListGates) Save(ctx context.Context, gate *traffictesting.Gate) error {
	return errors.New("not implemented")
}

func (m *mockGateRepositoryForListGates) GetByID(ctx context.Context, id traffictesting.GateID) (*traffictesting.Gate, error) {
	return nil, errors.New("not implemented")
}

func (m *mockGateRepositoryForListGates) GetAll(ctx context.Context, filters traffictesting.GateFilters, sort traffictesting.GateSort, params *pagination.Params) (*pagination.PagedResult[*traffictesting.Gate], error) {
	if m.getAllFn != nil {
		return m.getAllFn(ctx, filters, sort, params)
	}
	return nil, errors.New("not implemented")
}

func TestListGatesHandler_Handle_success(t *testing.T) {
	// Arrange
	name1, _ := traffictesting.ParseGateName("gate-1")
	liveURL1, _ := traffictesting.ParseGateURL("https://api1.example.com")
	shadowURL1, _ := traffictesting.ParseGateURL("https://api1-staging.example.com")
	gate1, _ := traffictesting.NewGate(name1, liveURL1, shadowURL1)

	name2, _ := traffictesting.ParseGateName("gate-2")
	liveURL2, _ := traffictesting.ParseGateURL("https://api2.example.com")
	shadowURL2, _ := traffictesting.ParseGateURL("https://api2-staging.example.com")
	gate2, _ := traffictesting.NewGate(name2, liveURL2, shadowURL2)

	repo := &mockGateRepositoryForListGates{
		getAllFn: func(ctx context.Context, filters traffictesting.GateFilters, sort traffictesting.GateSort, params *pagination.Params) (*pagination.PagedResult[*traffictesting.Gate], error) {
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
		getAllFn: func(ctx context.Context, filters traffictesting.GateFilters, sort traffictesting.GateSort, params *pagination.Params) (*pagination.PagedResult[*traffictesting.Gate], error) {
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
		getAllFn: func(ctx context.Context, filters traffictesting.GateFilters, sort traffictesting.GateSort, params *pagination.Params) (*pagination.PagedResult[*traffictesting.Gate], error) {
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
		getAllFn: func(ctx context.Context, filters traffictesting.GateFilters, sort traffictesting.GateSort, params *pagination.Params) (*pagination.PagedResult[*traffictesting.Gate], error) {
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


func TestListGatesHandler_Handle_with_filters(t *testing.T) {
	// Arrange
	name, _ := traffictesting.ParseGateName("filter-gate")
	liveURL, _ := traffictesting.ParseGateURL("https://api.example.com")
	shadowURL, _ := traffictesting.ParseGateURL("https://api-staging.example.com")
	gate1, _ := traffictesting.NewGate(name, liveURL, shadowURL)

	repo := &mockGateRepositoryForListGates{
		getAllFn: func(ctx context.Context, filters traffictesting.GateFilters, sort traffictesting.GateSort, params *pagination.Params) (*pagination.PagedResult[*traffictesting.Gate], error) {
			// Verify filters are passed through
			assert.True(t, filters.HasLiveURLFilter())
			assert.Equal(t, "example", filters.LiveURL())
			assert.True(t, filters.HasShadowURLFilter())
			assert.Equal(t, "staging", filters.ShadowURL())
			return pagination.NewPagedResult([]*traffictesting.Gate{gate1}, 1, params), nil
		},
	}
	handler := NewListGatesHandler(repo)

	query := ListGatesQuery{
		Limit:     10,
		Offset:    0,
		LiveURL:   "example",
		ShadowURL: "staging",
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Items, 1)
}

func TestListGatesHandler_Handle_with_sort(t *testing.T) {
	// Arrange
	repo := &mockGateRepositoryForListGates{
		getAllFn: func(ctx context.Context, filters traffictesting.GateFilters, sort traffictesting.GateSort, params *pagination.Params) (*pagination.PagedResult[*traffictesting.Gate], error) {
			// Verify sort is passed through
			assert.True(t, sort.Field().IsLiveURL())
			assert.True(t, sort.Order().IsAsc())
			return pagination.NewPagedResult([]*traffictesting.Gate{}, 0, params), nil
		},
	}
	handler := NewListGatesHandler(repo)

	query := ListGatesQuery{
		Limit:     10,
		Offset:    0,
		SortField: "live_url",
		SortOrder: "asc",
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestListGatesHandler_Handle_invalid_sort_field(t *testing.T) {
	// Arrange
	repo := &mockGateRepositoryForListGates{}
	handler := NewListGatesHandler(repo)

	query := ListGatesQuery{
		Limit:     10,
		Offset:    0,
		SortField: "invalid_field",
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, traffictesting.ErrInvalidGateSort)
}

func TestListGatesHandler_Handle_invalid_sort_order(t *testing.T) {
	// Arrange
	repo := &mockGateRepositoryForListGates{}
	handler := NewListGatesHandler(repo)

	query := ListGatesQuery{
		Limit:     10,
		Offset:    0,
		SortOrder: "invalid_order",
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, traffictesting.ErrInvalidGateSort)
}

func TestListGatesHandler_Handle_default_sort(t *testing.T) {
	// Arrange
	repo := &mockGateRepositoryForListGates{
		getAllFn: func(ctx context.Context, filters traffictesting.GateFilters, sort traffictesting.GateSort, params *pagination.Params) (*pagination.PagedResult[*traffictesting.Gate], error) {
			// Default sort should be id asc
			assert.True(t, sort.Field().IsID())
			assert.True(t, sort.Order().IsDesc()) // NewSortOrder defaults to desc
			return pagination.NewPagedResult([]*traffictesting.Gate{}, 0, params), nil
		},
	}
	handler := NewListGatesHandler(repo)

	query := ListGatesQuery{
		Limit:  10,
		Offset: 0,
		// No sort field or order — should use defaults
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestListGatesHandler_Handle_empty_filters(t *testing.T) {
	// Arrange
	repo := &mockGateRepositoryForListGates{
		getAllFn: func(ctx context.Context, filters traffictesting.GateFilters, sort traffictesting.GateSort, params *pagination.Params) (*pagination.PagedResult[*traffictesting.Gate], error) {
			// Empty strings should produce empty filters
			assert.True(t, filters.IsEmpty())
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
}