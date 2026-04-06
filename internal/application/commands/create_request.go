package commands

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/pedrobarco/mroki/pkg/diff"
)

// CreateRequestCommand represents the intent to create a new request with responses and diff
type CreateRequestCommand struct {
	ID             string
	GateID         string
	Method         string
	Path           string
	Headers        map[string][]string
	Body           []byte
	CreatedAt      time.Time
	LiveResponse   CreateRequestResponseProps
	ShadowResponse CreateRequestResponseProps
	Diff           *CreateRequestDiffProps // Optional: if nil, diff is computed server-side
}

// CreateRequestResponseProps represents response data in the command
type CreateRequestResponseProps struct {
	ID         string
	StatusCode int
	Headers    http.Header
	Body       []byte
	LatencyMs  int64
	CreatedAt  time.Time
}

// CreateRequestDiffProps represents diff data in the command
type CreateRequestDiffProps struct {
	Content []diff.PatchOp
}

// CreateRequestHandler handles the CreateRequest command
type CreateRequestHandler struct {
	repo traffictesting.RequestRepository
}

// NewCreateRequestHandler creates a new CreateRequestHandler
func NewCreateRequestHandler(repo traffictesting.RequestRepository) *CreateRequestHandler {
	return &CreateRequestHandler{repo: repo}
}

// Handle executes the CreateRequest command
func (h *CreateRequestHandler) Handle(ctx context.Context, cmd CreateRequestCommand) (*traffictesting.Request, error) {
	// Parse gate ID
	gateID, err := traffictesting.ParseGateID(cmd.GateID)
	if err != nil {
		return nil, err
	}

	// Parse request ID (optional)
	var requestID traffictesting.RequestID
	if cmd.ID != "" {
		requestID, err = traffictesting.ParseRequestID(cmd.ID)
		if err != nil {
			return nil, err
		}
	}

	// Parse HTTP method
	method, err := traffictesting.NewHTTPMethod(cmd.Method)
	if err != nil {
		return nil, err
	}

	// Parse path
	path, err := traffictesting.ParsePath(cmd.Path)
	if err != nil {
		return nil, err
	}

	// Parse and create live response
	live, err := buildResponse(cmd.LiveResponse)
	if err != nil {
		return nil, fmt.Errorf("live response: %w", err)
	}

	// Parse and create shadow response
	shadow, err := buildResponse(cmd.ShadowResponse)
	if err != nil {
		return nil, fmt.Errorf("shadow response: %w", err)
	}

	// Compute or use provided diff content
	var diffContent []diff.PatchOp
	if cmd.Diff != nil {
		diffContent = cmd.Diff.Content
	} else {
		computed, err := computeDiff(cmd.LiveResponse, cmd.ShadowResponse)
		if err != nil {
			// Diff computation errors are non-fatal — store empty diff
			diffContent = []diff.PatchOp{}
		} else {
			diffContent = computed
		}
	}

	d, err := traffictesting.NewDiff(live.ID, shadow.ID, diffContent)
	if err != nil {
		return nil, fmt.Errorf("failed to create diff: %w", err)
	}

	// Create request aggregate with value objects
	request, err := traffictesting.NewRequest(
		gateID,
		method,
		path,
		traffictesting.NewHeaders(cmd.Headers),
		cmd.Body,
		cmd.CreatedAt,
		*live,
		*shadow,
		*d,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Apply optional fields if provided
	if !requestID.IsZero() {
		request.ID = requestID
	}

	// Persist (transaction boundary)
	if err := h.repo.Save(ctx, request); err != nil {
		return nil, fmt.Errorf("failed to save request: %w", err)
	}

	return request, nil
}


// buildResponse creates a domain Response from command props.
func buildResponse(props CreateRequestResponseProps) (*traffictesting.Response, error) {
	statusCode, err := traffictesting.ParseStatusCode(props.StatusCode)
	if err != nil {
		return nil, err
	}

	opts := []traffictesting.ResponseOption{}
	if props.ID != "" {
		id, err := uuid.Parse(props.ID)
		if err != nil {
			return nil, fmt.Errorf("invalid response ID: %w", err)
		}
		opts = append(opts, traffictesting.WithResponseID(id))
	}

	return traffictesting.NewResponse(
		statusCode,
		traffictesting.NewHeaders(props.Headers),
		props.Body,
		props.LatencyMs,
		props.CreatedAt,
		opts...,
	)
}

// computeDiff computes a JSON diff between live and shadow response bodies.
// Response bodies are expected to be base64-encoded JSON strings.
// Returns RFC 6902 JSON Patch operations describing the differences.
func computeDiff(live, shadow CreateRequestResponseProps) ([]diff.PatchOp, error) {
	// Decode base64 bodies
	liveBody, err := base64.StdEncoding.DecodeString(string(live.Body))
	if err != nil {
		return nil, fmt.Errorf("failed to decode live response body: %w", err)
	}

	shadowBody, err := base64.StdEncoding.DecodeString(string(shadow.Body))
	if err != nil {
		return nil, fmt.Errorf("failed to decode shadow response body: %w", err)
	}

	// Marshal headers to JSON (same format as proxy-side diffing)
	liveHeaders, err := json.Marshal(live.Headers)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal live headers: %w", err)
	}

	shadowHeaders, err := json.Marshal(shadow.Headers)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal shadow headers: %w", err)
	}

	// Construct synthetic JSON documents matching proxy-side format:
	// {"statusCode": N, "headers": {...}, "body": ...}
	liveJSON := fmt.Sprintf(`{"statusCode": %d, "headers": %s, "body": %s}`,
		live.StatusCode, liveHeaders, liveBody)
	shadowJSON := fmt.Sprintf(`{"statusCode": %d, "headers": %s, "body": %s}`,
		shadow.StatusCode, shadowHeaders, shadowBody)

	return diff.JSON(liveJSON, shadowJSON)
}