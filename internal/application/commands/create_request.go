package commands

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
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
	repo     traffictesting.RequestRepository
	gateRepo traffictesting.GateRepository
}

// NewCreateRequestHandler creates a new CreateRequestHandler
func NewCreateRequestHandler(repo traffictesting.RequestRepository, gateRepo traffictesting.GateRepository) *CreateRequestHandler {
	return &CreateRequestHandler{repo: repo, gateRepo: gateRepo}
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

	// Fetch gate for redacted fields and diff config
	gate, gateErr := h.gateRepo.GetByID(ctx, gateID)
	if gateErr != nil && !errors.Is(gateErr, traffictesting.ErrGateNotFound) {
		return nil, fmt.Errorf("failed to fetch gate: %w", gateErr)
	}

	// Build redactor from gate config (or defaults)
	var redactedFields traffictesting.RedactedFields
	if gateErr == nil {
		redactedFields = gate.RedactedFields
	} else {
		redactedFields = traffictesting.DefaultRedactedFields()
	}
	redactor := traffictesting.NewRedactor(redactedFields.AllFields())

	// Decode and redact request headers + body
	reqBodyDecoded, reqDecodeErr := decodeBase64Body(cmd.Body)
	reqResult, err := redactor.Redact(cmd.Headers, reqBodyDecoded)
	if err != nil {
		return nil, fmt.Errorf("failed to redact request: %w", err)
	}
	cmd.Headers = reqResult.Headers
	if reqDecodeErr == nil {
		cmd.Body = encodeBase64Body(reqResult.Body)
	}

	// Decode base64 response bodies once, redact headers + body, re-encode
	liveBodyDecoded, liveDecodeErr := decodeBase64Body(cmd.LiveResponse.Body)
	shadowBodyDecoded, shadowDecodeErr := decodeBase64Body(cmd.ShadowResponse.Body)

	liveResult, err := redactor.Redact(cmd.LiveResponse.Headers, liveBodyDecoded)
	if err != nil {
		return nil, fmt.Errorf("failed to redact live response: %w", err)
	}
	cmd.LiveResponse.Headers = liveResult.Headers
	liveBodyDecoded = liveResult.Body

	shadowResult, err := redactor.Redact(cmd.ShadowResponse.Headers, shadowBodyDecoded)
	if err != nil {
		return nil, fmt.Errorf("failed to redact shadow response: %w", err)
	}
	cmd.ShadowResponse.Headers = shadowResult.Headers
	shadowBodyDecoded = shadowResult.Body

	// Re-encode redacted bodies back to base64 for storage
	if liveDecodeErr == nil {
		cmd.LiveResponse.Body = encodeBase64Body(liveBodyDecoded)
	}
	if shadowDecodeErr == nil {
		cmd.ShadowResponse.Body = encodeBase64Body(shadowBodyDecoded)
	}

	// Parse and create live response (with already-redacted headers + body)
	live, err := buildResponse(cmd.LiveResponse)
	if err != nil {
		return nil, fmt.Errorf("live response: %w", err)
	}

	// Parse and create shadow response (with already-redacted headers + body)
	shadow, err := buildResponse(cmd.ShadowResponse)
	if err != nil {
		return nil, fmt.Errorf("shadow response: %w", err)
	}

	// Compute or use provided diff content
	var diffContent []diff.PatchOp
	if cmd.Diff != nil {
		diffContent = cmd.Diff.Content
	} else {
		var diffOpts []diff.Option
		if gateErr == nil {
			diffOpts = gate.DiffConfig.ToDiffOptions()
		}

		// Use already-decoded bodies to avoid double base64 decode
		computed, err := computeDiffFromDecoded(
			cmd.LiveResponse, liveBodyDecoded,
			cmd.ShadowResponse, shadowBodyDecoded,
			diffOpts...,
		)
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

// decodeBase64Body decodes a base64-encoded body. Returns the raw bytes or an error.
func decodeBase64Body(body []byte) ([]byte, error) {
	if len(body) == 0 {
		return body, nil
	}
	return base64.StdEncoding.DecodeString(string(body))
}

// encodeBase64Body re-encodes raw bytes to base64 for storage.
func encodeBase64Body(body []byte) []byte {
	return []byte(base64.StdEncoding.EncodeToString(body))
}

// computeDiffFromDecoded computes a JSON diff using already-decoded body bytes.
// This avoids double base64 decoding when bodies have been pre-decoded for redaction.
func computeDiffFromDecoded(
	live CreateRequestResponseProps, liveBody []byte,
	shadow CreateRequestResponseProps, shadowBody []byte,
	opts ...diff.Option,
) ([]diff.PatchOp, error) {
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
	liveBodyJSON := jsonBodyOrNull(liveBody)
	shadowBodyJSON := jsonBodyOrNull(shadowBody)

	liveJSON := fmt.Sprintf(`{"statusCode": %d, "headers": %s, "body": %s}`,
		live.StatusCode, liveHeaders, liveBodyJSON)
	shadowJSON := fmt.Sprintf(`{"statusCode": %d, "headers": %s, "body": %s}`,
		shadow.StatusCode, shadowHeaders, shadowBodyJSON)

	return diff.JSON(liveJSON, shadowJSON, opts...)
}

// jsonBodyOrNull returns the body bytes as a string for JSON embedding,
// or "null" if the body is empty/nil. This prevents producing invalid JSON
// like `"body": ` when body decoding failed or body was empty.
func jsonBodyOrNull(body []byte) string {
	if len(body) == 0 {
		return "null"
	}
	return string(body)
}