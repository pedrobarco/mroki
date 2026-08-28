package proxy

import (
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

// Diff compares two proxy responses by building a synthetic envelope
// ({"statusCode", "headers", "body"}) for each and comparing them with
// diff.Parsed. Bodies are embedded per Content-Type via diff.EmbedBody.
//
// This is the standalone-proxy fallback used when no redactor is configured; the
// redacted path goes through the ResponseComparer service and shares the same
// classification.
func (p *proxyResponseDiffer) Diff(a, b ProxyResponse) ([]diff.PatchOp, error) {
	live := diff.BuildEnvelope(
		a.StatusCode,
		a.Response.Header,
		diff.EmbedBody(a.Response.Header.Get("Content-Type"), a.Body, nil),
	)
	shadow := diff.BuildEnvelope(
		b.StatusCode,
		b.Response.Header,
		diff.EmbedBody(b.Response.Header.Get("Content-Type"), b.Body, nil),
	)

	return diff.Parsed(live, shadow, p.opts...)
}
