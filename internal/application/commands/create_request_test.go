package commands

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/pedrobarco/mroki/internal/domain/pagination"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/pedrobarco/mroki/pkg/diff"
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
		GateID:   gateID.String(),
		Method:   "GET",
		Path:     "/api/test",
		RawQuery: "page=1&limit=20",
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
	assert.Equal(t, "page=1&limit=20", req.RawQuery)
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

// --- BuildEnvelope + diff.Parsed tests ---

func b64(s string) []byte {
	return []byte(base64.StdEncoding.EncodeToString([]byte(s)))
}

// parseBody is a test helper that unmarshals JSON into a Go value tree.
func parseBody(t *testing.T, s string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatalf("parseBody: %v", err)
	}
	return v
}

// ---------------------------------------------------------------------------
// Unit tests for bodyToRawMessage and rawBytesToJSONString
// ---------------------------------------------------------------------------

func TestBodyToRawMessage_json_body(t *testing.T) {
	parsed := map[string]any{"user": "Alice", "age": float64(30)}
	raw := []byte(`{"user":"Alice","age":30}`)

	result, err := bodyToRawMessage(raw, parsed)

	require.NoError(t, err)
	require.NotNil(t, result)
	// Should be valid JSON that matches the parsed tree
	assert.JSONEq(t, `{"user":"Alice","age":30}`, string(result))
}

func TestBodyToRawMessage_empty_body(t *testing.T) {
	result, err := bodyToRawMessage(nil, nil)
	require.NoError(t, err)
	assert.Nil(t, result)

	result, err = bodyToRawMessage([]byte{}, nil)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestBodyToRawMessage_non_json_body(t *testing.T) {
	tests := []struct {
		name string
		raw  []byte
	}{
		{
			name: "plain text",
			raw:  []byte("hello world"),
		},
		{
			name: "HTML",
			raw:  []byte("<html><body><h1>Hello</h1></body></html>"),
		},
		{
			name: "XML",
			raw:  []byte(`<?xml version="1.0"?><root><item>test</item></root>`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// bodyParsed is nil for non-JSON content
			result, err := bodyToRawMessage(tt.raw, nil)

			require.NoError(t, err)
			require.NotNil(t, result)
			// Must be valid JSON (a JSON string value)
			assert.True(t, json.Valid(result), "result should be valid JSON")
			// Unmarshaling should recover the original string
			var stored string
			require.NoError(t, json.Unmarshal(result, &stored))
			assert.Equal(t, string(tt.raw), stored)
		})
	}
}

func TestRawBytesToJSONString(t *testing.T) {
	tests := []struct {
		name     string
		raw      []byte
		expected string
	}{
		{
			name:     "plain text",
			raw:      []byte("hello"),
			expected: `"hello"`,
		},
		{
			name:     "text with quotes",
			raw:      []byte(`say "hi"`),
			expected: `"say \"hi\""`,
		},
		{
			name:     "text with newlines",
			raw:      []byte("line1\nline2"),
			expected: `"line1\nline2"`,
		},
		{
			name:     "binary-like content",
			raw:      []byte{0x00, 0x01, 0x02, 0xFF},
			expected: `"\u0000\u0001\u0002\u00ff"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := rawBytesToJSONString(tt.raw)

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.True(t, json.Valid(result), "result should be valid JSON")
		})
	}
}

// ---------------------------------------------------------------------------
// Handler-level tests for non-JSON body types
// ---------------------------------------------------------------------------

func TestCreateRequestHandler_Handle_html_body(t *testing.T) {
	var savedReq *traffictesting.Request
	repo := &mockRequestRepository{
		saveFn: func(ctx context.Context, req *traffictesting.Request) error {
			savedReq = req
			return nil
		},
	}
	handler := NewCreateRequestHandler(repo, &mockGateRepoForRequest{})

	htmlBody := "<html><body><h1>Hello</h1></body></html>"
	cmd := CreateRequestCommand{
		GateID:    traffictesting.NewGateID().String(),
		Method:    "GET",
		Path:      "/page",
		Headers:   map[string][]string{"Content-Type": {"text/html"}},
		Body:      b64(htmlBody),
		CreatedAt: time.Now(),
		LiveResponse: CreateRequestResponseProps{
			StatusCode: 200,
			Headers:    http.Header{"Content-Type": []string{"text/html"}},
			Body:       b64(htmlBody),
			CreatedAt:  time.Now(),
		},
		ShadowResponse: CreateRequestResponseProps{
			StatusCode: 200,
			Headers:    http.Header{"Content-Type": []string{"text/html"}},
			Body:       b64(htmlBody),
			CreatedAt:  time.Now(),
		},
		Diff: &CreateRequestDiffProps{Content: nil},
	}

	req, err := handler.Handle(context.Background(), cmd)

	require.NoError(t, err)
	require.NotNil(t, req)
	require.NotNil(t, savedReq)

	// Non-JSON body should be stored as a JSON string value
	assert.True(t, json.Valid(savedReq.Body), "request body should be valid JSON")
	assert.True(t, json.Valid(savedReq.LiveResponse.Body), "live body should be valid JSON")

	// Unwrap the JSON string to get the original HTML back
	var stored string
	require.NoError(t, json.Unmarshal(savedReq.LiveResponse.Body, &stored))
	assert.Equal(t, htmlBody, stored)
}

func TestCreateRequestHandler_Handle_plain_text_body(t *testing.T) {
	var savedReq *traffictesting.Request
	repo := &mockRequestRepository{
		saveFn: func(ctx context.Context, req *traffictesting.Request) error {
			savedReq = req
			return nil
		},
	}
	handler := NewCreateRequestHandler(repo, &mockGateRepoForRequest{})

	textBody := "Just some plain text response"
	cmd := CreateRequestCommand{
		GateID:    traffictesting.NewGateID().String(),
		Method:    "GET",
		Path:      "/text",
		Headers:   map[string][]string{"Content-Type": {"text/plain"}},
		Body:      b64(textBody),
		CreatedAt: time.Now(),
		LiveResponse: CreateRequestResponseProps{
			StatusCode: 200,
			Headers:    http.Header{"Content-Type": []string{"text/plain"}},
			Body:       b64(textBody),
			CreatedAt:  time.Now(),
		},
		ShadowResponse: CreateRequestResponseProps{
			StatusCode: 200,
			Headers:    http.Header{"Content-Type": []string{"text/plain"}},
			Body:       b64(textBody),
			CreatedAt:  time.Now(),
		},
		Diff: &CreateRequestDiffProps{Content: nil},
	}

	req, err := handler.Handle(context.Background(), cmd)

	require.NoError(t, err)
	require.NotNil(t, req)
	require.NotNil(t, savedReq)

	assert.True(t, json.Valid(savedReq.LiveResponse.Body), "body should be valid JSON")

	var stored string
	require.NoError(t, json.Unmarshal(savedReq.LiveResponse.Body, &stored))
	assert.Equal(t, textBody, stored)
}

func TestCreateRequestHandler_Handle_empty_body(t *testing.T) {
	var savedReq *traffictesting.Request
	repo := &mockRequestRepository{
		saveFn: func(ctx context.Context, req *traffictesting.Request) error {
			savedReq = req
			return nil
		},
	}
	handler := NewCreateRequestHandler(repo, &mockGateRepoForRequest{})

	cmd := CreateRequestCommand{
		GateID:    traffictesting.NewGateID().String(),
		Method:    "GET",
		Path:      "/empty",
		Headers:   map[string][]string{},
		Body:      []byte{},
		CreatedAt: time.Now(),
		LiveResponse: CreateRequestResponseProps{
			StatusCode: 204,
			Headers:    http.Header{},
			Body:       []byte{},
			CreatedAt:  time.Now(),
		},
		ShadowResponse: CreateRequestResponseProps{
			StatusCode: 204,
			Headers:    http.Header{},
			Body:       []byte{},
			CreatedAt:  time.Now(),
		},
		Diff: &CreateRequestDiffProps{Content: nil},
	}

	req, err := handler.Handle(context.Background(), cmd)

	require.NoError(t, err)
	require.NotNil(t, req)
	require.NotNil(t, savedReq)

	// Empty bodies should be nil (→ NULL in DB)
	assert.Nil(t, savedReq.Body)
	assert.Nil(t, savedReq.LiveResponse.Body)
	assert.Nil(t, savedReq.ShadowResponse.Body)
}

func TestCreateRequestHandler_Handle_invalid_base64_body(t *testing.T) {
	var savedReq *traffictesting.Request
	repo := &mockRequestRepository{
		saveFn: func(ctx context.Context, req *traffictesting.Request) error {
			savedReq = req
			return nil
		},
	}
	handler := NewCreateRequestHandler(repo, &mockGateRepoForRequest{})

	// Not valid base64 — the handler should fallback to wrapping as JSON string
	invalidB64 := []byte("this-is-not-valid-base64!!!")
	cmd := CreateRequestCommand{
		GateID:    traffictesting.NewGateID().String(),
		Method:    "POST",
		Path:      "/bad-encoding",
		Headers:   map[string][]string{},
		Body:      invalidB64,
		CreatedAt: time.Now(),
		LiveResponse: CreateRequestResponseProps{
			StatusCode: 200,
			Headers:    http.Header{},
			Body:       invalidB64,
			CreatedAt:  time.Now(),
		},
		ShadowResponse: CreateRequestResponseProps{
			StatusCode: 200,
			Headers:    http.Header{},
			Body:       invalidB64,
			CreatedAt:  time.Now(),
		},
		Diff: &CreateRequestDiffProps{Content: nil},
	}

	req, err := handler.Handle(context.Background(), cmd)

	require.NoError(t, err)
	require.NotNil(t, req)
	require.NotNil(t, savedReq)

	// Body should be stored as a JSON string wrapping the raw input
	assert.True(t, json.Valid(savedReq.LiveResponse.Body), "body should be valid JSON")
	assert.NotNil(t, savedReq.LiveResponse.Body)

	var stored string
	require.NoError(t, json.Unmarshal(savedReq.LiveResponse.Body, &stored))
	assert.Equal(t, string(invalidB64), stored)
}

func TestBuildEnvelopeParsed_identical_responses(t *testing.T) {
	body := parseBody(t, `{"ok":true}`)

	liveEnvelope := diff.BuildEnvelope(200, http.Header{}, body)
	shadowEnvelope := diff.BuildEnvelope(200, http.Header{}, body)
	ops, err := diff.Parsed(liveEnvelope, shadowEnvelope)

	require.NoError(t, err)
	assert.Empty(t, ops)
}

func TestBuildEnvelopeParsed_different_bodies(t *testing.T) {
	liveBody := parseBody(t, `{"user":"alice"}`)
	shadowBody := parseBody(t, `{"user":"bob"}`)

	liveEnvelope := diff.BuildEnvelope(200, http.Header{}, liveBody)
	shadowEnvelope := diff.BuildEnvelope(200, http.Header{}, shadowBody)
	ops, err := diff.Parsed(liveEnvelope, shadowEnvelope)

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

func TestBuildEnvelopeParsed_different_status_codes(t *testing.T) {
	liveBody := parseBody(t, `{"ok":true}`)
	shadowBody := parseBody(t, `{"ok":false}`)

	liveEnvelope := diff.BuildEnvelope(200, http.Header{}, liveBody)
	shadowEnvelope := diff.BuildEnvelope(500, http.Header{}, shadowBody)
	ops, err := diff.Parsed(liveEnvelope, shadowEnvelope)

	require.NoError(t, err)
	require.NotEmpty(t, ops)

	paths := map[string]bool{}
	for _, op := range ops {
		paths[op.Path] = true
	}
	assert.True(t, paths["/statusCode"], "expected diff on /statusCode")
}

func TestBuildEnvelopeParsed_different_headers(t *testing.T) {
	body := parseBody(t, `{"ok":true}`)

	liveEnvelope := diff.BuildEnvelope(200, http.Header{"X-Req-Id": []string{"abc"}}, body)
	shadowEnvelope := diff.BuildEnvelope(200, http.Header{"X-Req-Id": []string{"def"}}, body)
	ops, err := diff.Parsed(liveEnvelope, shadowEnvelope)

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

	// Stored body is now json.RawMessage (no base64 decoding needed)
	liveBody := string(savedReq.LiveResponse.Body)
	assert.Contains(t, liveBody, `"[REDACTED]"`)
	assert.NotContains(t, liveBody, "s3cret")
	assert.NotContains(t, liveBody, "123-45-6789")
	assert.Contains(t, liveBody, "Alice") // non-redacted field preserved

	// Stored request body is now json.RawMessage
	reqBody := string(savedReq.Body)
	assert.Contains(t, reqBody, `"[REDACTED]"`)
	assert.NotContains(t, reqBody, "req-s3cret")
	assert.NotContains(t, reqBody, "111-22-3333")
	assert.Contains(t, reqBody, "Bob") // non-redacted field preserved
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

	// Body field redacted (stored as json.RawMessage, no base64)
	liveBody := string(savedReq.LiveResponse.Body)
	assert.NotContains(t, liveBody, "abc123")
	assert.Contains(t, liveBody, "ok") // non-redacted field preserved
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

	// Body should be unchanged (stored as json.RawMessage, no base64)
	assert.JSONEq(t, `{"kept":"yes"}`, string(savedReq.LiveResponse.Body))
}

// fakeDispatcher captures dispatched domain events for assertions.
type fakeDispatcher struct {
	events []traffictesting.DomainEvent
}

func (f *fakeDispatcher) Dispatch(_ context.Context, events ...traffictesting.DomainEvent) {
	f.events = append(f.events, events...)
}

func TestCreateRequestHandler_Handle_DispatchesRequestCompared(t *testing.T) {
	repo := &mockRequestRepository{
		saveFn: func(ctx context.Context, req *traffictesting.Request) error { return nil },
	}
	dispatcher := &fakeDispatcher{}
	handler := NewCreateRequestHandler(repo, &mockGateRepoForRequest{}, WithEventDispatcher(dispatcher))

	gateID := traffictesting.NewGateID()
	cmd := CreateRequestCommand{
		GateID:    gateID.String(),
		Method:    "GET",
		Path:      "/api/test",
		CreatedAt: time.Now(),
		LiveResponse: CreateRequestResponseProps{
			StatusCode: 200, Headers: http.Header{}, Body: nil, LatencyMs: 12, CreatedAt: time.Now(),
		},
		ShadowResponse: CreateRequestResponseProps{
			StatusCode: 500, Headers: http.Header{}, Body: nil, LatencyMs: 34, CreatedAt: time.Now(),
		},
		Diff: &CreateRequestDiffProps{
			Content: []diff.PatchOp{{Op: "replace", Path: "/status", Value: "error"}},
		},
	}

	_, err := handler.Handle(context.Background(), cmd)
	require.NoError(t, err)

	require.Len(t, dispatcher.events, 1, "one RequestCompared event dispatched after save")
	evt, ok := dispatcher.events[0].(traffictesting.RequestCompared)
	require.True(t, ok, "expected a RequestCompared event")
	assert.Equal(t, gateID.String(), evt.GateID().String())
	assert.True(t, evt.HasDiff())
	assert.Equal(t, 200, evt.LiveStatus())
	assert.Equal(t, 500, evt.ShadowStatus())
}

func TestCreateRequestHandler_Handle_NoDispatchOnSaveError(t *testing.T) {
	repo := &mockRequestRepository{
		saveFn: func(ctx context.Context, req *traffictesting.Request) error {
			return errors.New("save failed")
		},
	}
	dispatcher := &fakeDispatcher{}
	handler := NewCreateRequestHandler(repo, &mockGateRepoForRequest{}, WithEventDispatcher(dispatcher))

	gateID := traffictesting.NewGateID()
	cmd := CreateRequestCommand{
		GateID:         gateID.String(),
		Method:         "GET",
		Path:           "/api/test",
		CreatedAt:      time.Now(),
		LiveResponse:   CreateRequestResponseProps{StatusCode: 200, Headers: http.Header{}, CreatedAt: time.Now()},
		ShadowResponse: CreateRequestResponseProps{StatusCode: 200, Headers: http.Header{}, CreatedAt: time.Now()},
		Diff:           &CreateRequestDiffProps{Content: nil},
	}

	_, err := handler.Handle(context.Background(), cmd)
	require.Error(t, err)
	assert.Empty(t, dispatcher.events, "no event dispatched when save fails")
}
