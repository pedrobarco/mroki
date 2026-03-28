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
	agentID string,
) *CapturedRequest {
	now := time.Now()

	return &CapturedRequest{
		AgentID:   agentID,
		Method:    req.Method,
		Path:      req.Path,
		Headers:   req.Headers,
		Body:      base64.StdEncoding.EncodeToString(req.Body),
		CreatedAt: now,

		Responses: []CapturedResponse{
			{
				Type:       "live",
				StatusCode: live.StatusCode,
				Headers:    live.Response.Header,
				Body:       base64.StdEncoding.EncodeToString(live.Body),
				CreatedAt:  now,
			},
			{
				Type:       "shadow",
				StatusCode: shadow.StatusCode,
				Headers:    shadow.Response.Header,
				Body:       base64.StdEncoding.EncodeToString(shadow.Body),
				CreatedAt:  now,
			},
		},
	}
}
