package client

import (
	"time"
)

// CapturedRequest represents a complete request/response pair with diff
// This is what gets sent from agent to API
type CapturedRequest struct {
	// Request metadata
	ID        string              `json:"id,omitempty"` // Optional: API will generate if empty
	AgentID   string              `json:"agent_id"`     // Required: identifies which agent
	Method    string              `json:"method"`       // e.g., "GET", "POST"
	Path      string              `json:"path"`         // e.g., "/api/users/123"
	Headers   map[string][]string `json:"headers"`      // HTTP headers
	Body      string              `json:"body"`         // Base64 encoded
	CreatedAt time.Time           `json:"created_at"`   // When request was captured

	// Responses from both services
	Responses []CapturedResponse `json:"responses"` // Always 2: live + shadow

	// Computed diff
	Diff CapturedDiff `json:"diff"`
}

// CapturedResponse represents a single HTTP response (live or shadow)
type CapturedResponse struct {
	ID         string              `json:"id,omitempty"` // Optional
	Type       string              `json:"type"`         // "live" or "shadow"
	StatusCode int                 `json:"status_code"`  // e.g., 200, 404, 500
	Headers    map[string][]string `json:"headers"`      // Response headers
	Body       string              `json:"body"`         // Base64 encoded
	CreatedAt  time.Time           `json:"created_at"`   // Same as request
}

// CapturedDiff contains the computed difference between responses
type CapturedDiff struct {
	Content string `json:"content"` // JSON diff format
}
