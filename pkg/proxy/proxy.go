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

var defaultTimeout = 5 * time.Second

type CallbackFunc func(live, shadow ProxyResponse) error

type Proxy struct {
	Live    *url.URL
	Shadow  *url.URL
	Timeout time.Duration

	callbackFn CallbackFunc
	logger     *slog.Logger
	client     *http.Client
}

var (
	_ http.Handler = (*Proxy)(nil)
)

func NewProxy(live, shadow *url.URL) *Proxy {
	return &Proxy{
		Live:       live,
		Shadow:     shadow,
		Timeout:    defaultTimeout,
		callbackFn: defaultCallbackFn(),
		logger:     slog.Default(),
		client:     http.DefaultClient,
	}
}

type ProxyResponse struct {
	Response *http.Response
	Body     []byte
}

type responseResult struct {
	resp *http.Response
	body []byte
	err  error
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	liveCtx, liveCancel := context.WithTimeout(r.Context(), p.Timeout)
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
		if err := resp.Body.Close(); err != nil {
			liveCh <- responseResult{err: err}
		}
		liveCh <- responseResult{resp: resp, body: b, err: err}
	}()

	shadowCtx, shadowCancel := context.WithTimeout(context.Background(), p.Timeout)

	// Launch shadow request
	go func() {
		defer shadowCancel()
		resp, err := p.forwardRequest(shadowCtx, r, p.Shadow, body)
		if err != nil {
			shadowCh <- responseResult{err: err}
			return
		}
		b, err := io.ReadAll(resp.Body)
		if err := resp.Body.Close(); err != nil {
			shadowCh <- responseResult{err: err}
		}
		shadowCh <- responseResult{resp: resp, body: b, err: err}
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
		http.Error(w, "Failed to write live response", http.StatusInternalServerError)
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

			live := ProxyResponse{
				Response: liveResp.resp,
				Body:     liveBody,
			}
			shadow := ProxyResponse{
				Response: shadowResp.resp,
				Body:     shadowResp.body,
			}

			if err := p.callbackFn(live, shadow); err != nil {
				p.logger.Error("callback error", "error", err)
			}

		case <-time.After(p.Timeout):
			p.logger.Error("shadow request timeout", "timeout", p.Timeout)
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
	p.logger.Debug("Forwarding request",
		"method", req.Method,
		"url", req.URL.String())
	return p.client.Do(req)
}

func copyHeader(src, dst http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Set(k, v)
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
		return resp.Header.Get("Content-Type") == "application/json"
	}

	logger := slog.Default()
	differ := NewProxyResponseDiffer()

	return func(live, shadow ProxyResponse) error {
		// Only diff if both responses are JSON
		if !isJSONContent(live.Response) || !isJSONContent(shadow.Response) {
			return nil
		}

		res, err := differ.Diff(live, shadow)
		if err != nil {
			logger.Error("Failed to diff responses", "error", err)
		}

		if len(res) > 0 {
			logger.Debug("Response diff detected",
				"live_status", live.Response.StatusCode,
				"shadow_status", shadow.Response.StatusCode,
			)
		}

		return nil
	}
}
