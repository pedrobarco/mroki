package queries

import (
	"context"
	"encoding/json"
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
		"",
		traffictesting.NewHeaders(map[string][]string{"Content-Type": {"application/json"}}),
		[]byte(`{"test":"data"}`),
		time.Now(),
		traffictesting.Response{},
		traffictesting.Response{},
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

func TestGetRequestHandler_Handle_sort_arrays_sorts_response_bodies(t *testing.T) {
	// Arrange
	gateID := traffictesting.NewGateID()
	requestID := traffictesting.NewRequestID()
	method, _ := traffictesting.NewHTTPMethod("GET")
	path, _ := traffictesting.ParsePath("/api/test")

	liveBody := json.RawMessage(`{"items":[3,1,2],"tags":["z","a"]}`)
	shadowBody := json.RawMessage(`{"items":[2,3,1],"tags":["b","a"]}`)

	sortConfig, _ := traffictesting.NewDiffConfig(nil, nil, 0, true)

	req, _ := traffictesting.NewRequest(
		gateID,
		method,
		path,
		"",
		traffictesting.NewHeaders(map[string][]string{"Content-Type": {"application/json"}}),
		nil,
		time.Now(),
		traffictesting.Response{Body: liveBody},
		traffictesting.Response{Body: shadowBody},
		traffictesting.Diff{Config: sortConfig},
	)
	req.ID = requestID

	repo := &mockRequestRepositoryForGetRequest{
		getByIDFn: func(ctx context.Context, id traffictesting.RequestID, gid traffictesting.GateID) (*traffictesting.Request, error) {
			return req, nil
		},
	}
	handler := NewGetRequestHandler(repo)

	query := GetRequestQuery{
		ID:     requestID.String(),
		GateID: gateID.String(),
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify bodies are sorted
	var liveResult, shadowResult map[string]any
	require.NoError(t, json.Unmarshal(result.LiveResponse.Body, &liveResult))
	require.NoError(t, json.Unmarshal(result.ShadowResponse.Body, &shadowResult))

	assert.Equal(t, []any{float64(1), float64(2), float64(3)}, liveResult["items"])
	assert.Equal(t, []any{"a", "z"}, liveResult["tags"])
	assert.Equal(t, []any{float64(1), float64(2), float64(3)}, shadowResult["items"])
	assert.Equal(t, []any{"a", "b"}, shadowResult["tags"])
}

func TestGetRequestHandler_Handle_no_sort_arrays_leaves_bodies_unchanged(t *testing.T) {
	// Arrange
	gateID := traffictesting.NewGateID()
	requestID := traffictesting.NewRequestID()
	method, _ := traffictesting.NewHTTPMethod("GET")
	path, _ := traffictesting.ParsePath("/api/test")

	liveBody := json.RawMessage(`{"items":[3,1,2]}`)

	noSortConfig, _ := traffictesting.NewDiffConfig(nil, nil, 0, false)

	req, _ := traffictesting.NewRequest(
		gateID,
		method,
		path,
		"",
		traffictesting.NewHeaders(map[string][]string{"Content-Type": {"application/json"}}),
		nil,
		time.Now(),
		traffictesting.Response{Body: liveBody},
		traffictesting.Response{},
		traffictesting.Diff{Config: noSortConfig},
	)
	req.ID = requestID

	repo := &mockRequestRepositoryForGetRequest{
		getByIDFn: func(ctx context.Context, id traffictesting.RequestID, gid traffictesting.GateID) (*traffictesting.Request, error) {
			return req, nil
		},
	}
	handler := NewGetRequestHandler(repo)

	query := GetRequestQuery{
		ID:     requestID.String(),
		GateID: gateID.String(),
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)

	// Body should remain in original order
	var liveResult map[string]any
	require.NoError(t, json.Unmarshal(result.LiveResponse.Body, &liveResult))
	assert.Equal(t, []any{float64(3), float64(1), float64(2)}, liveResult["items"])
}

func TestSortResponseBody_nil_body(t *testing.T) {
	resp := &traffictesting.Response{Body: nil}
	sortResponseBody(resp)
	assert.Nil(t, resp.Body)
}

func TestSortResponseBody_empty_body(t *testing.T) {
	resp := &traffictesting.Response{Body: json.RawMessage{}}
	sortResponseBody(resp)
	assert.Empty(t, resp.Body)
}

func TestSortResponseBody_non_json_body(t *testing.T) {
	original := json.RawMessage(`not valid json`)
	resp := &traffictesting.Response{Body: original}
	sortResponseBody(resp)
	assert.Equal(t, original, resp.Body)
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
