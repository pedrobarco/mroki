package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/pedrobarco/mroki/pkg/dto"
)

type MrokiClient struct {
	baseURL    *url.URL
	gateID     string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewMrokiClient creates a new API client.
// Resilience (retries, circuit breaker) and auth are expected to be
// configured in the injected http.Client's transport layer.
func NewMrokiClient(apiURL *url.URL, gateID string, opts ...ClientOption) *MrokiClient {
	client := &MrokiClient{
		baseURL:    apiURL,
		gateID:     gateID,
		httpClient: &http.Client{},
		logger:     slog.Default(),
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// ClientOption allows configuring the client
type ClientOption func(*MrokiClient)

func WithHTTPClient(c *http.Client) ClientOption {
	return func(mc *MrokiClient) { mc.httpClient = c }
}

func WithLogger(l *slog.Logger) ClientOption {
	return func(mc *MrokiClient) { mc.logger = l }
}

// SendRequest sends a captured request to the API.
// Retry and circuit-breaker logic are handled by the HTTP transport layer.
func (c *MrokiClient) SendRequest(ctx context.Context, req *CapturedRequest) error {
	return c.sendRequestOnce(ctx, req)
}

// GetGate fetches gate configuration from the API
// Returns gate with live_url and shadow_url
func (c *MrokiClient) GetGate(ctx context.Context) (*dto.Gate, error) {
	// Build URL: GET /gates/{gate_id}
	endpoint := fmt.Sprintf("%s/gates/%s", c.baseURL.String(), c.gateID)

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.Warn("failed to close response body", "error", closeErr)
		}
	}()

	// Check status code
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("gate not found: %s", c.gateID)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Try to parse RFC 7807 error response
		var apiErr dto.APIError
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil {
			return nil, fmt.Errorf("API error (status %d): %s - %s",
				apiErr.Status, apiErr.Title, apiErr.Detail)
		}
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Parse response
	var response dto.Response[dto.Gate]
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response.Data, nil
}

// sendRequestOnce makes a single API call without retry
func (c *MrokiClient) sendRequestOnce(ctx context.Context, req *CapturedRequest) error {
	// Build URL: POST /gates/{gate_id}/requests
	endpoint := fmt.Sprintf("%s/gates/%s/requests", c.baseURL.String(), c.gateID)

	// Marshal request to JSON
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	if req.ID != "" {
		httpReq.Header.Set("X-Request-ID", req.ID)
	}

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.Warn("failed to close response body", "error", closeErr)
		}
	}()

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Try to parse RFC 7807 error response
		var apiErr dto.APIError
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil {
			// Successfully parsed RFC 7807 error
			return fmt.Errorf("API error (status %d): %s - %s [type: %s, instance: %s]",
				apiErr.Status, apiErr.Title, apiErr.Detail, apiErr.Type, apiErr.Instance)
		}
		// Fallback to generic error if parsing failed
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	return nil
}
