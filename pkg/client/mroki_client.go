package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"time"
)

type MrokiClient struct {
	baseURL      *url.URL
	gateID       string
	agentID      string
	httpClient   *http.Client
	maxRetries   int
	initialDelay time.Duration
	logger       *slog.Logger
}

// NewMrokiClient creates a new API client
func NewMrokiClient(apiURL *url.URL, gateID, agentID string, opts ...ClientOption) *MrokiClient {
	client := &MrokiClient{
		baseURL:      apiURL,
		gateID:       gateID,
		agentID:      agentID,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		maxRetries:   3,
		initialDelay: 1 * time.Second,
		logger:       slog.Default(),
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

func WithMaxRetries(n int) ClientOption {
	return func(mc *MrokiClient) { mc.maxRetries = n }
}

func WithInitialDelay(d time.Duration) ClientOption {
	return func(mc *MrokiClient) { mc.initialDelay = d }
}

func WithLogger(l *slog.Logger) ClientOption {
	return func(mc *MrokiClient) { mc.logger = l }
}

// SendRequest sends a captured request to the API with retry logic
func (c *MrokiClient) SendRequest(ctx context.Context, req *CapturedRequest) error {
	// Ensure agent_id is set
	req.AgentID = c.agentID

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			delay := c.initialDelay * time.Duration(math.Pow(2, float64(attempt-1)))

			c.logger.Info("retrying API request",
				"attempt", attempt,
				"delay", delay,
				"gate_id", c.gateID,
			)

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		err := c.sendRequestOnce(ctx, req)
		if err == nil {
			if attempt > 0 {
				c.logger.Info("request succeeded after retry",
					"attempts", attempt+1,
					"gate_id", c.gateID,
				)
			}
			return nil
		}

		lastErr = err
		c.logger.Warn("API request failed",
			"attempt", attempt+1,
			"error", err,
			"gate_id", c.gateID,
		)
	}

	// All retries exhausted
	c.logger.Error("all retries exhausted",
		"attempts", c.maxRetries+1,
		"gate_id", c.gateID,
		"last_error", lastErr,
	)

	return fmt.Errorf("failed after %d attempts: %w", c.maxRetries+1, lastErr)
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

	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	return nil
}
