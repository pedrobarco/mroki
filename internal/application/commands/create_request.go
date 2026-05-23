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

	// Convert redacted bodies to json.RawMessage for JSONB storage.
	// If base64 decode succeeded, use the redacted body/parsed tree.
	// If decode failed, preserve the original bytes as a JSON string.
	var reqBodyJSON json.RawMessage
	if reqDecodeErr == nil {
		reqBodyJSON, err = bodyToRawMessage(reqResult.Body, reqResult.BodyParsed)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	} else if len(cmd.Body) > 0 {
		reqBodyJSON, err = rawBytesToJSONString(cmd.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal raw request body: %w", err)
		}
	}

	// Decode base64 response bodies once, redact headers + body
	liveBodyDecoded, liveDecodeErr := decodeBase64Body(cmd.LiveResponse.Body)
	shadowBodyDecoded, shadowDecodeErr := decodeBase64Body(cmd.ShadowResponse.Body)

	liveResult, err := redactor.Redact(cmd.LiveResponse.Headers, liveBodyDecoded)
	if err != nil {
		return nil, fmt.Errorf("failed to redact live response: %w", err)
	}
	cmd.LiveResponse.Headers = liveResult.Headers

	shadowResult, err := redactor.Redact(cmd.ShadowResponse.Headers, shadowBodyDecoded)
	if err != nil {
		return nil, fmt.Errorf("failed to redact shadow response: %w", err)
	}
	cmd.ShadowResponse.Headers = shadowResult.Headers

	var liveBodyJSON, shadowBodyJSON json.RawMessage
	if liveDecodeErr == nil {
		liveBodyJSON, err = bodyToRawMessage(liveResult.Body, liveResult.BodyParsed)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal live response body: %w", err)
		}
	} else if len(cmd.LiveResponse.Body) > 0 {
		liveBodyJSON, err = rawBytesToJSONString(cmd.LiveResponse.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal raw live response body: %w", err)
		}
	}
	if shadowDecodeErr == nil {
		shadowBodyJSON, err = bodyToRawMessage(shadowResult.Body, shadowResult.BodyParsed)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal shadow response body: %w", err)
		}
	} else if len(cmd.ShadowResponse.Body) > 0 {
		shadowBodyJSON, err = rawBytesToJSONString(cmd.ShadowResponse.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal raw shadow response body: %w", err)
		}
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

	// Compute or use provided diff content
	var diffContent []diff.PatchOp
	if cmd.Diff != nil {
		diffContent = cmd.Diff.Content
	} else {
		var diffOpts []diff.Option
		if gateErr == nil {
			diffOpts = gate.DiffConfig.ToDiffOptions()
		}

		// Use pre-parsed trees from redaction to avoid re-parsing.
		// BuildEnvelope wraps statusCode + headers + body into the diff-ready structure.
		liveEnvelope := diff.BuildEnvelope(cmd.LiveResponse.StatusCode, cmd.LiveResponse.Headers, liveResult.BodyParsed)
		shadowEnvelope := diff.BuildEnvelope(cmd.ShadowResponse.StatusCode, cmd.ShadowResponse.Headers, shadowResult.BodyParsed)
		computed, err := diff.Parsed(liveEnvelope, shadowEnvelope, diffOpts...)
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
