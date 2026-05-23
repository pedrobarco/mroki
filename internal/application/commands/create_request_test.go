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
type mockGateRepoForRequest struct {
	getByIDFn func(ctx context.Context, id traffictesting.GateID) (*traffictesting.Gate, error)
}

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
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
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

// --- computeDiffFromDecoded tests ---

func b64(s string) []byte {
	return []byte(base64.StdEncoding.EncodeToString([]byte(s)))
}

func TestComputeDiffFromDecoded_identical_responses(t *testing.T) {
	body := []byte(`{"ok":true}`)
	live := CreateRequestResponseProps{StatusCode: 200, Headers: http.Header{}, Body: b64(`{"ok":true}`)}
	shadow := CreateRequestResponseProps{StatusCode: 200, Headers: http.Header{}, Body: b64(`{"ok":true}`)}

	ops, err := computeDiffFromDecoded(live, body, shadow, body)

	require.NoError(t, err)
	assert.Empty(t, ops)
}

func TestComputeDiffFromDecoded_different_bodies(t *testing.T) {
	live := CreateRequestResponseProps{StatusCode: 200, Headers: http.Header{}, Body: b64(`{"user":"alice"}`)}
	shadow := CreateRequestResponseProps{StatusCode: 200, Headers: http.Header{}, Body: b64(`{"user":"bob"}`)}

	ops, err := computeDiffFromDecoded(live, []byte(`{"user":"alice"}`), shadow, []byte(`{"user":"bob"}`))

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

func TestComputeDiffFromDecoded_different_status_codes(t *testing.T) {
	live := CreateRequestResponseProps{StatusCode: 200, Headers: http.Header{}, Body: b64(`{"ok":true}`)}
	shadow := CreateRequestResponseProps{StatusCode: 500, Headers: http.Header{}, Body: b64(`{"ok":false}`)}

	ops, err := computeDiffFromDecoded(live, []byte(`{"ok":true}`), shadow, []byte(`{"ok":false}`))

	require.NoError(t, err)
	require.NotEmpty(t, ops)

	paths := map[string]bool{}
	for _, op := range ops {
		paths[op.Path] = true
	}
	assert.True(t, paths["/statusCode"], "expected diff on /statusCode")
}

func TestComputeDiffFromDecoded_different_headers(t *testing.T) {
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

	ops, err := computeDiffFromDecoded(live, []byte(`{"ok":true}`), shadow, []byte(`{"ok":true}`))

	require.NoError(t, err)
	require.NotEmpty(t, ops, "expected diff on headers")
}

// --- Redaction integration tests ---

func TestCreateRequestHandler_Handle_redacts_default_headers(t *testing.T) {
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
			"Authorization": {"Bearer super-secret"},
			"Content-Type":  {"application/json"},
		},
		Body:      []byte(`{}`),
		CreatedAt: time.Now(),
		LiveResponse: CreateRequestResponseProps{
			StatusCode: 200,
			Headers: http.Header{
				"Set-Cookie": {"session=abc123"},
				"X-Req-Id":   {"req-1"},
			},
			Body:      []byte(`{}`),
			CreatedAt: time.Now(),
		},
		ShadowResponse: CreateRequestResponseProps{
			StatusCode: 200,
			Headers: http.Header{
				"Cookie": {"session=abc123"},
				"X-Req-Id": {"req-1"},
			},
			Body:      []byte(`{}`),
			CreatedAt: time.Now(),
		},
		Diff: &CreateRequestDiffProps{Content: nil},
	}

	req, err := handler.Handle(context.Background(), cmd)

	require.NoError(t, err)
	require.NotNil(t, req)
	require.NotNil(t, savedReq)

	// Request headers: Authorization should be redacted, Content-Type preserved
	assert.Equal(t, traffictesting.RedactedValue, savedReq.Headers.HTTPHeader().Get("Authorization"))
	assert.Equal(t, "application/json", savedReq.Headers.HTTPHeader().Get("Content-Type"))

	// Live response: Set-Cookie redacted, X-Req-Id preserved
	assert.Equal(t, traffictesting.RedactedValue, savedReq.LiveResponse.Headers.HTTPHeader().Get("Set-Cookie"))
	assert.Equal(t, "req-1", savedReq.LiveResponse.Headers.HTTPHeader().Get("X-Req-Id"))

	// Shadow response: Cookie redacted, X-Req-Id preserved
	assert.Equal(t, traffictesting.RedactedValue, savedReq.ShadowResponse.Headers.HTTPHeader().Get("Cookie"))
	assert.Equal(t, "req-1", savedReq.ShadowResponse.Headers.HTTPHeader().Get("X-Req-Id"))
}

func TestCreateRequestHandler_Handle_redacts_additional_gate_fields(t *testing.T) {
	var savedReq *traffictesting.Request
	repo := &mockRequestRepository{
		saveFn: func(ctx context.Context, req *traffictesting.Request) error {
			savedReq = req
			return nil
		},
	}

	redactedFields, _ := traffictesting.NewRedactedFields([]string{"headers.X-Internal-Token"})
	gateRepo := &mockGateRepoForRequest{
		getByIDFn: func(ctx context.Context, id traffictesting.GateID) (*traffictesting.Gate, error) {
			name, _ := traffictesting.ParseGateName("test")
			live, _ := traffictesting.ParseGateURL("http://live.example.com")
			shadow, _ := traffictesting.ParseGateURL("http://shadow.example.com")
			gate, _ := traffictesting.NewGate(name, live, shadow,
				traffictesting.WithGateID(id),
				traffictesting.WithGateRedactedFields(redactedFields),
			)
			return gate, nil
		},
	}
	handler := NewCreateRequestHandler(repo, gateRepo)

	gateID := traffictesting.NewGateID()
	cmd := CreateRequestCommand{
		GateID: gateID.String(),
		Method: "GET",
		Path:   "/api/test",
		Headers: map[string][]string{
			"X-Internal-Token": {"secret-internal"},
			"Content-Type":     {"application/json"},
		},
		Body:      []byte(`{}`),
		CreatedAt: time.Now(),
		LiveResponse: CreateRequestResponseProps{
			StatusCode: 200,
			Headers:    http.Header{},
			Body:       []byte(`{}`),
			CreatedAt:  time.Now(),
		},
		ShadowResponse: CreateRequestResponseProps{
			StatusCode: 200,
			Headers:    http.Header{},
			Body:       []byte(`{}`),
			CreatedAt:  time.Now(),
		},
		Diff: &CreateRequestDiffProps{Content: nil},
	}

	req, err := handler.Handle(context.Background(), cmd)

	require.NoError(t, err)
	require.NotNil(t, req)
	require.NotNil(t, savedReq)

	// Custom gate field should be redacted
	assert.Equal(t, traffictesting.RedactedValue, savedReq.Headers.HTTPHeader().Get("X-Internal-Token"))
	assert.Equal(t, "application/json", savedReq.Headers.HTTPHeader().Get("Content-Type"))
}

func TestCreateRequestHandler_Handle_gate_not_found_uses_defaults(t *testing.T) {
	var savedReq *traffictesting.Request
	repo := &mockRequestRepository{
		saveFn: func(ctx context.Context, req *traffictesting.Request) error {
			savedReq = req
			return nil
		},
	}

	gateRepo := &mockGateRepoForRequest{
		getByIDFn: func(ctx context.Context, id traffictesting.GateID) (*traffictesting.Gate, error) {
			return nil, traffictesting.ErrGateNotFound
		},
	}
	handler := NewCreateRequestHandler(repo, gateRepo)

	gateID := traffictesting.NewGateID()
	cmd := CreateRequestCommand{
		GateID: gateID.String(),
		Method: "GET",
		Path:   "/api/test",
		Headers: map[string][]string{
			"Authorization": {"Bearer secret"},
			"Content-Type":  {"text/plain"},
		},
		Body:      []byte(`{}`),
		CreatedAt: time.Now(),
		LiveResponse: CreateRequestResponseProps{
			StatusCode: 200,
			Headers:    http.Header{},
			Body:       []byte(`{}`),
			CreatedAt:  time.Now(),
		},
		ShadowResponse: CreateRequestResponseProps{
			StatusCode: 200,
			Headers:    http.Header{},
			Body:       []byte(`{}`),
			CreatedAt:  time.Now(),
		},
		Diff: &CreateRequestDiffProps{Content: nil},
	}

	req, err := handler.Handle(context.Background(), cmd)

	require.NoError(t, err)
	require.NotNil(t, req)
	require.NotNil(t, savedReq)

	// Default redaction should still apply
	assert.Equal(t, traffictesting.RedactedValue, savedReq.Headers.HTTPHeader().Get("Authorization"))
	assert.Equal(t, "text/plain", savedReq.Headers.HTTPHeader().Get("Content-Type"))
}

func TestCreateRequestHandler_Handle_gate_fetch_infra_error_returns_error(t *testing.T) {
	repo := &mockRequestRepository{}
	infraErr := errors.New("database connection failed")
	gateRepo := &mockGateRepoForRequest{
		getByIDFn: func(ctx context.Context, id traffictesting.GateID) (*traffictesting.Gate, error) {
			return nil, infraErr
		},
	}
	handler := NewCreateRequestHandler(repo, gateRepo)

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
			Body:       []byte(`{}`),
			CreatedAt:  time.Now(),
		},
		ShadowResponse: CreateRequestResponseProps{
			StatusCode: 200,
			Headers:    http.Header{},
			Body:       []byte(`{}`),
			CreatedAt:  time.Now(),
		},
		Diff: &CreateRequestDiffProps{Content: nil},
	}

	req, err := handler.Handle(context.Background(), cmd)

	require.Error(t, err)
	assert.Nil(t, req)
	assert.Contains(t, err.Error(), "failed to fetch gate")
	assert.ErrorIs(t, err, infraErr)
}

func TestCreateRequestHandler_Handle_redacts_body_fields(t *testing.T) {
	var savedReq *traffictesting.Request
	repo := &mockRequestRepository{
		saveFn: func(ctx context.Context, req *traffictesting.Request) error {
			savedReq = req
			return nil
		},
	}

	redactedFields, _ := traffictesting.NewRedactedFields([]string{"body.password", "body.user.ssn"})
	gateRepo := &mockGateRepoForRequest{
		getByIDFn: func(ctx context.Context, id traffictesting.GateID) (*traffictesting.Gate, error) {
			name, _ := traffictesting.ParseGateName("test")
			live, _ := traffictesting.ParseGateURL("http://live.example.com")
			shadow, _ := traffictesting.ParseGateURL("http://shadow.example.com")
			gate, _ := traffictesting.NewGate(name, live, shadow,
				traffictesting.WithGateID(id),
				traffictesting.WithGateRedactedFields(redactedFields),
			)
			return gate, nil
		},
	}
	handler := NewCreateRequestHandler(repo, gateRepo)

	gateID := traffictesting.NewGateID()
	cmd := CreateRequestCommand{
		GateID:    gateID.String(),
		Method:    "POST",
		Path:      "/api/login",
		Headers:   map[string][]string{"Content-Type": {"application/json"}},
		Body:      b64(`{"password":"req-s3cret","user":{"ssn":"111-22-3333","name":"Bob"}}`),
		CreatedAt: time.Now(),
		LiveResponse: CreateRequestResponseProps{
			StatusCode: 200,
			Headers:    http.Header{},
			Body:       b64(`{"password":"s3cret","user":{"ssn":"123-45-6789","name":"Alice"}}`),
			CreatedAt:  time.Now(),
		},
		ShadowResponse: CreateRequestResponseProps{
			StatusCode: 200,
			Headers:    http.Header{},
			Body:       b64(`{"password":"s3cret","user":{"ssn":"123-45-6789","name":"Alice"}}`),
			CreatedAt:  time.Now(),
		},
		Diff: &CreateRequestDiffProps{Content: nil},
	}

	req, err := handler.Handle(context.Background(), cmd)

	require.NoError(t, err)
	require.NotNil(t, req)
	require.NotNil(t, savedReq)

	// Decode stored response body to verify redaction
	liveBody, _ := base64.StdEncoding.DecodeString(string(savedReq.LiveResponse.Body))
	assert.Contains(t, string(liveBody), `"[REDACTED]"`)
	assert.NotContains(t, string(liveBody), "s3cret")
	assert.NotContains(t, string(liveBody), "123-45-6789")
	assert.Contains(t, string(liveBody), "Alice") // non-redacted field preserved

	// Decode stored request body to verify redaction
	reqBody, _ := base64.StdEncoding.DecodeString(string(savedReq.Body))
	assert.Contains(t, string(reqBody), `"[REDACTED]"`)
	assert.NotContains(t, string(reqBody), "req-s3cret")
	assert.NotContains(t, string(reqBody), "111-22-3333")
	assert.Contains(t, string(reqBody), "Bob") // non-redacted field preserved
}

func TestCreateRequestHandler_Handle_redacts_mixed_header_and_body(t *testing.T) {
	var savedReq *traffictesting.Request
	repo := &mockRequestRepository{
		saveFn: func(ctx context.Context, req *traffictesting.Request) error {
			savedReq = req
			return nil
		},
	}

	redactedFields, _ := traffictesting.NewRedactedFields([]string{"headers.X-Secret", "body.token"})
	gateRepo := &mockGateRepoForRequest{
		getByIDFn: func(ctx context.Context, id traffictesting.GateID) (*traffictesting.Gate, error) {
			name, _ := traffictesting.ParseGateName("test")
			live, _ := traffictesting.ParseGateURL("http://live.example.com")
			shadow, _ := traffictesting.ParseGateURL("http://shadow.example.com")
			gate, _ := traffictesting.NewGate(name, live, shadow,
				traffictesting.WithGateID(id),
				traffictesting.WithGateRedactedFields(redactedFields),
			)
			return gate, nil
		},
	}
	handler := NewCreateRequestHandler(repo, gateRepo)

	gateID := traffictesting.NewGateID()
	cmd := CreateRequestCommand{
		GateID:  gateID.String(),
		Method:  "GET",
		Path:    "/api/test",
		Headers: map[string][]string{"X-Secret": {"hidden"}, "Content-Type": {"application/json"}},
		Body:    []byte(`{}`),
		CreatedAt: time.Now(),
		LiveResponse: CreateRequestResponseProps{
			StatusCode: 200,
			Headers:    http.Header{"X-Secret": {"hidden"}},
			Body:       b64(`{"token":"abc123","data":"ok"}`),
			CreatedAt:  time.Now(),
		},
		ShadowResponse: CreateRequestResponseProps{
			StatusCode: 200,
			Headers:    http.Header{},
			Body:       b64(`{"token":"abc123","data":"ok"}`),
			CreatedAt:  time.Now(),
		},
		Diff: &CreateRequestDiffProps{Content: nil},
	}

	req, err := handler.Handle(context.Background(), cmd)

	require.NoError(t, err)
	require.NotNil(t, req)
	require.NotNil(t, savedReq)

	// Header redacted
	assert.Equal(t, traffictesting.RedactedValue, savedReq.Headers.HTTPHeader().Get("X-Secret"))
	assert.Equal(t, traffictesting.RedactedValue, savedReq.LiveResponse.Headers.HTTPHeader().Get("X-Secret"))

	// Body field redacted
	liveBody, _ := base64.StdEncoding.DecodeString(string(savedReq.LiveResponse.Body))
	assert.NotContains(t, string(liveBody), "abc123")
	assert.Contains(t, string(liveBody), "ok") // non-redacted field preserved
}

func TestCreateRequestHandler_Handle_missing_body_path_silently_skipped(t *testing.T) {
	var savedReq *traffictesting.Request
	repo := &mockRequestRepository{
		saveFn: func(ctx context.Context, req *traffictesting.Request) error {
			savedReq = req
			return nil
		},
	}

	redactedFields, _ := traffictesting.NewRedactedFields([]string{"body.nonexistent.deep.path"})
	gateRepo := &mockGateRepoForRequest{
		getByIDFn: func(ctx context.Context, id traffictesting.GateID) (*traffictesting.Gate, error) {
			name, _ := traffictesting.ParseGateName("test")
			live, _ := traffictesting.ParseGateURL("http://live.example.com")
			shadow, _ := traffictesting.ParseGateURL("http://shadow.example.com")
			gate, _ := traffictesting.NewGate(name, live, shadow,
				traffictesting.WithGateID(id),
				traffictesting.WithGateRedactedFields(redactedFields),
			)
			return gate, nil
		},
	}
	handler := NewCreateRequestHandler(repo, gateRepo)

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
			Body:       b64(`{"kept":"yes"}`),
			CreatedAt:  time.Now(),
		},
		ShadowResponse: CreateRequestResponseProps{
			StatusCode: 200,
			Headers:    http.Header{},
			Body:       b64(`{"kept":"yes"}`),
			CreatedAt:  time.Now(),
		},
		Diff: &CreateRequestDiffProps{Content: nil},
	}

	req, err := handler.Handle(context.Background(), cmd)

	require.NoError(t, err)
	require.NotNil(t, req)

	// Body should be unchanged
	liveBody, _ := base64.StdEncoding.DecodeString(string(savedReq.LiveResponse.Body))
	assert.JSONEq(t, `{"kept":"yes"}`, string(liveBody))
}
