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
	"github.com/pedrobarco/mroki/internal/application/services"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/pedrobarco/mroki/pkg/diff"
)

// CreateRequestCommand represents the intent to create a new request with responses and diff
type CreateRequestCommand struct {
	ID             string
	GateID         string
	Method         string
	Path           string
	RawQuery       string
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

	// Decode base64 bodies
	reqBodyDecoded, reqDecodeErr := decodeBase64Body(cmd.Body)
	liveBodyDecoded, liveDecodeErr := decodeBase64Body(cmd.LiveResponse.Body)
	shadowBodyDecoded, shadowDecodeErr := decodeBase64Body(cmd.ShadowResponse.Body)

	// Build diff options from gate config
	var diffOpts []diff.Option
	if gateErr == nil {
		diffOpts = gate.DiffConfig.ToDiffOptions()
	}

	// Redact all three inputs and compute diff
	comparer := services.NewResponseComparer(redactor, diffOpts)
	result, err := comparer.Compare(
		services.ResponseData{Headers: cmd.Headers, Body: reqBodyDecoded},
		services.ResponseData{StatusCode: cmd.LiveResponse.StatusCode, Headers: cmd.LiveResponse.Headers, Body: liveBodyDecoded},
		services.ResponseData{StatusCode: cmd.ShadowResponse.StatusCode, Headers: cmd.ShadowResponse.Headers, Body: shadowBodyDecoded},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to redact/diff: %w", err)
	}

	// Update command headers with redacted versions
	cmd.Headers = result.Request.Headers
	cmd.LiveResponse.Headers = result.Live.Headers
	cmd.ShadowResponse.Headers = result.Shadow.Headers

	// Convert redacted bodies to json.RawMessage for JSONB storage
	reqBodyJSON, err := redactResultToRawMessage(reqDecodeErr, result.Request, cmd.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}
	liveBodyJSON, err := redactResultToRawMessage(liveDecodeErr, result.Live, cmd.LiveResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal live response body: %w", err)
	}
	shadowBodyJSON, err := redactResultToRawMessage(shadowDecodeErr, result.Shadow, cmd.ShadowResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal shadow response body: %w", err)
	}

	// Parse and create live response (with already-redacted headers + body)
	live, err := buildResponse(cmd.LiveResponse, liveBodyJSON)
	if err != nil {
		return nil, fmt.Errorf("live response: %w", err)
	}

	// Parse and create shadow response (with already-redacted headers + body)
	shadow, err := buildResponse(cmd.ShadowResponse, shadowBodyJSON)
	if err != nil {
		return nil, fmt.Errorf("shadow response: %w", err)
	}

	// Use pre-provided diff or computed diff
	var diffContent []diff.PatchOp
	if cmd.Diff != nil {
		diffContent = cmd.Diff.Content
	} else {
		diffContent = result.Ops
	}

	// Snapshot the gate's diff config so the frontend knows how to interpret
	// the patch indices even if the gate config changes later.
	var diffConfig traffictesting.DiffConfig
	if gateErr == nil {
		diffConfig = gate.DiffConfig
	}

	d, err := traffictesting.NewDiff(live.ID, shadow.ID, diffContent, diffConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create diff: %w", err)
	}

	// Create request aggregate with value objects
	request, err := traffictesting.NewRequest(
		gateID,
		method,
		path,
		cmd.RawQuery,
		traffictesting.NewHeaders(cmd.Headers),
		reqBodyJSON,
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
// bodyJSON is the redacted body as json.RawMessage for JSONB storage.
func buildResponse(props CreateRequestResponseProps, bodyJSON json.RawMessage) (*traffictesting.Response, error) {
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
		bodyJSON,
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

// bodyToRawMessage converts a redacted body to json.RawMessage for JSONB storage.
//   - JSON bodies: marshal the parsed tree (BodyParsed) to json.RawMessage
//   - Non-JSON bodies: wrap raw bytes as a JSON string value
//   - Empty bodies: return nil (→ NULL in DB)
func bodyToRawMessage(rawBody []byte, bodyParsed any) (json.RawMessage, error) {
	if len(rawBody) == 0 {
		return nil, nil
	}
	if bodyParsed != nil {
		b, err := json.Marshal(bodyParsed)
		if err != nil {
			return nil, fmt.Errorf("marshal parsed body: %w", err)
		}
		return json.RawMessage(b), nil
	}
	// Non-JSON body: store as a JSON string
	return rawBytesToJSONString(rawBody)
}

// rawBytesToJSONString wraps arbitrary bytes as a JSON string value.
// Used for non-JSON bodies and for bodies where base64 decoding failed.
func rawBytesToJSONString(raw []byte) (json.RawMessage, error) {
	b, err := json.Marshal(string(raw))
	if err != nil {
		return nil, fmt.Errorf("marshal raw bytes as string: %w", err)
	}
	return json.RawMessage(b), nil
}

// redactResultToRawMessage converts a RedactResult to json.RawMessage for JSONB storage.
// If base64 decode failed (decodeErr != nil), falls back to storing the original bytes as a JSON string.
func redactResultToRawMessage(decodeErr error, result traffictesting.RedactResult, originalBody []byte) (json.RawMessage, error) {
	if decodeErr == nil {
		return bodyToRawMessage(result.Body, result.BodyParsed)
	}
	if len(originalBody) > 0 {
		return rawBytesToJSONString(originalBody)
	}
	return nil, nil
}
