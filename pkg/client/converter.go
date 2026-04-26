package client

import (
	"encoding/base64"
	"time"

	"github.com/pedrobarco/mroki/pkg/proxy"
)

// ConvertProxyToCapture converts proxy types to API types.
// Diff content is omitted — diff computation is handled server-side by mroki-api.
func ConvertProxyToCapture(
	req proxy.ProxyRequest,
	live, shadow proxy.ProxyResponse,
) *CapturedRequest {
	now := time.Now()

	return &CapturedRequest{
		ID:        req.Headers.Get("X-Request-ID"),
		Method:    req.Method,
		Path:      req.Path,
		Headers:   req.Headers,
		Body:      base64.StdEncoding.EncodeToString(req.Body),
		CreatedAt: now,

		LiveResponse: CapturedResponse{
			StatusCode: live.StatusCode,
			Headers:    live.Response.Header,
			Body:       base64.StdEncoding.EncodeToString(live.Body),
			LatencyMs:  live.LatencyMs,
			CreatedAt:  now,
		},
		ShadowResponse: CapturedResponse{
			StatusCode: shadow.StatusCode,
			Headers:    shadow.Response.Header,
			Body:       base64.StdEncoding.EncodeToString(shadow.Body),
			LatencyMs:  shadow.LatencyMs,
			CreatedAt:  now,
		},
	}
}
