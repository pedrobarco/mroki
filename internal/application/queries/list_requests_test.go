package queries

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pedrobarco/mroki/internal/domain/pagination"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRequestRepositoryForListRequests struct {
	getAllByGateIDFn func(context.Context, traffictesting.GateID, traffictesting.RequestFilters, traffictesting.RequestSort, *pagination.Params) (*pagination.PagedResult[*traffictesting.Request], error)
}

func (m *mockRequestRepositoryForListRequests) Save(ctx context.Context, req *traffictesting.Request) error {
	return errors.New("not implemented")
}

func (m *mockRequestRepositoryForListRequests) GetByID(ctx context.Context, id traffictesting.RequestID, gateID traffictesting.GateID) (*traffictesting.Request, error) {
	return nil, errors.New("not implemented")
}

func (m *mockRequestRepositoryForListRequests) GetAllByGateID(ctx context.Context, gateID traffictesting.GateID, filters traffictesting.RequestFilters, sort traffictesting.RequestSort, params *pagination.Params) (*pagination.PagedResult[*traffictesting.Request], error) {
	if m.getAllByGateIDFn != nil {
		return m.getAllByGateIDFn(ctx, gateID, filters, sort, params)
	}
	return nil, errors.New("not implemented")
}

func TestListRequestsHandler_Handle_success(t *testing.T) {
	// Arrange
	gateID := traffictesting.NewGateID()
	method1, _ := traffictesting.NewHTTPMethod("GET")
	path1, _ := traffictesting.ParsePath("/api/test1")
	method2, _ := traffictesting.NewHTTPMethod("POST")
	path2, _ := traffictesting.ParsePath("/api/test2")

	req1, _ := traffictesting.NewRequest(
		gateID,
		method1,
		path1,
		traffictesting.NewHeaders(map[string][]string{}),
		[]byte{},
		time.Now(),
		traffictesting.Response{},
		traffictesting.Response{},
		traffictesting.Diff{},
	)
	req2, _ := traffictesting.NewRequest(
		gateID,
		method2,
		path2,
		traffictesting.NewHeaders(map[string][]string{}),
		[]byte{},
		time.Now(),
		traffictesting.Response{},
		traffictesting.Response{},
		traffictesting.Diff{},
	)

	repo := &mockRequestRepositoryForListRequests{
		getAllByGateIDFn: func(ctx context.Context, gid traffictesting.GateID, filters traffictesting.RequestFilters, sort traffictesting.RequestSort, params *pagination.Params) (*pagination.PagedResult[*traffictesting.Request], error) {
			assert.Equal(t, gateID, gid)
			return pagination.NewPagedResult([]*traffictesting.Request{req1, req2}, 2, params), nil
		},
	}
	handler := NewListRequestsHandler(repo)

	query := ListRequestsQuery{
		GateID: gateID.String(),
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
}

func TestListRequestsHandler_Handle_invalid_gate_id(t *testing.T) {
	// Arrange
	repo := &mockRequestRepositoryForListRequests{}
	handler := NewListRequestsHandler(repo)

	query := ListRequestsQuery{
		GateID: "invalid-uuid",
		Limit:  10,
		Offset: 0,
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
}

func TestListRequestsHandler_Handle_empty_result(t *testing.T) {
	// Arrange
	gateID := traffictesting.NewGateID()
	repo := &mockRequestRepositoryForListRequests{
		getAllByGateIDFn: func(ctx context.Context, gid traffictesting.GateID, filters traffictesting.RequestFilters, sort traffictesting.RequestSort, params *pagination.Params) (*pagination.PagedResult[*traffictesting.Request], error) {
			return pagination.NewPagedResult([]*traffictesting.Request{}, 0, params), nil
		},
	}
	handler := NewListRequestsHandler(repo)

	query := ListRequestsQuery{
		GateID: gateID.String(),
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

func TestListRequestsHandler_Handle_repository_error(t *testing.T) {
	// Arrange
	expectedErr := errors.New("database connection failed")
	gateID := traffictesting.NewGateID()
	repo := &mockRequestRepositoryForListRequests{
		getAllByGateIDFn: func(ctx context.Context, gid traffictesting.GateID, filters traffictesting.RequestFilters, sort traffictesting.RequestSort, params *pagination.Params) (*pagination.PagedResult[*traffictesting.Request], error) {
			return nil, expectedErr
		},
	}
	handler := NewListRequestsHandler(repo)

	query := ListRequestsQuery{
		GateID: gateID.String(),
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
