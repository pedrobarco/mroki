package commands

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/pedrobarco/mroki/internal/domain/pagination"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRequestRepository is a mock implementation of RequestRepository for testing
type mockRequestRepository struct {
	saveFn func(context.Context, *traffictesting.Request) error
}

func (m *mockRequestRepository) Save(ctx context.Context, req *traffictesting.Request) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, req)
	}
	return nil
}

func (m *mockRequestRepository) GetByID(ctx context.Context, id traffictesting.RequestID, gateID traffictesting.GateID) (*traffictesting.Request, error) {
	return nil, errors.New("not implemented")
}

func (m *mockRequestRepository) GetAllByGateID(ctx context.Context, gateID traffictesting.GateID, filters traffictesting.RequestFilters, sort traffictesting.RequestSort, params *pagination.Params) (*pagination.PagedResult[*traffictesting.Request], error) {
	return nil, errors.New("not implemented")
}

// mockGateRepoForRequest is a minimal mock of GateRepository for create_request tests
type mockGateRepoForRequest struct{}

func (m *mockGateRepoForRequest) Save(ctx context.Context, gate *traffictesting.Gate) error {
	return nil
}

func (m *mockGateRepoForRequest) Update(ctx context.Context, gate *traffictesting.Gate) error {
	return nil
}

func (m *mockGateRepoForRequest) Delete(ctx context.Context, id traffictesting.GateID) error {
	return nil
}

func (m *mockGateRepoForRequest) GetByID(ctx context.Context, id traffictesting.GateID) (*traffictesting.Gate, error) {
	name, _ := traffictesting.ParseGateName("test")
	live, _ := traffictesting.ParseGateURL("http://live.example.com")
	shadow, _ := traffictesting.ParseGateURL("http://shadow.example.com")
	gate, _ := traffictesting.NewGate(name, live, shadow, traffictesting.WithGateID(id))
	return gate, nil
}

func (m *mockGateRepoForRequest) GetAll(ctx context.Context, filters traffictesting.GateFilters, sort traffictesting.GateSort, params *pagination.Params) (*pagination.PagedResult[*traffictesting.Gate], error) {
	return nil, errors.New("not implemented")
}

func TestCreateRequestHandler_Handle_success(t *testing.T) {
	// Arrange
	repo := &mockRequestRepository{
		saveFn: func(ctx context.Context, req *traffictesting.Request) error {
			assert.NotNil(t, req)
			assert.False(t, req.ID.IsZero())
			assert.NotEqual(t, 0, req.LiveResponse.StatusCode.Int())
			assert.NotEqual(t, 0, req.ShadowResponse.StatusCode.Int())
			return nil
		},
	}
	handler := NewCreateRequestHandler(repo, &mockGateRepoForRequest{})

	gateID := traffictesting.NewGateID()
	cmd := CreateRequestCommand{
		GateID: gateID.String(),
		Method: "GET",
		Path:   "/api/test",
		Headers: map[string][]string{
			"Content-Type": {"application/json"},
		},
		Body:      []byte(`{"test": "data"}`),
		CreatedAt: time.Now(),
		LiveResponse: CreateRequestResponseProps{
			StatusCode: 200,
			Headers:    http.Header{"Content-Type": []string{"application/json"}},
			Body:       []byte(`{"result": "ok"}`),
			CreatedAt:  time.Now(),
		},
		ShadowResponse: CreateRequestResponseProps{
			StatusCode: 200,
			Headers:    http.Header{"Content-Type": []string{"application/json"}},
			Body:       []byte(`{"result": "ok"}`),
			CreatedAt:  time.Now(),
		},
		Diff: &CreateRequestDiffProps{
			Content: nil,
		},
	}

	// Act
	req, err := handler.Handle(context.Background(), cmd)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, req)
	assert.False(t, req.ID.IsZero())
	assert.Equal(t, gateID, req.GateID)
	assert.Equal(t, "GET", req.Method.String())
	assert.Equal(t, "/api/test", req.Path.String())
	assert.Equal(t, 200, req.LiveResponse.StatusCode.Int())
	assert.Equal(t, 200, req.ShadowResponse.StatusCode.Int())
}

func TestCreateRequestHandler_Handle_server_side_diff_computation(t *testing.T) {
	// When Diff is nil (API mode), the handler should compute the diff server-side
	var savedReq *traffictesting.Request
	repo := &mockRequestRepository{
		saveFn: func(ctx context.Context, req *traffictesting.Request) error {
			savedReq = req
			return nil
		},
	}
	handler := NewCreateRequestHandler(repo, &mockGateRepoForRequest{})

	gateID := traffictesting.NewGateID()
	cmd := CreateRequestCommand{
		GateID: gateID.String(),
		Method: "GET",
		Path:   "/api/test",
		Headers: map[string][]string{
			"Content-Type": {"application/json"},
		},
		Body:      []byte(`{}`),
		CreatedAt: time.Now(),
		LiveResponse: CreateRequestResponseProps{
			StatusCode: 200,
			Headers:    http.Header{"Content-Type": []string{"application/json"}},
			Body:       b64(`{"user":"alice"}`),
			CreatedAt:  time.Now(),
		},
		ShadowResponse: CreateRequestResponseProps{
			StatusCode: 200,
			Headers:    http.Header{"Content-Type": []string{"application/json"}},
			Body:       b64(`{"user":"bob"}`),
			CreatedAt:  time.Now(),
		},
		Diff: nil, // No diff provided — should be computed server-side
	}

	req, err := handler.Handle(context.Background(), cmd)

	require.NoError(t, err)
	require.NotNil(t, req)
	require.NotNil(t, savedReq)

	// The diff should have been computed and contain a replace op on /body/user
	assert.NotEmpty(t, savedReq.Diff.Content, "server-side diff should detect differences")

	found := false
	for _, op := range savedReq.Diff.Content {
		if op.Path == "/body/user" && op.Op == "replace" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected replace op on /body/user, got: %v", savedReq.Diff.Content)
}

func TestCreateRequestHandler_Handle_server_side_diff_identical_responses(t *testing.T) {
	// When responses are identical and Diff is nil, server-side diff should produce empty ops
	var savedReq *traffictesting.Request
	repo := &mockRequestRepository{
		saveFn: func(ctx context.Context, req *traffictesting.Request) error {
			savedReq = req
			return nil
		},
	}
	handler := NewCreateRequestHandler(repo, &mockGateRepoForRequest{})

	gateID := traffictesting.NewGateID()
	cmd := CreateRequestCommand{
		GateID:    gateID.String(),
		Method:    "GET",
		Path:      "/api/test",
		Headers:   map[string][]string{},
		Body:      []byte(`{}`),
		CreatedAt: time.Now(),
		LiveResponse: CreateRequestResponseProps{
			StatusCode: 200,
			Headers:    http.Header{},
			Body:       b64(`{"status":"ok"}`),
			CreatedAt:  time.Now(),
		},
		ShadowResponse: CreateRequestResponseProps{
			StatusCode: 200,
			Headers:    http.Header{},
			Body:       b64(`{"status":"ok"}`),
			CreatedAt:  time.Now(),
		},
		Diff: nil, // No diff provided
	}

	req, err := handler.Handle(context.Background(), cmd)

	require.NoError(t, err)
	require.NotNil(t, req)
	require.NotNil(t, savedReq)
	assert.Empty(t, savedReq.Diff.Content, "identical responses should produce empty diff")
}

func TestCreateRequestHandler_Handle_with_custom_ids(t *testing.T) {
	// Arrange
	repo := &mockRequestRepository{
		saveFn: func(ctx context.Context, req *traffictesting.Request) error {
			return nil
		},
	}
	handler := NewCreateRequestHandler(repo, &mockGateRepoForRequest{})

	gateID := traffictesting.NewGateID()
	requestID := traffictesting.NewRequestID()

	cmd := CreateRequestCommand{
		ID:        requestID.String(),
		GateID:    gateID.String(),
		Method:    "POST",
		Path:      "/api/create",
		Headers:   map[string][]string{},
		Body:      []byte(`{"key": "value"}`),
		CreatedAt: time.Now(),
		LiveResponse: CreateRequestResponseProps{
			StatusCode: 201,
			Headers:    http.Header{},
			Body:       []byte(`{"id": 1}`),
			CreatedAt:  time.Now(),
		},
		ShadowResponse: CreateRequestResponseProps{
			StatusCode: 201,
			Headers:    http.Header{},
			Body:       []byte(`{"id": 1}`),
			CreatedAt:  time.Now(),
		},
		Diff: &CreateRequestDiffProps{
			Content: nil,
		},
	}

	// Act
	req, err := handler.Handle(context.Background(), cmd)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, req)
	assert.Equal(t, requestID, req.ID)
	assert.Equal(t, gateID, req.GateID)
}

func TestCreateRequestHandler_Handle_invalid_gate_id(t *testing.T) {
	// Arrange
	repo := &mockRequestRepository{}
	handler := NewCreateRequestHandler(repo, &mockGateRepoForRequest{})

	cmd := CreateRequestCommand{
		GateID:         "invalid-uuid",
		Method:         "GET",
		Path:           "/test",
		LiveResponse:   CreateRequestResponseProps{StatusCode: 200, CreatedAt: time.Now()},
		ShadowResponse: CreateRequestResponseProps{StatusCode: 200, CreatedAt: time.Now()},
	}

	// Act
	req, err := handler.Handle(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	assert.Nil(t, req)
}

func TestCreateRequestHandler_Handle_invalid_request_id(t *testing.T) {
	// Arrange
	repo := &mockRequestRepository{}
	handler := NewCreateRequestHandler(repo, &mockGateRepoForRequest{})

	gateID := traffictesting.NewGateID()
	cmd := CreateRequestCommand{
		ID:             "invalid-uuid",
		GateID:         gateID.String(),
		Method:         "GET",
		Path:           "/test",
		LiveResponse:   CreateRequestResponseProps{StatusCode: 200, CreatedAt: time.Now()},
		ShadowResponse: CreateRequestResponseProps{StatusCode: 200, CreatedAt: time.Now()},
	}

	// Act
	req, err := handler.Handle(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	assert.Nil(t, req)
}

func TestCreateRequestHandler_Handle_invalid_live_status_code(t *testing.T) {
	// Arrange
	repo := &mockRequestRepository{}
	handler := NewCreateRequestHandler(repo, &mockGateRepoForRequest{})

	gateID := traffictesting.NewGateID()
	cmd := CreateRequestCommand{
		GateID:         gateID.String(),
		Method:         "GET",
		Path:           "/test",
		LiveResponse:   CreateRequestResponseProps{StatusCode: 999, CreatedAt: time.Now()},
		ShadowResponse: CreateRequestResponseProps{StatusCode: 200, CreatedAt: time.Now()},
	}

	// Act
	req, err := handler.Handle(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	assert.Nil(t, req)
	assert.Contains(t, err.Error(), "live response")
}

func TestCreateRequestHandler_Handle_invalid_shadow_status_code(t *testing.T) {
	// Arrange
	repo := &mockRequestRepository{}
	handler := NewCreateRequestHandler(repo, &mockGateRepoForRequest{})

	gateID := traffictesting.NewGateID()
	cmd := CreateRequestCommand{
		GateID:         gateID.String(),
		Method:         "GET",
		Path:           "/test",
		LiveResponse:   CreateRequestResponseProps{StatusCode: 200, CreatedAt: time.Now()},
		ShadowResponse: CreateRequestResponseProps{StatusCode: 999, CreatedAt: time.Now()},
	}

	// Act
	req, err := handler.Handle(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	assert.Nil(t, req)
	assert.Contains(t, err.Error(), "shadow response")
}

func TestCreateRequestHandler_Handle_repository_error(t *testing.T) {
	// Arrange
	expectedErr := errors.New("database connection failed")
	repo := &mockRequestRepository{
		saveFn: func(ctx context.Context, req *traffictesting.Request) error {
			return expectedErr
		},
	}
	handler := NewCreateRequestHandler(repo, &mockGateRepoForRequest{})

	gateID := traffictesting.NewGateID()
	cmd := CreateRequestCommand{
		GateID:         gateID.String(),
		Method:         "GET",
		Path:           "/test",
		LiveResponse:   CreateRequestResponseProps{StatusCode: 200, CreatedAt: time.Now()},
		ShadowResponse: CreateRequestResponseProps{StatusCode: 200, CreatedAt: time.Now()},
		Diff:           &CreateRequestDiffProps{Content: nil},
	}

	// Act
	req, err := handler.Handle(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	assert.Nil(t, req)
	assert.Contains(t, err.Error(), "failed to save request")
	assert.ErrorIs(t, err, expectedErr)
}

// --- computeDiff tests ---

func b64(s string) []byte {
	return []byte(base64.StdEncoding.EncodeToString([]byte(s)))
}

func TestComputeDiff_identical_responses(t *testing.T) {
	live := CreateRequestResponseProps{StatusCode: 200, Headers: http.Header{}, Body: b64(`{"ok":true}`)}
	shadow := CreateRequestResponseProps{StatusCode: 200, Headers: http.Header{}, Body: b64(`{"ok":true}`)}

	ops, err := computeDiff(live, shadow)

	require.NoError(t, err)
	assert.Empty(t, ops)
}

func TestComputeDiff_different_bodies(t *testing.T) {
	live := CreateRequestResponseProps{StatusCode: 200, Headers: http.Header{}, Body: b64(`{"user":"alice"}`)}
	shadow := CreateRequestResponseProps{StatusCode: 200, Headers: http.Header{}, Body: b64(`{"user":"bob"}`)}

	ops, err := computeDiff(live, shadow)

	require.NoError(t, err)
	require.NotEmpty(t, ops)

	// Should have a replace op on /body/user
	found := false
	for _, op := range ops {
		if op.Path == "/body/user" && op.Op == "replace" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected replace op on /body/user, got: %v", ops)
}

func TestComputeDiff_different_status_codes(t *testing.T) {
	live := CreateRequestResponseProps{StatusCode: 200, Headers: http.Header{}, Body: b64(`{"ok":true}`)}
	shadow := CreateRequestResponseProps{StatusCode: 500, Headers: http.Header{}, Body: b64(`{"ok":false}`)}

	ops, err := computeDiff(live, shadow)

	require.NoError(t, err)
	require.NotEmpty(t, ops)

	paths := map[string]bool{}
	for _, op := range ops {
		paths[op.Path] = true
	}
	assert.True(t, paths["/statusCode"], "expected diff on /statusCode")
}

func TestComputeDiff_different_headers(t *testing.T) {
	live := CreateRequestResponseProps{
		StatusCode: 200,
		Headers:    http.Header{"X-Req-Id": []string{"abc"}},
		Body:       b64(`{"ok":true}`),
	}
	shadow := CreateRequestResponseProps{
		StatusCode: 200,
		Headers:    http.Header{"X-Req-Id": []string{"def"}},
		Body:       b64(`{"ok":true}`),
	}

	ops, err := computeDiff(live, shadow)

	require.NoError(t, err)
	require.NotEmpty(t, ops, "expected diff on headers")
}

func TestComputeDiff_invalid_base64_body(t *testing.T) {
	live := CreateRequestResponseProps{StatusCode: 200, Headers: http.Header{}, Body: []byte("not-valid-base64!!!")}
	shadow := CreateRequestResponseProps{StatusCode: 200, Headers: http.Header{}, Body: b64(`{"ok":true}`)}

	_, err := computeDiff(live, shadow)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode live response body")
}