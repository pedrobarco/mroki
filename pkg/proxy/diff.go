package proxy

import (
	"encoding/json"
	"fmt"

	"github.com/pedrobarco/mroki/pkg/diff"
)

type proxyResponseDiffer struct {
	opts []diff.Option
}

// Compile-time check that proxyResponseDiffer implements diff.Differ[ProxyResponse]
var _ diff.Differ[ProxyResponse] = (*proxyResponseDiffer)(nil)

func NewProxyResponseDiffer(opts ...diff.Option) *proxyResponseDiffer {
	return &proxyResponseDiffer{opts: opts}
}

// Diff compares two proxy responses using configured diff options
func (p *proxyResponseDiffer) Diff(a, b ProxyResponse) (string, error) {
	ah, err := json.Marshal(a.Response.Header)
	if err != nil {
		return "", fmt.Errorf("failed to marshal live response header: %w", err)
	}

	bh, err := json.Marshal(b.Response.Header)
	if err != nil {
		return "", fmt.Errorf("failed to marshal shadow response header: %w", err)
	}

	live := jsonString(a.StatusCode, ah, a.Body)
	shadow := jsonString(b.StatusCode, bh, b.Body)

	// Use configured options from struct
	return diff.JSON(live, shadow, p.opts...)
}

func jsonString(status int, headers, body []byte) string {
	return fmt.Sprintf(`{"statusCode": %d, "headers": %s, "body": %s}`, status, headers, body)
}
