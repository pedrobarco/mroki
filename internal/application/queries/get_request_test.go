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

type mockRequestRepositoryForGetRequest struct {
	getByIDFn func(context.Context, traffictesting.RequestID, traffictesting.GateID) (*traffictesting.Request, error)
}

func (m *mockRequestRepositoryForGetRequest) Save(ctx context.Context, req *traffictesting.Request) error {
	return errors.New("not implemented")
}

func (m *mockRequestRepositoryForGetRequest) GetByID(ctx context.Context, id traffictesting.RequestID, gateID traffictesting.GateID) (*traffictesting.Request, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id, gateID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRequestRepositoryForGetRequest) GetAllByGateID(ctx context.Context, gateID traffictesting.GateID, filters traffictesting.RequestFilters, sort traffictesting.RequestSort, params *pagination.Params) (*pagination.PagedResult[*traffictesting.Request], error) {
	return nil, errors.New("not implemented")
}

func TestGetRequestHandler_Handle_success(t *testing.T) {
	// Arrange
	gateID := traffictesting.NewGateID()
	requestID := traffictesting.NewRequestID()
	method, _ := traffictesting.NewHTTPMethod("GET")
	path, _ := traffictesting.ParsePath("/api/test")

	expectedRequest, _ := traffictesting.NewRequest(
		gateID,
		method,
		path,
		traffictesting.NewHeaders(map[string][]string{"Content-Type": {"application/json"}}),
		[]byte(`{"test":"data"}`),
		time.Now(),
		[]traffictesting.Response{},
		traffictesting.Diff{},
	)
	expectedRequest.ID = requestID

	repo := &mockRequestRepositoryForGetRequest{
		getByIDFn: func(ctx context.Context, id traffictesting.RequestID, gid traffictesting.GateID) (*traffictesting.Request, error) {
			assert.Equal(t, requestID, id)
			assert.Equal(t, gateID, gid)
			return expectedRequest, nil
		},
	}
	handler := NewGetRequestHandler(repo)

	query := GetRequestQuery{
		ID:     requestID.String(),
		GateID: gateID.String(),
	}

	// Act
	req, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, req)
	assert.Equal(t, requestID, req.ID)
	assert.Equal(t, gateID, req.GateID)
}

func TestGetRequestHandler_Handle_invalid_request_id(t *testing.T) {
	// Arrange
	repo := &mockRequestRepositoryForGetRequest{}
	handler := NewGetRequestHandler(repo)

	gateID := traffictesting.NewGateID()
	query := GetRequestQuery{
		ID:     "invalid-uuid",
		GateID: gateID.String(),
	}

	// Act
	req, err := handler.Handle(context.Background(), query)

	// Assert
	require.Error(t, err)
	assert.Nil(t, req)
}

func TestGetRequestHandler_Handle_invalid_gate_id(t *testing.T) {
	// Arrange
	repo := &mockRequestRepositoryForGetRequest{}
	handler := NewGetRequestHandler(repo)

	requestID := traffictesting.NewRequestID()
	query := GetRequestQuery{
		ID:     requestID.String(),
		GateID: "invalid-uuid",
	}

	// Act
	req, err := handler.Handle(context.Background(), query)

	// Assert
	require.Error(t, err)
	assert.Nil(t, req)
}

func TestGetRequestHandler_Handle_not_found(t *testing.T) {
	// Arrange
	repo := &mockRequestRepositoryForGetRequest{
		getByIDFn: func(ctx context.Context, id traffictesting.RequestID, gateID traffictesting.GateID) (*traffictesting.Request, error) {
			return nil, traffictesting.ErrRequestNotFound
		},
	}
	handler := NewGetRequestHandler(repo)

	requestID := traffictesting.NewRequestID()
	gateID := traffictesting.NewGateID()
	query := GetRequestQuery{
		ID:     requestID.String(),
		GateID: gateID.String(),
	}

	// Act
	req, err := handler.Handle(context.Background(), query)

	// Assert
	require.Error(t, err)
	assert.Nil(t, req)
	assert.ErrorIs(t, err, traffictesting.ErrRequestNotFound)
}
