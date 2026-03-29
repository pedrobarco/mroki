package commands

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/pedrobarco/mroki/pkg/diff"
)

// CreateRequestCommand represents the intent to create a new request with responses and diff
type CreateRequestCommand struct {
	ID        string
	GateID    string
	Method    string
	Path      string
	Headers   map[string][]string
	Body      []byte
	CreatedAt time.Time
	Responses []CreateRequestResponseProps
	Diff      *CreateRequestDiffProps // Optional: if nil, diff is computed server-side
}

// CreateRequestResponseProps represents response data in the command
type CreateRequestResponseProps struct {
	ID         string
	Type       string
	StatusCode int
	Headers    http.Header
	Body       []byte
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

	// Parse and create responses
	var live, shadow *traffictesting.Response
	var responses []traffictesting.Response
	for i, dto := range cmd.Responses {
		// Parse response type
		rtype, err := traffictesting.NewResponseType(dto.Type)
		if err != nil {
			return nil, fmt.Errorf("response %d: invalid response type: %w", i, err)
		}

		// Parse status code
		statusCode, err := traffictesting.ParseStatusCode(dto.StatusCode)
		if err != nil {
			return nil, fmt.Errorf("response %d: %w", i, err)
		}

		// Create response with value objects
		resp, err := traffictesting.NewResponse(
			rtype,
			statusCode,
			traffictesting.NewHeaders(dto.Headers),
			dto.Body,
			dto.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("response %d: failed to create response: %w", i, err)
		}
		responses = append(responses, *resp)

		switch resp.Type {
		case traffictesting.ResponseTypeLive:
			live = resp
		case traffictesting.ResponseTypeShadow:
			shadow = resp
		}
	}

	if len(responses) != 2 {
		return nil, fmt.Errorf("exactly two responses are required, got %d", len(responses))
	}

	if live == nil {
		return nil, fmt.Errorf("live response is required")
	}

	if shadow == nil {
		return nil, fmt.Errorf("shadow response is required")
	}

	// Compute or use provided diff content
	var diffContent []diff.PatchOp
	if cmd.Diff != nil {
		// Agent provided pre-computed diff (backward compatibility)
		diffContent = cmd.Diff.Content
	} else {
		// Compute diff server-side from response bodies
		computed, err := computeDiff(cmd.Responses)
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
		responses,
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


// computeDiff computes a JSON diff between live and shadow response bodies.
// Response bodies are expected to be base64-encoded JSON strings.
// Returns RFC 6902 JSON Patch operations describing the differences.
func computeDiff(responses []CreateRequestResponseProps) ([]diff.PatchOp, error) {
	var live, shadow *CreateRequestResponseProps
	for i := range responses {
		switch responses[i].Type {
		case "live":
			live = &responses[i]
		case "shadow":
			shadow = &responses[i]
		}
	}

	if live == nil || shadow == nil {
		return nil, fmt.Errorf("both live and shadow responses are required for diff computation")
	}

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