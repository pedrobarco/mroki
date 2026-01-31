package proxy

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"time"
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

type CallbackFunc func(req ProxyRequest, live, shadow ProxyResponse) error

type Proxy struct {
	Live   *url.URL
	Shadow *url.URL

	liveTimeout   time.Duration
	shadowTimeout time.Duration
	samplingRate  *SamplingRate
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

func WithSamplingRate(rate *SamplingRate) Option {
	return func(p *Proxy) {
		p.samplingRate = rate
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
		samplingRate:  nil,
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
	liveCtx, liveCancel := context.WithTimeout(r.Context(), p.liveTimeout)
	defer liveCancel()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	if err := r.Body.Close(); err != nil {
		http.Error(w, "Failed to close request body", http.StatusInternalServerError)
		return
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

	sample := p.samplingRate == nil || p.samplingRate.Sample()
	if sample {
		// Shadow uses Background() context so it's independent of client connection
		// This allows shadow requests to complete even if the client disconnects,
		// ensuring we collect complete diff data for monitoring
		shadowCtx, shadowCancel := context.WithTimeout(context.Background(), p.shadowTimeout)
		// Launch shadow request
		go func() {
			defer shadowCancel()
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
	}

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

	if !sample {
		return
	}

	// Wait for shadow and compare in background
	go func(liveBody []byte) {
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

			if err := p.callbackFn(proxyReq, live, shadow); err != nil {
				p.logger.Error("callback error", "error", err)
			}

		case <-time.After(p.shadowTimeout):
			p.logger.Error("shadow request timeout", "timeout", p.shadowTimeout)
		}
	}(liveResp.body)
}

func (p *Proxy) forwardRequest(ctx context.Context, original *http.Request, target *url.URL, body []byte) (*http.Response, error) {
	url := rewriteRequestURL(original, target)
	req, err := http.NewRequestWithContext(ctx, original.Method, url.String(), bytes.NewReader(body))
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

func defaultCallbackFn() CallbackFunc {
	isJSONContent := func(resp *http.Response) bool {
		contentType := resp.Header.Get("Content-Type")
		return strings.Contains(contentType, "application/json")
	}

	logger := slog.Default()
	differ := NewProxyResponseDiffer()

	return func(req ProxyRequest, live, shadow ProxyResponse) error {
		// Skip non-JSON responses
		if !isJSONContent(live.Response) || !isJSONContent(shadow.Response) {
			logger.Debug("skipping diff for non-JSON responses",
				"live_content_type", live.Response.Header.Get("Content-Type"),
				"shadow_content_type", shadow.Response.Header.Get("Content-Type"),
			)
			return nil
		}

		res, err := differ.Diff(live, shadow)
		if err != nil {
			// Log error but don't fail - shadow testing shouldn't break
			logger.Warn("failed to diff responses",
				"error", err,
				"live_status", live.StatusCode,
				"shadow_status", shadow.StatusCode,
			)
			// Return nil - we logged the issue, no need to fail callback
			return nil
		}

		if len(res) > 0 {
			logger.Info("response diff detected",
				"method", req.Method,
				"path", req.Path,
				"live_status", live.StatusCode,
				"shadow_status", shadow.StatusCode,
				"diff_length", len(res),
			)
		} else {
			logger.Debug("responses match",
				"method", req.Method,
				"path", req.Path,
				"live_status", live.StatusCode,
				"shadow_status", shadow.StatusCode,
			)
		}

		return nil
	}
}
