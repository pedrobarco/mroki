package services

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/pedrobarco/mroki/pkg/diff"
)

// ResponseData holds the raw inputs for one side of the comparison.
type ResponseData struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

// CompareResult holds all redacted results and the computed diff ops.
type CompareResult struct {
	Request traffictesting.RedactResult
	Live    traffictesting.RedactResult
	Shadow  traffictesting.RedactResult
	Ops     []diff.PatchOp
}

// ResponseComparer encapsulates the redact + envelope + diff pipeline.
type ResponseComparer struct {
	redactor *traffictesting.Redactor
	diffOpts []diff.Option
}

// NewResponseComparer creates a ResponseComparer with the given redactor and diff options.
func NewResponseComparer(redactor *traffictesting.Redactor, diffOpts []diff.Option) *ResponseComparer {
	return &ResponseComparer{
		redactor: redactor,
		diffOpts: diffOpts,
	}
}

// ErrNilRedactor is returned when Compare is called with a nil redactor.
var ErrNilRedactor = errors.New("redactor must not be nil")

// Compare redacts all three inputs, builds envelopes from live and shadow,
// and computes the diff between them.
//
// Redaction errors are fatal (returned immediately).
// Diff errors are non-fatal (ops defaults to empty slice).
func (c *ResponseComparer) Compare(req, live, shadow ResponseData) (*CompareResult, error) {
	if c.redactor == nil {
		return nil, ErrNilRedactor
	}

	// 1. Redact all three inputs.
	reqResult, err := c.redactor.Redact(req.Headers, req.Body)
	if err != nil {
		return nil, fmt.Errorf("request redaction: %w", err)
	}

	liveResult, err := c.redactor.Redact(live.Headers, live.Body)
	if err != nil {
		return nil, fmt.Errorf("live response redaction: %w", err)
	}

	shadowResult, err := c.redactor.Redact(shadow.Headers, shadow.Body)
	if err != nil {
		return nil, fmt.Errorf("shadow response redaction: %w", err)
	}

	// 2. Build envelopes from live and shadow using redacted headers.
	liveEnvelope := diff.BuildEnvelope(live.StatusCode, liveResult.Headers, liveResult.BodyParsed)
	shadowEnvelope := diff.BuildEnvelope(shadow.StatusCode, shadowResult.Headers, shadowResult.BodyParsed)

	// 3. Compute diff — errors are non-fatal.
	ops, err := diff.Parsed(liveEnvelope, shadowEnvelope, c.diffOpts...)
	if err != nil {
		ops = []diff.PatchOp{}
	}

	return &CompareResult{
		Request: reqResult,
		Live:    liveResult,
		Shadow:  shadowResult,
		Ops:     ops,
	}, nil
}
