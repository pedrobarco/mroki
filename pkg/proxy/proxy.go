package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pedrobarco/mroki/pkg/diff"
)

var (
	defaultLiveTimeout   = 5 * time.Second
	defaultShadowTimeout = 10 * time.Second
)

// ProxyRequest represents the original HTTP request being proxied
type ProxyRequest struct {
	Method  string
	Path    string
	Headers http.Header
	Body    []byte
}

// DiffResult contains the result of comparing live and shadow responses
type DiffResult struct {
	Content string // The diff output (empty if responses match or not JSON)
	Error   error  // Error from diffing, if any
}

type CallbackFunc func(req ProxyRequest, live, shadow ProxyResponse, diff DiffResult) error

type Proxy struct {
	Live   *url.URL
	Shadow *url.URL

	liveTimeout   time.Duration
	shadowTimeout time.Duration
	checks        []CheckFunc
	differ        *proxyResponseDiffer
	callbackFn    CallbackFunc
	logger        *slog.Logger
	client        *http.Client
}

var (
	_ http.Handler = (*Proxy)(nil)
)

type Option func(*Proxy)

// WithLiveTimeout sets the timeout for live requests
func WithLiveTimeout(timeout time.Duration) Option {
	return func(p *Proxy) {
		p.liveTimeout = timeout
	}
}

// WithShadowTimeout sets the timeout for shadow requests
func WithShadowTimeout(timeout time.Duration) Option {
	return func(p *Proxy) {
		p.shadowTimeout = timeout
	}
}

// WithShouldProxyToShadow adds check functions to determine if shadow should be proxied
// Multiple checks can be provided - all must return true (AND logic)
// Can be called multiple times to add more checks
func WithShouldProxyToShadow(checks ...CheckFunc) Option {
	return func(p *Proxy) {
		p.checks = append(p.checks, checks...)
	}
}

// WithDiffOptions configures the differ with custom options
// If not called, a default differ with no options is used
func WithDiffOptions(opts ...diff.Option) Option {
	return func(p *Proxy) {
		p.differ = NewProxyResponseDiffer(opts...)
	}
}

func WithCallbackFn(fn CallbackFunc) Option {
	return func(p *Proxy) {
		p.callbackFn = fn
	}
}

// WithHTTPClient allows setting a custom HTTP client
func WithHTTPClient(client *http.Client) Option {
	return func(p *Proxy) {
		p.client = client
	}
}

// newDefaultHTTPClient creates an HTTP client with sensible defaults
// for connection pooling and timeouts
func newDefaultHTTPClient() *http.Client {
	return &http.Client{
		// No client-level timeout - we use context timeouts
		Timeout: 0,
		Transport: &http.Transport{
			// Connection pool settings
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			MaxConnsPerHost:     100,
			IdleConnTimeout:     90 * time.Second,

			// Timeout settings
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 0, // Use context timeout
			ExpectContinueTimeout: 1 * time.Second,

			// Performance
			DisableKeepAlives:  false,
			DisableCompression: false,
			ForceAttemptHTTP2:  true,
		},
	}
}

func NewProxy(live, shadow *url.URL, opts ...Option) *Proxy {
	proxy := &Proxy{
		Live:          live,
		Shadow:        shadow,
		liveTimeout:   defaultLiveTimeout,
		shadowTimeout: defaultShadowTimeout,
		checks:        []CheckFunc{},
		differ:        NewProxyResponseDiffer(),
		callbackFn:    defaultCallbackFn(),
		logger:        slog.Default(),
		client:        newDefaultHTTPClient(),
	}

	for _, o := range opts {
		o(proxy)
	}

	return proxy
}

type ProxyResponse struct {
	StatusCode int
	Response   *http.Response
	Body       []byte
}

type responseResult struct {
	resp *http.Response
	body []byte
	err  error
}

// shouldProxyToShadow evaluates all registered checks
func (p *Proxy) shouldProxyToShadow(r *http.Request) bool {
	// No checks = always proxy
	if len(p.checks) == 0 {
		return true
	}

	// All checks must pass (AND logic)
	for _, check := range p.checks {
		if !check(r) {
			p.logger.Info("skipping shadow proxy",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int64("content_length", r.ContentLength),
			)
			return false
		}
	}

	return true
}

// ServeHTTP implements the http.Handler interface for proxying requests.
//
// Request Flow:
//  1. Reads request body (needed to replay to both services)
//  2. Launches live request with client context + timeout
//  3. Launches shadow request (if sampled) with independent context + timeout
//  4. Waits for live response and returns to client immediately
//  5. Compares responses in background goroutine
//
// Context Handling:
//   - Live requests inherit the client's request context (r.Context())
//     This means if the client disconnects, live request is cancelled
//   - Shadow requests use context.Background() as parent context
//     This makes shadow independent of client connection state, ensuring
//     complete diff data collection even if client disconnects
//   - Both contexts have their own timeout values for safety
//
// Timeouts:
//   - liveTimeout: Controls how long to wait for live service (blocks client)
//   - shadowTimeout: Controls how long shadow service can run (doesn't block client)
//   - Shadow timeout can be longer since it doesn't impact user experience
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check if we should proxy to shadow BEFORE reading body
	if !p.shouldProxyToShadow(r) {
		p.proxyToLiveOnly(w, r)
		return
	}

	liveCtx, liveCancel := context.WithTimeout(r.Context(), p.liveTimeout)
	defer liveCancel()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	// Best effort close - log error but don't fail the request
	// The body has already been read successfully
	if err := r.Body.Close(); err != nil {
		p.logger.Warn("failed to close request body", "error", err)
	}

	liveCh := make(chan responseResult, 1)
	shadowCh := make(chan responseResult, 1)

	// Launch live request
	go func() {
		resp, err := p.forwardRequest(liveCtx, r, p.Live, body)
		if err != nil {
			liveCh <- responseResult{err: err}
			return
		}
		b, err := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if err != nil {
			liveCh <- responseResult{err: err}
			return
		}
		if closeErr != nil {
			liveCh <- responseResult{err: closeErr}
			return
		}
		liveCh <- responseResult{resp: resp, body: b, err: nil}
	}()

	// Shadow uses Background() context so it's independent of client connection
	// This allows shadow requests to complete even if the client disconnects,
	// ensuring we collect complete diff data for monitoring
	shadowCtx, shadowCancel := context.WithTimeout(context.Background(), p.shadowTimeout)
	// Launch shadow request
	go func() {
		resp, err := p.forwardRequest(shadowCtx, r, p.Shadow, body)
		if err != nil {
			shadowCh <- responseResult{err: err}
			return
		}
		b, err := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if err != nil {
			shadowCh <- responseResult{err: err}
			return
		}
		if closeErr != nil {
			shadowCh <- responseResult{err: closeErr}
			return
		}
		shadowCh <- responseResult{resp: resp, body: b, err: nil}
	}()

	// Wait for live first
	var liveResp responseResult
	select {
	case liveResp = <-liveCh:
	case <-liveCtx.Done():
		http.Error(w, "timeout waiting for live response", http.StatusGatewayTimeout)
		return
	}

	if liveResp.err != nil {
		http.Error(w, "live backend error: "+liveResp.err.Error(), http.StatusBadGateway)
		return
	}

	// Write live response to client
	copyHeader(liveResp.resp.Header, w.Header())
	w.WriteHeader(liveResp.resp.StatusCode)
	if _, err = w.Write(liveResp.body); err != nil {
		// Can't call http.Error after writing response body
		p.logger.Error("failed to write response body", "error", err)
		return
	}

	// Wait for shadow and compare in background
	go func(liveBody []byte) {
		// Cancel shadow context when this goroutine completes
		// This ensures context is not cancelled before we read from shadowCh
		defer shadowCancel()

		select {
		case shadowResp := <-shadowCh:
			if shadowResp.err != nil {
				p.logger.Error("shadow request error", "error", shadowResp.err)
				return
			}

			proxyReq := ProxyRequest{
				Method:  r.Method,
				Path:    r.URL.Path,
				Headers: r.Header.Clone(),
				Body:    body,
			}
			live := ProxyResponse{
				StatusCode: liveResp.resp.StatusCode,
				Response:   liveResp.resp,
				Body:       liveBody,
			}
			shadow := ProxyResponse{
				StatusCode: shadowResp.resp.StatusCode,
				Response:   shadowResp.resp,
				Body:       shadowResp.body,
			}

			// Compute diff before calling callback
			var diffResult DiffResult

			// Check if both responses are JSON
			if isJSONContent(live.Response) && isJSONContent(shadow.Response) {
				content, err := p.differ.Diff(live, shadow)
				diffResult = DiffResult{
					Content: content,
					Error:   err,
				}
			} else {
				// Skip diffing for non-JSON, log it
				p.logger.Debug("skipping diff for non-JSON responses",
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.String("live_content_type", live.Response.Header.Get("Content-Type")),
					slog.String("shadow_content_type", shadow.Response.Header.Get("Content-Type")),
				)
			}

			// Invoke callback with diff result
			if err := p.callbackFn(proxyReq, live, shadow, diffResult); err != nil {
				p.logger.Error("callback error", "error", err)
			}

		case <-shadowCtx.Done():
			// Shadow request timed out or was cancelled
			p.logger.Error("shadow request timeout", "timeout", p.shadowTimeout)
		}
	}(liveResp.body)
}

func (p *Proxy) forwardRequest(ctx context.Context, original *http.Request, target *url.URL, body []byte) (*http.Response, error) {
	url := rewriteRequestURL(original, target)

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	} else {
		bodyReader = original.Body // Use original body for streaming
	}

	req, err := http.NewRequestWithContext(ctx, original.Method, url.String(), bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header = original.Header.Clone()
	p.logger.Debug("forwarding request",
		"method", req.Method,
		"url", req.URL.String())
	return p.client.Do(req)
}

func copyHeader(src, dst http.Header) {
	// Use Add instead of Set to preserve multiple header values
	// (e.g., Set-Cookie, Accept-Encoding can have multiple values)
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func rewriteRequestURL(original *http.Request, target *url.URL) *url.URL {
	// Copy the original URL structure
	newURL := *original.URL

	// Overwrite scheme and host
	newURL.Scheme = target.Scheme
	newURL.Host = target.Host

	// Join paths without trailing slash
	newURL.Path = strings.TrimSuffix(
		strings.TrimRight(target.Path, "/")+"/"+strings.TrimLeft(original.URL.Path, "/"),
		"/",
	)

	// Merge query parameters (original takes precedence)
	mergedQuery := target.Query()
	maps.Copy(mergedQuery, original.URL.Query())
	newURL.RawQuery = mergedQuery.Encode()

	return &newURL
}

// isJSONContent checks if the response has JSON content type
func isJSONContent(resp *http.Response) bool {
	if resp == nil {
		return false
	}
	contentType := resp.Header.Get("Content-Type")
	return strings.Contains(contentType, "application/json")
}

func defaultCallbackFn() CallbackFunc {
	logger := slog.Default()

	return func(req ProxyRequest, live, shadow ProxyResponse, diff DiffResult) error {
		// Handle diff error
		if diff.Error != nil {
			logger.Warn("failed to diff responses",
				slog.String("error", diff.Error.Error()),
				slog.Int("live_status", live.StatusCode),
				slog.Int("shadow_status", shadow.StatusCode),
			)
			return nil
		}

		// Log diff result
		if diff.Content != "" {
			logger.Info("response diff detected",
				slog.String("method", req.Method),
				slog.String("path", req.Path),
				slog.Int("live_status", live.StatusCode),
				slog.Int("shadow_status", shadow.StatusCode),
			)
			// Print the diff content in human-readable format
			fmt.Println("Diff:")
			fmt.Println(diff.Content)
		} else {
			logger.Debug("responses match",
				slog.String("method", req.Method),
				slog.String("path", req.Path),
				slog.Int("live_status", live.StatusCode),
				slog.Int("shadow_status", shadow.StatusCode),
			)
		}

		return nil
	}
}

// proxyToLiveOnly forwards request to live service only, skipping shadow/diff
// Used when body size exceeds limit or chunked encoding is detected
func (p *Proxy) proxyToLiveOnly(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), p.liveTimeout)
	defer cancel()

	// Forward request to live service (pass nil for body to use original body)
	resp, err := p.forwardRequest(ctx, r, p.Live, nil)
	if err != nil {
		http.Error(w, "live backend error: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			p.logger.Error("failed to close response body", "error", closeErr)
		}
	}()

	// Copy response to client
	copyHeader(resp.Header, w.Header())
	w.WriteHeader(resp.StatusCode)

	// Stream body directly (don't buffer)
	if _, err := io.Copy(w, resp.Body); err != nil {
		p.logger.Error("failed to copy response body", "error", err)
	}
}
