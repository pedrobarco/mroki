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

func TestCreateRequestHandler_Handle_success(t *testing.T) {
	// Arrange
	repo := &mockRequestRepository{
		saveFn: func(ctx context.Context, req *traffictesting.Request) error {
			assert.NotNil(t, req)
			assert.False(t, req.ID.IsZero())
			assert.Len(t, req.Responses, 2)
			return nil
		},
	}
	handler := NewCreateRequestHandler(repo)

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
		Responses: []CreateRequestResponseProps{
			{
				Type:       "live",
				StatusCode: 200,
				Headers:    http.Header{"Content-Type": []string{"application/json"}},
				Body:       []byte(`{"result": "ok"}`),
				CreatedAt:  time.Now(),
			},
			{
				Type:       "shadow",
				StatusCode: 200,
				Headers:    http.Header{"Content-Type": []string{"application/json"}},
				Body:       []byte(`{"result": "ok"}`),
				CreatedAt:  time.Now(),
			},
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
	assert.Len(t, req.Responses, 2)
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
	handler := NewCreateRequestHandler(repo)

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
		Responses: []CreateRequestResponseProps{
			{
				Type:       "live",
				StatusCode: 200,
				Headers:    http.Header{"Content-Type": []string{"application/json"}},
				Body:       b64(`{"user":"alice"}`),
				CreatedAt:  time.Now(),
			},
			{
				Type:       "shadow",
				StatusCode: 200,
				Headers:    http.Header{"Content-Type": []string{"application/json"}},
				Body:       b64(`{"user":"bob"}`),
				CreatedAt:  time.Now(),
			},
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
	handler := NewCreateRequestHandler(repo)

	gateID := traffictesting.NewGateID()
	cmd := CreateRequestCommand{
		GateID:    gateID.String(),
		Method:    "GET",
		Path:      "/api/test",
		Headers:   map[string][]string{},
		Body:      []byte(`{}`),
		CreatedAt: time.Now(),
		Responses: []CreateRequestResponseProps{
			{
				Type:       "live",
				StatusCode: 200,
				Headers:    http.Header{},
				Body:       b64(`{"status":"ok"}`),
				CreatedAt:  time.Now(),
			},
			{
				Type:       "shadow",
				StatusCode: 200,
				Headers:    http.Header{},
				Body:       b64(`{"status":"ok"}`),
				CreatedAt:  time.Now(),
			},
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
	handler := NewCreateRequestHandler(repo)

	gateID := traffictesting.NewGateID()
	requestID := traffictesting.NewRequestID()
	agentID := traffictesting.NewAgentIDWithHostname("test-agent")

	cmd := CreateRequestCommand{
		ID:        requestID.String(),
		GateID:    gateID.String(),
		AgentID:   agentID.String(),
		Method:    "POST",
		Path:      "/api/create",
		Headers:   map[string][]string{},
		Body:      []byte(`{"key": "value"}`),
		CreatedAt: time.Now(),
		Responses: []CreateRequestResponseProps{
			{
				Type:       "live",
				StatusCode: 201,
				Headers:    http.Header{},
				Body:       []byte(`{"id": 1}`),
				CreatedAt:  time.Now(),
			},
			{
				Type:       "shadow",
				StatusCode: 201,
				Headers:    http.Header{},
				Body:       []byte(`{"id": 1}`),
				CreatedAt:  time.Now(),
			},
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
	assert.Equal(t, agentID, req.AgentID)
}

func TestCreateRequestHandler_Handle_invalid_gate_id(t *testing.T) {
	// Arrange
	repo := &mockRequestRepository{}
	handler := NewCreateRequestHandler(repo)

	cmd := CreateRequestCommand{
		GateID: "invalid-uuid",
		Method: "GET",
		Path:   "/test",
		Responses: []CreateRequestResponseProps{
			{Type: "live", StatusCode: 200, CreatedAt: time.Now()},
			{Type: "shadow", StatusCode: 200, CreatedAt: time.Now()},
		},
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
	handler := NewCreateRequestHandler(repo)

	gateID := traffictesting.NewGateID()
	cmd := CreateRequestCommand{
		ID:     "invalid-uuid",
		GateID: gateID.String(),
		Method: "GET",
		Path:   "/test",
		Responses: []CreateRequestResponseProps{
			{Type: "live", StatusCode: 200, CreatedAt: time.Now()},
			{Type: "shadow", StatusCode: 200, CreatedAt: time.Now()},
		},
	}

	// Act
	req, err := handler.Handle(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	assert.Nil(t, req)
}

func TestCreateRequestHandler_Handle_invalid_agent_id(t *testing.T) {
	// Arrange
	repo := &mockRequestRepository{}
	handler := NewCreateRequestHandler(repo)

	gateID := traffictesting.NewGateID()
	cmd := CreateRequestCommand{
		GateID:  gateID.String(),
		AgentID: "invalid-format",
		Method:  "GET",
		Path:    "/test",
		Responses: []CreateRequestResponseProps{
			{Type: "live", StatusCode: 200, CreatedAt: time.Now()},
			{Type: "shadow", StatusCode: 200, CreatedAt: time.Now()},
		},
	}

	// Act
	req, err := handler.Handle(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	assert.Nil(t, req)
}

func TestCreateRequestHandler_Handle_invalid_response_type(t *testing.T) {
	// Arrange
	repo := &mockRequestRepository{}
	handler := NewCreateRequestHandler(repo)

	gateID := traffictesting.NewGateID()
	cmd := CreateRequestCommand{
		GateID: gateID.String(),
		Method: "GET",
		Path:   "/test",
		Responses: []CreateRequestResponseProps{
			{Type: "invalid-type", StatusCode: 200, CreatedAt: time.Now()},
			{Type: "shadow", StatusCode: 200, CreatedAt: time.Now()},
		},
	}

	// Act
	req, err := handler.Handle(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	assert.Nil(t, req)
	assert.Contains(t, err.Error(), "invalid response type")
}

func TestCreateRequestHandler_Handle_wrong_number_of_responses(t *testing.T) {
	tests := []struct {
		name      string
		responses []CreateRequestResponseProps
	}{
		{
			name:      "no responses",
			responses: []CreateRequestResponseProps{},
		},
		{
			name: "one response",
			responses: []CreateRequestResponseProps{
				{Type: "live", StatusCode: 200, CreatedAt: time.Now()},
			},
		},
		{
			name: "three responses",
			responses: []CreateRequestResponseProps{
				{Type: "live", StatusCode: 200, CreatedAt: time.Now()},
				{Type: "shadow", StatusCode: 200, CreatedAt: time.Now()},
				{Type: "live", StatusCode: 200, CreatedAt: time.Now()},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo := &mockRequestRepository{}
			handler := NewCreateRequestHandler(repo)

			gateID := traffictesting.NewGateID()
			cmd := CreateRequestCommand{
				GateID:    gateID.String(),
				Method:    "GET",
				Path:      "/test",
				Responses: tt.responses,
			}

			// Act
			req, err := handler.Handle(context.Background(), cmd)

			// Assert
			require.Error(t, err)
			assert.Nil(t, req)
			assert.Contains(t, err.Error(), "exactly two responses are required")
		})
	}
}

func TestCreateRequestHandler_Handle_missing_live_response(t *testing.T) {
	// Arrange
	repo := &mockRequestRepository{}
	handler := NewCreateRequestHandler(repo)

	gateID := traffictesting.NewGateID()
	cmd := CreateRequestCommand{
		GateID: gateID.String(),
		Method: "GET",
		Path:   "/test",
		Responses: []CreateRequestResponseProps{
			{Type: "shadow", StatusCode: 200, CreatedAt: time.Now()},
			{Type: "shadow", StatusCode: 200, CreatedAt: time.Now()},
		},
	}

	// Act
	req, err := handler.Handle(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	assert.Nil(t, req)
	assert.Contains(t, err.Error(), "live response is required")
}

func TestCreateRequestHandler_Handle_missing_shadow_response(t *testing.T) {
	// Arrange
	repo := &mockRequestRepository{}
	handler := NewCreateRequestHandler(repo)

	gateID := traffictesting.NewGateID()
	cmd := CreateRequestCommand{
		GateID: gateID.String(),
		Method: "GET",
		Path:   "/test",
		Responses: []CreateRequestResponseProps{
			{Type: "live", StatusCode: 200, CreatedAt: time.Now()},
			{Type: "live", StatusCode: 200, CreatedAt: time.Now()},
		},
	}

	// Act
	req, err := handler.Handle(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	assert.Nil(t, req)
	assert.Contains(t, err.Error(), "shadow response is required")
}

func TestCreateRequestHandler_Handle_repository_error(t *testing.T) {
	// Arrange
	expectedErr := errors.New("database connection failed")
	repo := &mockRequestRepository{
		saveFn: func(ctx context.Context, req *traffictesting.Request) error {
			return expectedErr
		},
	}
	handler := NewCreateRequestHandler(repo)

	gateID := traffictesting.NewGateID()
	cmd := CreateRequestCommand{
		GateID: gateID.String(),
		Method: "GET",
		Path:   "/test",
		Responses: []CreateRequestResponseProps{
			{Type: "live", StatusCode: 200, CreatedAt: time.Now()},
			{Type: "shadow", StatusCode: 200, CreatedAt: time.Now()},
		},
		Diff: &CreateRequestDiffProps{Content: nil},
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
	responses := []CreateRequestResponseProps{
		{Type: "live", StatusCode: 200, Headers: http.Header{}, Body: b64(`{"ok":true}`)},
		{Type: "shadow", StatusCode: 200, Headers: http.Header{}, Body: b64(`{"ok":true}`)},
	}

	ops, err := computeDiff(responses)

	require.NoError(t, err)
	assert.Empty(t, ops)
}

func TestComputeDiff_different_bodies(t *testing.T) {
	responses := []CreateRequestResponseProps{
		{Type: "live", StatusCode: 200, Headers: http.Header{}, Body: b64(`{"user":"alice"}`)},
		{Type: "shadow", StatusCode: 200, Headers: http.Header{}, Body: b64(`{"user":"bob"}`)},
	}

	ops, err := computeDiff(responses)

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
	responses := []CreateRequestResponseProps{
		{Type: "live", StatusCode: 200, Headers: http.Header{}, Body: b64(`{"ok":true}`)},
		{Type: "shadow", StatusCode: 500, Headers: http.Header{}, Body: b64(`{"ok":false}`)},
	}

	ops, err := computeDiff(responses)

	require.NoError(t, err)
	require.NotEmpty(t, ops)

	paths := map[string]bool{}
	for _, op := range ops {
		paths[op.Path] = true
	}
	assert.True(t, paths["/statusCode"], "expected diff on /statusCode")
}

func TestComputeDiff_different_headers(t *testing.T) {
	responses := []CreateRequestResponseProps{
		{
			Type:       "live",
			StatusCode: 200,
			Headers:    http.Header{"X-Req-Id": []string{"abc"}},
			Body:       b64(`{"ok":true}`),
		},
		{
			Type:       "shadow",
			StatusCode: 200,
			Headers:    http.Header{"X-Req-Id": []string{"def"}},
			Body:       b64(`{"ok":true}`),
		},
	}

	ops, err := computeDiff(responses)

	require.NoError(t, err)
	require.NotEmpty(t, ops, "expected diff on headers")
}

func TestComputeDiff_invalid_base64_body(t *testing.T) {
	responses := []CreateRequestResponseProps{
		{Type: "live", StatusCode: 200, Headers: http.Header{}, Body: []byte("not-valid-base64!!!")},
		{Type: "shadow", StatusCode: 200, Headers: http.Header{}, Body: b64(`{"ok":true}`)},
	}

	_, err := computeDiff(responses)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode live response body")
}

func TestComputeDiff_missing_live_response(t *testing.T) {
	responses := []CreateRequestResponseProps{
		{Type: "shadow", StatusCode: 200, Headers: http.Header{}, Body: b64(`{}`)},
	}

	_, err := computeDiff(responses)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "both live and shadow responses are required")
}

func TestComputeDiff_missing_shadow_response(t *testing.T) {
	responses := []CreateRequestResponseProps{
		{Type: "live", StatusCode: 200, Headers: http.Header{}, Body: b64(`{}`)},
	}

	_, err := computeDiff(responses)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "both live and shadow responses are required")
}