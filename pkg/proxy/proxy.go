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
	"runtime/debug"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	defaultLiveTimeout   = 5 * time.Second
	defaultShadowTimeout = 10 * time.Second
)

// ShadowHeader is the fixed header added to requests forwarded to the shadow
// service so downstream systems can distinguish shadow traffic from live
// traffic. The name is intentionally not configurable so consumers can rely on
// a stable convention. The live request path is never modified.
const ShadowHeader = "X-Mroki-Mode"

// ShadowHeaderValue is the value set on ShadowHeader for shadow requests.
// Consumers can match it to distinguish a shadow request from a live one even
// when both upstreams share a host (differing only by path or query).
const ShadowHeaderValue = "shadow"

// ProxyRequest represents the original HTTP request being proxied
type ProxyRequest struct {
	Method   string
	Path     string
	RawQuery string
	Headers  http.Header
	Body     []byte
}

type CallbackFunc func(req ProxyRequest, live, shadow ProxyResponse) error

type Proxy struct {
	Live   *url.URL
	Shadow *url.URL

	liveTimeout   time.Duration
	shadowTimeout time.Duration
	maxBodySize   int64
	checks        []CheckFunc
	callbackFn    CallbackFunc
	logger        *slog.Logger
	client        *http.Client
	// callbackSem bounds the number of concurrent background callback
	// goroutines. A nil semaphore means unbounded — this package is
	// intentionally unopinionated and holds no operational default; callers
	// that want a limit (e.g. the proxy binary's config layer) set one via
	// WithMaxConcurrentCallbacks.
	callbackSem chan struct{}
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

// WithMaxBodySize sets the maximum request body size (in bytes) that will be
// buffered for shadow replay. The limit governs shadowing and buffering only —
// live traffic is always forwarded regardless of size:
//   - Requests whose body exceeds the limit are still forwarded to live by
//     streaming (no full-body buffering); shadow and comparison are skipped.
//   - Requests with an unknown Content-Length (chunked) are streamed to live
//     while at most maxBytes are buffered for shadow; shadow is skipped if the
//     body turns out to exceed the limit.
//
// A value <= 0 means unlimited: the body is fully buffered and shadow is never
// skipped for size.
func WithMaxBodySize(maxBytes int64) Option {
	return func(p *Proxy) {
		p.maxBodySize = maxBytes
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

// WithLogger sets the logger for proxy operations
func WithLogger(logger *slog.Logger) Option {
	return func(p *Proxy) {
		p.logger = logger
	}
}

// WithMaxConcurrentCallbacks bounds the number of background callback
// goroutines that may run concurrently. When the limit is reached, further
// shadow comparisons are dropped (with a warning) instead of spawning unbounded
// goroutines under load — the live response is unaffected. A value <= 0 leaves
// the limit unset (unbounded); this package holds no default, so the limit is
// owned by the caller's config layer.
func WithMaxConcurrentCallbacks(n int) Option {
	return func(p *Proxy) {
		if n > 0 {
			p.callbackSem = make(chan struct{}, n)
		}
	}
}

// HTTPClientConfig holds the tunable connection-pool settings for the proxy's
// HTTP client. Values are applied verbatim by NewHTTPClient — net/http's zero
// semantics apply (0 means "no limit", or for IdleConnTimeout "no timeout").
// This package is intentionally unopinionated: it holds no operational
// defaults. Callers that need tuned pool sizes (e.g. the proxy binary's config
// layer) own those defaults and pass them in.
type HTTPClientConfig struct {
	// MaxIdleConns is the maximum number of idle connections across all hosts.
	MaxIdleConns int
	// MaxIdleConnsPerHost is the maximum number of idle connections per host.
	MaxIdleConnsPerHost int
	// MaxConnsPerHost limits the total number of connections per host.
	MaxConnsPerHost int
	// IdleConnTimeout is how long an idle connection is kept before closing.
	IdleConnTimeout time.Duration
}

// NewHTTPClient builds an HTTP client for proxying live and shadow traffic. The
// connection-pool fields from cfg are applied verbatim; the remaining transport
// settings (timeouts, keep-alive, HTTP/2) are fixed and not configurable.
func NewHTTPClient(cfg HTTPClientConfig) *http.Client {
	return &http.Client{
		// No client-level timeout - we use context timeouts
		Timeout: 0,
		Transport: &http.Transport{
			// Connection pool settings (configurable)
			MaxIdleConns:        cfg.MaxIdleConns,
			MaxIdleConnsPerHost: cfg.MaxIdleConnsPerHost,
			MaxConnsPerHost:     cfg.MaxConnsPerHost,
			IdleConnTimeout:     cfg.IdleConnTimeout,

			// Timeout settings
			// TLSHandshakeTimeout should not exceed the default live request
			// timeout (LIVE_TIMEOUT=5s). Context cancellation handles the
			// actual deadline; this is a safety net for the TLS phase.
			TLSHandshakeTimeout:   5 * time.Second,
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
		callbackFn:    defaultCallbackFn(),
		logger:        slog.Default(),
		// Zero-config client: net/http pool semantics. Callers that need tuned
		// pool sizes pass a client built via NewHTTPClient using WithHTTPClient.
		client: NewHTTPClient(HTTPClientConfig{}),
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
	LatencyMs  int64
}

type responseResult struct {
	resp      *http.Response
	body      []byte
	latencyMs int64
	err       error
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
			// A skipped shadow is routine per-request sampling behaviour, not
			// something an operator needs to see at info level.
			p.logger.Debug("skipping shadow proxy",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int64("content_length", r.ContentLength),
			)
			return false
		}
	}

	return true
}

// bodyPlan describes how the request body should be handled for a request that
// passed the sampling/shadow-rule checks. When shadow is true the body was fully
// buffered into body and can be replayed to both live and shadow. When shadow is
// false the body is too large (or unbounded) to buffer for comparison, so it is
// streamed to live via liveReader with liveContentLength and shadow is skipped.
type bodyPlan struct {
	shadow            bool
	body              []byte
	liveReader        io.Reader
	liveContentLength int64
}

// planBody decides how to handle the request body based on the configured
// maxBodySize and the request's Content-Length. It only buffers memory bounded
// by maxBodySize (or by a known, within-limit Content-Length); oversized or
// unbounded bodies are streamed to live without full-body buffering. The size
// limit never rejects live traffic — it only governs whether shadow runs.
func (p *Proxy) planBody(r *http.Request) (bodyPlan, error) {
	// Unlimited: buffer the whole body and always allow shadow.
	if p.maxBodySize <= 0 {
		body, err := p.readAndCloseBody(r)
		if err != nil {
			return bodyPlan{}, err
		}
		return bodyPlan{shadow: true, body: body}, nil
	}

	// Known Content-Length.
	if r.ContentLength >= 0 {
		if r.ContentLength > p.maxBodySize {
			// Stream live directly from the untouched body; skip shadow.
			return bodyPlan{shadow: false, liveReader: r.Body, liveContentLength: r.ContentLength}, nil
		}
		body, err := p.readAndCloseBody(r)
		if err != nil {
			return bodyPlan{}, err
		}
		return bodyPlan{shadow: true, body: body}, nil
	}

	// Unknown Content-Length (chunked): buffer at most maxBodySize+1 bytes to
	// detect whether the body fits within the limit without reading it all.
	buf, err := io.ReadAll(io.LimitReader(r.Body, p.maxBodySize+1))
	if err != nil {
		return bodyPlan{}, err
	}
	if int64(len(buf)) <= p.maxBodySize {
		// Whole body fits within the limit: safe to buffer and shadow.
		if closeErr := r.Body.Close(); closeErr != nil {
			p.logger.Warn("failed to close request body", slog.String("error", closeErr.Error()))
		}
		return bodyPlan{shadow: true, body: buf}, nil
	}

	// Body exceeds the limit: stream the buffered prefix plus the remaining body
	// to live (unknown length), and skip shadow.
	return bodyPlan{
		shadow:            false,
		liveReader:        io.MultiReader(bytes.NewReader(buf), r.Body),
		liveContentLength: -1,
	}, nil
}

// readAndCloseBody reads the full request body and best-effort closes it. A
// close error is logged but not returned, matching the original behaviour: the
// body has already been read successfully.
func (p *Proxy) readAndCloseBody(r *http.Request) ([]byte, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	if closeErr := r.Body.Close(); closeErr != nil {
		p.logger.Warn("failed to close request body", slog.String("error", closeErr.Error()))
	}
	return body, nil
}

// acquireCallbackSlot reserves a slot for a background callback goroutine. It
// returns true immediately when no limit is configured (nil semaphore) or when
// a slot is free, and false without blocking when the limit is reached.
func (p *Proxy) acquireCallbackSlot() bool {
	if p.callbackSem == nil {
		return true
	}
	select {
	case p.callbackSem <- struct{}{}:
		return true
	default:
		return false
	}
}

// releaseCallbackSlot returns a previously acquired callback slot. It is a no-op
// when no limit is configured.
func (p *Proxy) releaseCallbackSlot() {
	if p.callbackSem == nil {
		return
	}
	<-p.callbackSem
}

// recoverGoroutine contains a panic in a best-effort background goroutine so it
// never terminates the process or affects live traffic. It logs the panic with
// a stack trace at error level and, when provided, runs onPanic to let the
// caller unblock any waiter (e.g. by pushing an error onto its result channel).
// It must be used via defer inside the goroutine it protects.
func (p *Proxy) recoverGoroutine(logger *slog.Logger, name string, onPanic func(v any)) {
	if r := recover(); r != nil {
		logger.Error("recovered from panic in background goroutine",
			slog.String("goroutine", name),
			slog.Any("panic", r),
			slog.String("stack", string(debug.Stack())),
		)
		if onPanic != nil {
			onPanic(r)
		}
	}
}

// ServeHTTP implements the http.Handler interface for proxying requests.
//
// Request Flow:
//  1. Decides shadow eligibility and body handling (see planBody): within-limit
//     bodies are buffered for replay; oversized/unbounded bodies are streamed to
//     live only, without full-body buffering
//  2. Launches live request with client context + timeout
//  3. Launches shadow request (if sampled) with independent context + timeout
//  4. Waits for live response and returns to client immediately
//  5. Invokes callback with raw responses in background goroutine
//
// Context Handling:
//   - Live requests inherit the client's request context (r.Context())
//     This means if the client disconnects, live request is cancelled
//   - Shadow requests use context.Background() as parent context
//     This makes shadow independent of client connection state, ensuring
//     complete response data collection even if client disconnects
//   - Both contexts have their own timeout values for safety
//
// Timeouts:
//   - liveTimeout: Controls how long to wait for live service (blocks client)
//   - shadowTimeout: Controls how long shadow service can run (doesn't block client)
//   - Shadow timeout can be longer since it doesn't impact user experience
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Generate or reuse X-Request-ID
	requestID := r.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = uuid.New().String()
		r.Header.Set("X-Request-ID", requestID)
	}

	reqLogger := p.logger.With(
		slog.String("request.id", requestID),
		slog.String("request.method", r.Method),
		slog.String("request.path", r.URL.Path),
	)

	// Check if we should proxy to shadow BEFORE reading body. Sampling and
	// shadow-rule checks are size-independent, so a skip here streams live
	// without touching the body.
	if !p.shouldProxyToShadow(r) {
		p.proxyToLiveOnly(w, r, reqLogger, r.Body, r.ContentLength)
		return
	}

	// Decide how to handle the body based on the configured size limit. Oversized
	// or unbounded (chunked) requests are streamed to live without full-body
	// buffering and skip shadow; within-limit requests are buffered so they can
	// be replayed to both live and shadow.
	plan, err := p.planBody(r)
	if err != nil {
		reqLogger.Error("failed to read request body", slog.String("error", err.Error()))
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	if !plan.shadow {
		// The size limit governs shadowing only; live traffic is never rejected
		// for size. Preserve the existing skip log for observability.
		reqLogger.Debug("skipping shadow proxy",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int64("content_length", r.ContentLength),
		)
		p.proxyToLiveOnly(w, r, reqLogger, plan.liveReader, plan.liveContentLength)
		return
	}

	body := plan.body

	liveCtx, liveCancel := context.WithTimeout(r.Context(), p.liveTimeout)
	defer liveCancel()

	liveCh := make(chan responseResult, 1)
	shadowCh := make(chan responseResult, 1)

	// Launch live request
	go func() {
		// Contain any panic so it never crashes the process; unblock the
		// waiter with an error result instead of stalling until the timeout.
		defer p.recoverGoroutine(reqLogger, "live", func(v any) {
			liveCh <- responseResult{err: fmt.Errorf("panic in live goroutine: %v", v)}
		})
		start := time.Now()
		resp, err := p.forwardRequest(liveCtx, r, p.Live, bytes.NewReader(body), int64(len(body)))
		if err != nil {
			liveCh <- responseResult{err: err}
			return
		}
		b, err := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		latencyMs := time.Since(start).Milliseconds()
		if err != nil {
			liveCh <- responseResult{err: err}
			return
		}
		if closeErr != nil {
			liveCh <- responseResult{err: closeErr}
			return
		}
		liveCh <- responseResult{resp: resp, body: b, latencyMs: latencyMs}
	}()

	// Shadow uses Background() context so it's independent of client connection
	// This allows shadow requests to complete even if the client disconnects,
	// ensuring we collect complete response data for the callback
	shadowCtx, shadowCancel := context.WithTimeout(context.Background(), p.shadowTimeout)

	// Clone the request for the shadow target and add the identification header
	// so the live request is never modified. The clone carries Background()
	// context to match the shadow lifecycle.
	shadowReq := r.Clone(context.Background())
	shadowReq.Header.Set(ShadowHeader, ShadowHeaderValue)

	// Launch shadow request
	go func() {
		// Contain any panic so it never crashes the process; unblock the
		// waiter with an error result instead of stalling until the timeout.
		defer p.recoverGoroutine(reqLogger, "shadow", func(v any) {
			shadowCh <- responseResult{err: fmt.Errorf("panic in shadow goroutine: %v", v)}
		})
		start := time.Now()
		resp, err := p.forwardRequest(shadowCtx, shadowReq, p.Shadow, bytes.NewReader(body), int64(len(body)))
		if err != nil {
			shadowCh <- responseResult{err: err}
			return
		}
		b, err := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		latencyMs := time.Since(start).Milliseconds()
		if err != nil {
			shadowCh <- responseResult{err: err}
			return
		}
		if closeErr != nil {
			shadowCh <- responseResult{err: closeErr}
			return
		}
		shadowCh <- responseResult{resp: resp, body: b, latencyMs: latencyMs}
	}()

	// Wait for live first
	var liveResp responseResult
	select {
	case liveResp = <-liveCh:
	case <-liveCtx.Done():
		shadowCancel()
		reqLogger.Error("timeout waiting for live response", slog.Duration("timeout", p.liveTimeout))
		http.Error(w, "timeout waiting for live response", http.StatusGatewayTimeout)
		return
	}

	if liveResp.err != nil {
		shadowCancel()
		reqLogger.Error("live backend error", slog.String("error", liveResp.err.Error()))
		http.Error(w, "live backend error: "+liveResp.err.Error(), http.StatusBadGateway)
		return
	}

	// Write live response to client
	copyHeader(liveResp.resp.Header, w.Header())
	w.Header().Set("X-Request-ID", requestID)
	w.WriteHeader(liveResp.resp.StatusCode)
	if _, err = w.Write(liveResp.body); err != nil {
		shadowCancel()
		// Can't call http.Error after writing response body
		reqLogger.Error("failed to write response body", slog.String("error", err.Error()))
		return
	}

	// Wait for shadow and compare in background, bounded by the optional
	// callback semaphore. When the limit is reached we drop this shadow
	// comparison rather than spawn an unbounded goroutine under load — the
	// live response has already been returned to the client, so only the
	// comparison is lost. The in-flight shadow request is cancelled so it
	// does not outlive the dropped comparison.
	if !p.acquireCallbackSlot() {
		shadowCancel()
		reqLogger.Warn("callback semaphore full, dropping shadow comparison",
			slog.Int("max_concurrent_callbacks", cap(p.callbackSem)))
		return
	}

	go func(liveBody []byte) {
		// Release the callback slot and cancel the shadow context when this
		// goroutine completes. Cancelling here ensures the context is not
		// cancelled before we read from shadowCh.
		defer p.releaseCallbackSlot()
		defer shadowCancel()
		// Registered last so it runs first during unwinding: a panic in the
		// diff/redaction/API callback is contained and logged. The slot release
		// and shadow cancel above still run, so there is no slot/context leak.
		// The live response was already returned, so this never affects the client.
		defer p.recoverGoroutine(reqLogger, "callback", nil)

		select {
		case shadowResp := <-shadowCh:
			if shadowResp.err != nil {
				// Shadow is best-effort and never affects live traffic, so a
				// failed shadow request is a warning, not an error.
				reqLogger.Warn("shadow request error", slog.String("error", shadowResp.err.Error()))
				return
			}

			proxyReq := ProxyRequest{
				Method:   r.Method,
				Path:     r.URL.Path,
				RawQuery: r.URL.RawQuery,
				// Capture the shadow request headers so the identification
				// header is recorded in stored request data for reference.
				Headers: shadowReq.Header.Clone(),
				Body:    body,
			}
			live := ProxyResponse{
				StatusCode: liveResp.resp.StatusCode,
				Response:   liveResp.resp,
				Body:       liveBody,
				LatencyMs:  liveResp.latencyMs,
			}
			shadow := ProxyResponse{
				StatusCode: shadowResp.resp.StatusCode,
				Response:   shadowResp.resp,
				Body:       shadowResp.body,
				LatencyMs:  shadowResp.latencyMs,
			}

			// Invoke callback with raw responses (no diff — computed by caller if needed)
			if err := p.callbackFn(proxyReq, live, shadow); err != nil {
				reqLogger.Error("callback error", slog.String("error", err.Error()))
			}

		case <-shadowCtx.Done():
			// Shadow request timed out or was cancelled. Best-effort only, so
			// this is a warning rather than an error.
			reqLogger.Warn("shadow request timeout", slog.Duration("timeout", p.shadowTimeout))
		}
	}(liveResp.body)
}

// forwardRequest builds and sends a request to target using body as the request
// body. contentLength is the value set on the outgoing request: a non-negative
// length lets net/http send a Content-Length header, while -1 streams the body
// with chunked transfer encoding (unknown length). A nil body sends no body.
func (p *Proxy) forwardRequest(ctx context.Context, original *http.Request, target *url.URL, body io.Reader, contentLength int64) (*http.Response, error) {
	url := rewriteRequestURL(original, target)

	req, err := http.NewRequestWithContext(ctx, original.Method, url.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header = original.Header.Clone()
	req.ContentLength = contentLength
	p.logger.Debug("forwarding request",
		slog.String("method", req.Method),
		slog.String("url", req.URL.String()))
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

// defaultCallbackFn is the fallback callback used by NewProxy when none is
// supplied. It is a test/default only: every real deployment (API, standalone
// proxy, and Caddy module) overrides it via WithCallbackFn, so the per-request
// Info log below never fires in production.
func defaultCallbackFn() CallbackFunc {
	logger := slog.Default()

	return func(req ProxyRequest, live, shadow ProxyResponse) error {
		logger.Info("shadow response captured",
			slog.String("method", req.Method),
			slog.String("path", req.Path),
			slog.Int("live_status", live.StatusCode),
			slog.Int("shadow_status", shadow.StatusCode),
		)
		return nil
	}
}

// proxyToLiveOnly forwards the request to the live service only, skipping shadow.
// Used when the sampling/shadow-rule checks skip shadow, or when the body is too
// large (or unbounded) to buffer for comparison. The body is streamed via
// bodyReader with contentLength (-1 for unknown/chunked), so live traffic of any
// size passes through without full-body buffering.
func (p *Proxy) proxyToLiveOnly(w http.ResponseWriter, r *http.Request, reqLogger *slog.Logger, bodyReader io.Reader, contentLength int64) {
	ctx, cancel := context.WithTimeout(r.Context(), p.liveTimeout)
	defer cancel()

	// Forward request to live service, streaming the body straight through.
	resp, err := p.forwardRequest(ctx, r, p.Live, bodyReader, contentLength)
	if err != nil {
		reqLogger.Error("live backend error", slog.String("error", err.Error()))
		http.Error(w, "live backend error: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			reqLogger.Error("failed to close response body", slog.String("error", closeErr.Error()))
		}
	}()

	// Copy response to client
	copyHeader(resp.Header, w.Header())
	w.Header().Set("X-Request-ID", r.Header.Get("X-Request-ID"))
	w.WriteHeader(resp.StatusCode)

	// Stream body directly (don't buffer)
	if _, err := io.Copy(w, resp.Body); err != nil {
		reqLogger.Error("failed to copy response body", slog.String("error", err.Error()))
	}
}
