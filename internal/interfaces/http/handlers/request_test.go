package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pedrobarco/mroki/internal/application/commands"
	"github.com/pedrobarco/mroki/internal/application/queries"
	"github.com/pedrobarco/mroki/internal/domain/pagination"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
)

// mockRequestRepository implements traffictesting.RequestRepository for testing
type mockRequestRepository struct {
	saveFunc           func(ctx context.Context, req *traffictesting.Request) error
	getByIDFunc        func(ctx context.Context, id traffictesting.RequestID, gateID traffictesting.GateID) (*traffictesting.Request, error)
	getAllByGateIDFunc func(ctx context.Context, gateID traffictesting.GateID, params *pagination.Params) (*pagination.PagedResult[*traffictesting.Request], error)
}

func (m *mockRequestRepository) Save(ctx context.Context, req *traffictesting.Request) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, req)
	}
	return nil
}

func (m *mockRequestRepository) GetByID(ctx context.Context, id traffictesting.RequestID, gateID traffictesting.GateID) (*traffictesting.Request, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id, gateID)
	}
	return nil, nil
}

func (m *mockRequestRepository) GetAllByGateID(ctx context.Context, gateID traffictesting.GateID, params *pagination.Params) (*pagination.PagedResult[*traffictesting.Request], error) {
	if m.getAllByGateIDFunc != nil {
		return m.getAllByGateIDFunc(ctx, gateID, params)
	}
	return nil, nil
}

func TestCreateRequest_Success(t *testing.T) {
	gateID := traffictesting.NewGateID()
	repo := &mockRequestRepository{
		saveFunc: func(ctx context.Context, req *traffictesting.Request) error {
			return nil
		},
	}
	handler := commands.NewCreateRequestHandler(repo)

	now := time.Now()
	agentID := "test-host-12345678" // Valid format: hostname-8hexchars
	body := map[string]interface{}{
		"agent_id":   agentID,
		"method":     "GET",
		"path":       "/api/test",
		"headers":    map[string][]string{"Content-Type": {"application/json"}},
		"body":       "test body",
		"created_at": now.Format(time.RFC3339Nano),
		"responses": []map[string]interface{}{
			{
				"type":        "live",
				"status_code": 200,
				"headers":     map[string][]string{"Content-Type": {"application/json"}},
				"body":        "response 1",
				"created_at":  now.Format(time.RFC3339Nano),
			},
			{
				"type":        "shadow",
				"status_code": 200,
				"headers":     map[string][]string{"Content-Type": {"application/json"}},
				"body":        "response 2",
				"created_at":  now.Format(time.RFC3339Nano),
			},
		},
		"diff": map[string]interface{}{
			"content": "diff content",
		},
	}

	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/gates/"+gateID.String()+"/requests", bytes.NewBuffer(jsonBody))
	req.SetPathValue("gate_id", gateID.String())
	rec := httptest.NewRecorder()

	appHandler := CreateRequest(handler)
	err := appHandler(rec, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rec.Code)
	}

	var response responseDTO[requestResponseDTO]
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Data.Method != "GET" {
		t.Errorf("expected method GET, got %s", response.Data.Method)
	}

	if response.Data.Path != "/api/test" {
		t.Errorf("expected path /api/test, got %s", response.Data.Path)
	}
}

func TestCreateRequest_InvalidJSON(t *testing.T) {
	gateID := traffictesting.NewGateID()
	repo := &mockRequestRepository{}
	handler := commands.NewCreateRequestHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/gates/"+gateID.String()+"/requests", bytes.NewBufferString("{invalid}"))
	req.SetPathValue("gate_id", gateID.String())
	rec := httptest.NewRecorder()

	appHandler := CreateRequest(handler)
	err := appHandler(rec, req)

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", apiErr.StatusCode)
	}
}

func TestCreateRequest_MissingGateID(t *testing.T) {
	repo := &mockRequestRepository{}
	handler := commands.NewCreateRequestHandler(repo)

	body := `{"agent_id":"test","method":"GET","path":"/test","headers":{},"body":"","responses":[],"diff":{"content":""}}`
	req := httptest.NewRequest(http.MethodPost, "/gates//requests", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	appHandler := CreateRequest(handler)
	err := appHandler(rec, req)

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", apiErr.StatusCode)
	}
}

func TestCreateRequest_InvalidGateID(t *testing.T) {
	repo := &mockRequestRepository{
		saveFunc: func(ctx context.Context, req *traffictesting.Request) error {
			return traffictesting.ErrInvalidGateID
		},
	}
	handler := commands.NewCreateRequestHandler(repo)

	now := time.Now()
	body := map[string]interface{}{
		"agent_id":   "test-agent",
		"method":     "GET",
		"path":       "/test",
		"headers":    map[string][]string{},
		"body":       "",
		"created_at": now.Format(time.RFC3339Nano),
		"responses": []map[string]interface{}{
			{
				"type":        "live",
				"status_code": 200,
				"headers":     map[string][]string{},
				"body":        "",
				"created_at":  now.Format(time.RFC3339Nano),
			},
			{
				"type":        "shadow",
				"status_code": 200,
				"headers":     map[string][]string{},
				"body":        "",
				"created_at":  now.Format(time.RFC3339Nano),
			},
		},
		"diff": map[string]interface{}{
			"content": "",
		},
	}

	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/gates/invalid-id/requests", bytes.NewBuffer(jsonBody))
	req.SetPathValue("gate_id", "invalid-id")
	rec := httptest.NewRecorder()

	appHandler := CreateRequest(handler)
	err := appHandler(rec, req)

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", apiErr.StatusCode)
	}
}

func TestCreateRequest_RepositoryError(t *testing.T) {
	gateID := traffictesting.NewGateID()
	repo := &mockRequestRepository{
		saveFunc: func(ctx context.Context, req *traffictesting.Request) error {
			return errors.New("database error")
		},
	}
	handler := commands.NewCreateRequestHandler(repo)

	now := time.Now()
	body := map[string]interface{}{
		"agent_id":   "test-agent",
		"method":     "GET",
		"path":       "/test",
		"headers":    map[string][]string{},
		"body":       "",
		"created_at": now.Format(time.RFC3339Nano),
		"responses": []map[string]interface{}{
			{
				"type":        "live",
				"status_code": 200,
				"headers":     map[string][]string{},
				"body":        "",
				"created_at":  now.Format(time.RFC3339Nano),
			},
			{
				"type":        "shadow",
				"status_code": 200,
				"headers":     map[string][]string{},
				"body":        "",
				"created_at":  now.Format(time.RFC3339Nano),
			},
		},
		"diff": map[string]interface{}{
			"content": "",
		},
	}

	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/gates/"+gateID.String()+"/requests", bytes.NewBuffer(jsonBody))
	req.SetPathValue("gate_id", gateID.String())
	rec := httptest.NewRecorder()

	appHandler := CreateRequest(handler)
	err := appHandler(rec, req)

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", apiErr.StatusCode)
	}
}

func TestGetRequestByID_Success(t *testing.T) {
	gateID := traffictesting.NewGateID()
	requestID := traffictesting.NewRequestID()

	liveResp, _ := traffictesting.NewResponse(
		traffictesting.ResponseTypeLive,
		200,
		http.Header{"Content-Type": []string{"application/json"}},
		[]byte("live body"),
		time.Now(),
	)
	shadowResp, _ := traffictesting.NewResponse(
		traffictesting.ResponseTypeShadow,
		200,
		http.Header{"Content-Type": []string{"application/json"}},
		[]byte("shadow body"),
		time.Now(),
	)

	diff, _ := traffictesting.NewDiff(liveResp.ID, shadowResp.ID, "diff content")
	expectedRequest, _ := traffictesting.NewRequest(
		gateID,
		"GET",
		"/test",
		http.Header{},
		[]byte("request body"),
		time.Now(),
		[]traffictesting.Response{*liveResp, *shadowResp},
		*diff,
		traffictesting.WithRequestID(requestID),
	)

	repo := &mockRequestRepository{
		getByIDFunc: func(ctx context.Context, id traffictesting.RequestID, gid traffictesting.GateID) (*traffictesting.Request, error) {
			if id != requestID {
				t.Errorf("unexpected request ID: %s", id)
			}
			if gid != gateID {
				t.Errorf("unexpected gate ID: %s", gid)
			}
			return expectedRequest, nil
		},
	}
	handler := queries.NewGetRequestHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/gates/"+gateID.String()+"/requests/"+requestID.String(), nil)
	req.SetPathValue("gate_id", gateID.String())
	req.SetPathValue("request_id", requestID.String())
	rec := httptest.NewRecorder()

	appHandler := GetRequestByID(handler)
	err := appHandler(rec, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response responseDTO[fullRequestResponseDTO]
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Data.ID != requestID.String() {
		t.Errorf("expected ID %s, got %s", requestID.String(), response.Data.ID)
	}

	if len(response.Data.Responses) != 2 {
		t.Errorf("expected 2 responses, got %d", len(response.Data.Responses))
	}
}

func TestGetRequestByID_MissingGateID(t *testing.T) {
	repo := &mockRequestRepository{}
	handler := queries.NewGetRequestHandler(repo)

	requestID := traffictesting.NewRequestID()
	req := httptest.NewRequest(http.MethodGet, "/gates//requests/"+requestID.String(), nil)
	req.SetPathValue("request_id", requestID.String())
	rec := httptest.NewRecorder()

	appHandler := GetRequestByID(handler)
	err := appHandler(rec, req)

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", apiErr.StatusCode)
	}
}

func TestGetRequestByID_MissingRequestID(t *testing.T) {
	gateID := traffictesting.NewGateID()
	repo := &mockRequestRepository{}
	handler := queries.NewGetRequestHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/gates/"+gateID.String()+"/requests/", nil)
	req.SetPathValue("gate_id", gateID.String())
	rec := httptest.NewRecorder()

	appHandler := GetRequestByID(handler)
	err := appHandler(rec, req)

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", apiErr.StatusCode)
	}
}

func TestGetRequestByID_InvalidRequestID(t *testing.T) {
	gateID := traffictesting.NewGateID()
	repo := &mockRequestRepository{
		getByIDFunc: func(ctx context.Context, id traffictesting.RequestID, gid traffictesting.GateID) (*traffictesting.Request, error) {
			return nil, traffictesting.ErrInvalidRequestID
		},
	}
	handler := queries.NewGetRequestHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/gates/"+gateID.String()+"/requests/invalid-id", nil)
	req.SetPathValue("gate_id", gateID.String())
	req.SetPathValue("request_id", "invalid-id")
	rec := httptest.NewRecorder()

	appHandler := GetRequestByID(handler)
	err := appHandler(rec, req)

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", apiErr.StatusCode)
	}
}

func TestGetRequestByID_NotFound(t *testing.T) {
	gateID := traffictesting.NewGateID()
	requestID := traffictesting.NewRequestID()
	repo := &mockRequestRepository{
		getByIDFunc: func(ctx context.Context, id traffictesting.RequestID, gid traffictesting.GateID) (*traffictesting.Request, error) {
			return nil, traffictesting.ErrRequestNotFound
		},
	}
	handler := queries.NewGetRequestHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/gates/"+gateID.String()+"/requests/"+requestID.String(), nil)
	req.SetPathValue("gate_id", gateID.String())
	req.SetPathValue("request_id", requestID.String())
	rec := httptest.NewRecorder()

	appHandler := GetRequestByID(handler)
	err := appHandler(rec, req)

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", apiErr.StatusCode)
	}
}

func TestGetAllRequestsByGateID_Success(t *testing.T) {
	gateID := traffictesting.NewGateID()

	req1, _ := createTestRequest(gateID)
	req2, _ := createTestRequest(gateID)
	requests := []*traffictesting.Request{req1, req2}

	params, _ := pagination.NewParams(10, 0)
	result := pagination.NewPagedResult(requests, 2, params)

	repo := &mockRequestRepository{
		getAllByGateIDFunc: func(ctx context.Context, gid traffictesting.GateID, p *pagination.Params) (*pagination.PagedResult[*traffictesting.Request], error) {
			if gid != gateID {
				t.Errorf("unexpected gate ID: %s", gid)
			}
			return result, nil
		},
	}
	handler := queries.NewListRequestsHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/gates/"+gateID.String()+"/requests?limit=10&offset=0", nil)
	req.SetPathValue("gate_id", gateID.String())
	rec := httptest.NewRecorder()

	appHandler := GetAllRequestsByGateID(handler)
	err := appHandler(rec, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response paginatedResponseDTO[[]requestResponseDTO]
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.Data) != 2 {
		t.Errorf("expected 2 requests, got %d", len(response.Data))
	}

	if response.Pagination.Total != 2 {
		t.Errorf("expected total 2, got %d", response.Pagination.Total)
	}
}

func TestGetAllRequestsByGateID_EmptyResults(t *testing.T) {
	gateID := traffictesting.NewGateID()
	params, _ := pagination.NewParams(50, 0)
	result := pagination.NewPagedResult([]*traffictesting.Request{}, 0, params)

	repo := &mockRequestRepository{
		getAllByGateIDFunc: func(ctx context.Context, gid traffictesting.GateID, p *pagination.Params) (*pagination.PagedResult[*traffictesting.Request], error) {
			return result, nil
		},
	}
	handler := queries.NewListRequestsHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/gates/"+gateID.String()+"/requests", nil)
	req.SetPathValue("gate_id", gateID.String())
	rec := httptest.NewRecorder()

	appHandler := GetAllRequestsByGateID(handler)
	err := appHandler(rec, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var response paginatedResponseDTO[[]requestResponseDTO]
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.Data) != 0 {
		t.Errorf("expected 0 requests, got %d", len(response.Data))
	}
}

func TestGetAllRequestsByGateID_MissingGateID(t *testing.T) {
	repo := &mockRequestRepository{}
	handler := queries.NewListRequestsHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/gates//requests", nil)
	rec := httptest.NewRecorder()

	appHandler := GetAllRequestsByGateID(handler)
	err := appHandler(rec, req)

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", apiErr.StatusCode)
	}
}

func TestGetAllRequestsByGateID_InvalidGateID(t *testing.T) {
	repo := &mockRequestRepository{
		getAllByGateIDFunc: func(ctx context.Context, gid traffictesting.GateID, p *pagination.Params) (*pagination.PagedResult[*traffictesting.Request], error) {
			return nil, traffictesting.ErrInvalidGateID
		},
	}
	handler := queries.NewListRequestsHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/gates/invalid-id/requests", nil)
	req.SetPathValue("gate_id", "invalid-id")
	rec := httptest.NewRecorder()

	appHandler := GetAllRequestsByGateID(handler)
	err := appHandler(rec, req)

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", apiErr.StatusCode)
	}
}

func TestGetAllRequestsByGateID_InvalidPagination(t *testing.T) {
	gateID := traffictesting.NewGateID()
	repo := &mockRequestRepository{}
	handler := queries.NewListRequestsHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/gates/"+gateID.String()+"/requests?limit=invalid", nil)
	req.SetPathValue("gate_id", gateID.String())
	rec := httptest.NewRecorder()

	appHandler := GetAllRequestsByGateID(handler)
	err := appHandler(rec, req)

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", apiErr.StatusCode)
	}
}

// Helper function to create a test request
func createTestRequest(gateID traffictesting.GateID) (*traffictesting.Request, error) {
	liveResp, _ := traffictesting.NewResponse(
		traffictesting.ResponseTypeLive,
		200,
		http.Header{},
		[]byte(""),
		time.Now(),
	)
	shadowResp, _ := traffictesting.NewResponse(
		traffictesting.ResponseTypeShadow,
		200,
		http.Header{},
		[]byte(""),
		time.Now(),
	)

	diff, _ := traffictesting.NewDiff(liveResp.ID, shadowResp.ID, "")
	return traffictesting.NewRequest(
		gateID,
		"GET",
		"/test",
		http.Header{},
		[]byte(""),
		time.Now(),
		[]traffictesting.Response{*liveResp, *shadowResp},
		*diff,
	)
}
