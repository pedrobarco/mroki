package dto

import (
	"net/http"
	"time"

	"github.com/pedrobarco/mroki/pkg/diff"
)

// CreateRequestPayload represents the payload for creating a request with responses and diff.
// This is sent from proxies to the API.
type CreateRequestPayload struct {
	// Request metadata
	ID        string              `json:"id,omitempty"` // Optional: API will generate if empty
	Method    string              `json:"method"`       // e.g., "GET", "POST"
	Path      string              `json:"path"`         // e.g., "/api/users/123"
	Headers   map[string][]string `json:"headers"`      // HTTP headers
	Body      string              `json:"body"`         // Base64 encoded
	CreatedAt time.Time           `json:"created_at"`   // When request was captured

	// Responses from both services
	LiveResponse   ResponsePayload `json:"live_response"`
	ShadowResponse ResponsePayload `json:"shadow_response"`

	// Computed diff (optional — if omitted, mroki-api computes it server-side)
	Diff *DiffPayload `json:"diff,omitempty"`
}

// ResponsePayload represents a single HTTP response.
type ResponsePayload struct {
	ID         string              `json:"id,omitempty"` // Optional
	StatusCode int                 `json:"status_code"`  // e.g., 200, 404, 500
	Headers    map[string][]string `json:"headers"`      // Response headers
	Body       string              `json:"body"`         // Base64 encoded
	LatencyMs  int64               `json:"latency_ms"`   // Response time in milliseconds
	CreatedAt  time.Time           `json:"created_at"`   // Same as request
}

// DiffPayload contains the computed difference between responses.
type DiffPayload struct {
	Content []diff.PatchOp `json:"content"` // RFC 6902 JSON Patch operations
}

// ResponseSummary represents a lightweight response summary (used in listings).
type ResponseSummary struct {
	StatusCode int   `json:"status_code"`
	LatencyMs  int64 `json:"latency_ms"`
}

// Request represents a summary of a captured request (used in listings).
type Request struct {
	ID             string           `json:"id"`
	Method         string           `json:"method"`
	Path           string           `json:"path"`
	CreatedAt      time.Time        `json:"created_at"`
	LiveResponse   *ResponseSummary `json:"live_response"`
	ShadowResponse *ResponseSummary `json:"shadow_response"`
	HasDiff        bool             `json:"has_diff"`
}

// RequestDetail represents a complete request with all responses and diff.
type RequestDetail struct {
	ID        string              `json:"id"`
	Method    string              `json:"method"`
	Path      string              `json:"path"`
	Headers   map[string][]string `json:"headers"`
	Body      string              `json:"body"`
	CreatedAt time.Time           `json:"created_at"`

	LiveResponse   ResponseDetail `json:"live_response"`
	ShadowResponse ResponseDetail `json:"shadow_response"`
	Diff           DiffDetail     `json:"diff"`
}

// ResponseDetail represents a response with full details (used in request detail view).
type ResponseDetail struct {
	ID         string      `json:"id"`
	StatusCode int         `json:"status_code"`
	Headers    http.Header `json:"headers"`
	Body       string      `json:"body"`
	LatencyMs  int64       `json:"latency_ms"`
	CreatedAt  time.Time   `json:"created_at"`
}

// DiffDetail represents diff content (used in request detail view).
type DiffDetail struct {
	Content []diff.PatchOp `json:"content"` // RFC 6902 JSON Patch operations
}
