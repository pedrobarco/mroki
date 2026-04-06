package ent_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pedrobarco/mroki/ent/enttest"
	"github.com/pedrobarco/mroki/internal/domain/pagination"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/pedrobarco/mroki/internal/infrastructure/persistence/ent"
	"github.com/pedrobarco/mroki/pkg/diff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRequestWithoutDiff(t *testing.T, gateID traffictesting.GateID) *traffictesting.Request {
	t.Helper()
	method, _ := traffictesting.NewHTTPMethod("GET")
	path, _ := traffictesting.ParsePath("/api/test")

	return &traffictesting.Request{
		ID:        traffictesting.NewRequestID(),
		GateID:    gateID,
		Method:    method,
		Path:      path,
		Headers:   traffictesting.NewHeaders(http.Header{}),
		Body:      []byte{},
		CreatedAt: time.Now(),
	}
}

func newTestRequest(t *testing.T, gateID traffictesting.GateID) *traffictesting.Request {
	t.Helper()
	method, _ := traffictesting.NewHTTPMethod("POST")
	path, _ := traffictesting.ParsePath("/api/test")
	statusCode, _ := traffictesting.ParseStatusCode(200)
	liveRespID := uuid.New()
	shadowRespID := uuid.New()

	return &traffictesting.Request{
		ID:        traffictesting.NewRequestID(),
		GateID:    gateID,
		Method:    method,
		Path:      path,
		Headers:   traffictesting.NewHeaders(http.Header{"Content-Type": []string{"application/json"}}),
		Body:      []byte(`{"test":"data"}`),
		CreatedAt: time.Now(),
		Responses: []traffictesting.Response{
			{
				ID:         liveRespID,
				Type:       traffictesting.ResponseTypeLive,
				StatusCode: statusCode,
				Headers:    traffictesting.NewHeaders(http.Header{}),
				Body:       []byte(`{"status":"ok"}`),
				LatencyMs:  142,
				CreatedAt:  time.Now(),
			},
			{
				ID:         shadowRespID,
				Type:       traffictesting.ResponseTypeShadow,
				StatusCode: statusCode,
				Headers:    traffictesting.NewHeaders(http.Header{}),
				Body:       []byte(`{"status":"ok"}`),
				LatencyMs:  187,
				CreatedAt:  time.Now(),
			},
		},
		Diff: traffictesting.Diff{
			FromResponseID: liveRespID,
			ToResponseID:   shadowRespID,
			Content:        []diff.PatchOp{},
			CreatedAt:      time.Now(),
		},
	}
}

func setupGate(t *testing.T, repo traffictesting.GateRepository) traffictesting.GateID {
	t.Helper()
	liveURL, _ := traffictesting.ParseGateURL("http://live.example.com")
	shadowURL, _ := traffictesting.ParseGateURL("http://shadow.example.com")
	gate, _ := traffictesting.NewGate(nextGateName(), liveURL, shadowURL)
	require.NoError(t, repo.Save(context.Background(), gate))
	return gate.ID
}

func TestRequestRepository_Save_success(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	gateRepo := ent.NewGateRepository(client)
	reqRepo := ent.NewRequestRepository(client)
	gateID := setupGate(t, gateRepo)

	req := newTestRequest(t, gateID)
	err := reqRepo.Save(context.Background(), req)

	assert.NoError(t, err)
}

func TestRequestRepository_GetByID_success(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	gateRepo := ent.NewGateRepository(client)
	reqRepo := ent.NewRequestRepository(client)
	gateID := setupGate(t, gateRepo)

	req := newTestRequest(t, gateID)
	require.NoError(t, reqRepo.Save(context.Background(), req))

	result, err := reqRepo.GetByID(context.Background(), req.ID, gateID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, req.ID.String(), result.ID.String())
	assert.Equal(t, gateID.String(), result.GateID.String())
	assert.Equal(t, "POST", result.Method.String())
	assert.Len(t, result.Responses, 2)
	assert.False(t, result.Diff.IsZero())
}

func TestRequestRepository_GetByID_not_found(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	reqRepo := ent.NewRequestRepository(client)

	requestID := traffictesting.NewRequestID()
	gateID := traffictesting.NewGateID()

	result, err := reqRepo.GetByID(context.Background(), requestID, gateID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, traffictesting.ErrRequestNotFound)
}

func TestRequestRepository_GetAllByGateID_success(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	gateRepo := ent.NewGateRepository(client)
	reqRepo := ent.NewRequestRepository(client)
	gateID := setupGate(t, gateRepo)

	req1 := newTestRequest(t, gateID)
	req2 := newTestRequest(t, gateID)
	require.NoError(t, reqRepo.Save(context.Background(), req1))
	require.NoError(t, reqRepo.Save(context.Background(), req2))

	params, _ := pagination.NewParams(50, 0)
	filters := traffictesting.EmptyRequestFilters()
	sort := traffictesting.DefaultRequestSort()

	result, err := reqRepo.GetAllByGateID(context.Background(), gateID, filters, sort, params)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, int64(2), result.Total)
	assert.False(t, result.HasMore)
}

func TestRequestRepository_GetAllByGateID_eager_loads_metadata(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	gateRepo := ent.NewGateRepository(client)
	reqRepo := ent.NewRequestRepository(client)
	gateID := setupGate(t, gateRepo)

	// Create a request with responses and diff
	reqWithDiff := newTestRequest(t, gateID)
	require.NoError(t, reqRepo.Save(context.Background(), reqWithDiff))

	// Create a request without responses or diff
	reqWithoutDiff := newTestRequestWithoutDiff(t, gateID)
	require.NoError(t, reqRepo.Save(context.Background(), reqWithoutDiff))

	params, _ := pagination.NewParams(50, 0)
	filters := traffictesting.EmptyRequestFilters()
	sort := traffictesting.DefaultRequestSort()

	result, err := reqRepo.GetAllByGateID(context.Background(), gateID, filters, sort, params)

	assert.NoError(t, err)
	assert.Len(t, result.Items, 2)

	// Find the request with diff and verify its metadata
	for _, req := range result.Items {
		switch req.ID {
		case reqWithDiff.ID:
			// Should have eager-loaded responses
			assert.Len(t, req.Responses, 2, "request with diff should have 2 responses")

			var liveResp, shadowResp *traffictesting.Response
			for i := range req.Responses {
				switch req.Responses[i].Type {
				case traffictesting.ResponseTypeLive:
					liveResp = &req.Responses[i]
				case traffictesting.ResponseTypeShadow:
					shadowResp = &req.Responses[i]
				}
			}

			assert.NotNil(t, liveResp, "should have live response")
			assert.NotNil(t, shadowResp, "should have shadow response")
			assert.Equal(t, 200, liveResp.StatusCode.Int())
			assert.Equal(t, 200, shadowResp.StatusCode.Int())
			assert.Equal(t, int64(142), liveResp.LatencyMs)
			assert.Equal(t, int64(187), shadowResp.LatencyMs)

			// Should have diff
			assert.False(t, req.Diff.IsZero(), "request with diff should have non-zero diff")
		case reqWithoutDiff.ID:
			// Should have no responses or diff
			assert.Empty(t, req.Responses, "request without diff should have no responses")
			assert.True(t, req.Diff.IsZero(), "request without diff should have zero diff")
		}
	}
}

func TestRequestRepository_GetAllByGateID_empty(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	reqRepo := ent.NewRequestRepository(client)
	gateID := traffictesting.NewGateID()

	params, _ := pagination.NewParams(50, 0)
	filters := traffictesting.EmptyRequestFilters()
	sort := traffictesting.DefaultRequestSort()

	result, err := reqRepo.GetAllByGateID(context.Background(), gateID, filters, sort, params)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Items)
	assert.Equal(t, int64(0), result.Total)
	assert.False(t, result.HasMore)
}

func TestRequestRepository_GetAllByGateID_with_method_filter(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	gateRepo := ent.NewGateRepository(client)
	reqRepo := ent.NewRequestRepository(client)
	gateID := setupGate(t, gateRepo)

	// Create a POST request
	req1 := newTestRequest(t, gateID)
	require.NoError(t, reqRepo.Save(context.Background(), req1))

	// Create a GET request
	req2 := newTestRequest(t, gateID)
	getMethod, _ := traffictesting.NewHTTPMethod("GET")
	req2.Method = getMethod
	require.NoError(t, reqRepo.Save(context.Background(), req2))

	params, _ := pagination.NewParams(50, 0)
	sort := traffictesting.DefaultRequestSort()

	// Filter for GET only
	filterMethod, _ := traffictesting.NewHTTPMethod("GET")
	filters := traffictesting.NewRequestFilters(
		[]traffictesting.HTTPMethod{filterMethod},
		traffictesting.EmptyPathPattern(),
		traffictesting.EmptyDateRange(),
		nil,
	)

	result, err := reqRepo.GetAllByGateID(context.Background(), gateID, filters, sort, params)

	assert.NoError(t, err)
	assert.Len(t, result.Items, 1)
	assert.Equal(t, "GET", result.Items[0].Method.String())
}

func TestRequestRepository_DeleteOlderThan(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	gateRepo := ent.NewGateRepository(client)
	reqRepo := ent.NewRequestRepository(client)
	gateID := setupGate(t, gateRepo)

	// Create request with old timestamp
	req := newTestRequest(t, gateID)
	req.CreatedAt = time.Now().Add(-48 * time.Hour)
	for i := range req.Responses {
		req.Responses[i].CreatedAt = req.CreatedAt
	}
	require.NoError(t, reqRepo.Save(context.Background(), req))

	// Delete requests older than 24 hours
	count, err := reqRepo.DeleteOlderThan(context.Background(), 24*time.Hour)

	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Verify it's gone
	params, _ := pagination.NewParams(50, 0)
	filters := traffictesting.EmptyRequestFilters()
	sort := traffictesting.DefaultRequestSort()
	result, _ := reqRepo.GetAllByGateID(context.Background(), gateID, filters, sort, params)
	assert.Empty(t, result.Items)
}

func TestRequestRepository_Save_without_diff(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	gateRepo := ent.NewGateRepository(client)
	reqRepo := ent.NewRequestRepository(client)
	gateID := setupGate(t, gateRepo)

	method, _ := traffictesting.NewHTTPMethod("GET")
	path, _ := traffictesting.ParsePath("/test")

	req := &traffictesting.Request{
		ID:        traffictesting.NewRequestID(),
		GateID:    gateID,
		Method:    method,
		Path:      path,
		Headers:   traffictesting.NewHeaders(http.Header{}),
		Body:      []byte{},
		CreatedAt: time.Now(),
	}

	err := reqRepo.Save(context.Background(), req)
	assert.NoError(t, err)

	result, err := reqRepo.GetByID(context.Background(), req.ID, gateID)
	assert.NoError(t, err)
	assert.True(t, result.Diff.IsZero())
	assert.Empty(t, result.Responses)
}
