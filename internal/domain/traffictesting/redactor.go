package traffictesting

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/pedrobarco/mroki/pkg/jsontree"
)

// Redactor redacts sensitive fields from headers and JSON bodies.
// Header and body fields are partitioned at construction time (once per gate).
type Redactor struct {
	headerFields []string // plain header names, e.g. "Authorization"
	bodyFields   []string // dot-separated body paths, e.g. "user.password"
}

// NewRedactor creates a Redactor by partitioning fields into header and body groups.
// Fields prefixed with "headers." are treated as header fields (prefix stripped).
// Fields prefixed with "body." are treated as body fields (prefix stripped).
// Fields without a recognized prefix are ignored.
func NewRedactor(fields []string) *Redactor {
	var headers, bodies []string
	for _, f := range fields {
		switch {
		case strings.HasPrefix(f, "headers."):
			if name := f[len("headers."):]; name != "" {
				headers = append(headers, name)
			}
		case strings.HasPrefix(f, "body."):
			if path := f[len("body."):]; path != "" {
				bodies = append(bodies, path)
			}
		}
	}
	return &Redactor{
		headerFields: headers,
		bodyFields:   bodies,
	}
}

// RedactResult holds the redacted headers and body.
type RedactResult struct {
	Headers    http.Header
	Body       []byte        // marshaled redacted body (for storage)
	BodyParsed jsontree.Tree // in-memory JSON tree post-redaction (for diff optimization); nil if body is not JSON
}

// Redact replaces sensitive values in headers and body with [REDACTED].
// Headers: map lookup per field — O(h).
// Body: json.Unmarshal → walk tree → set [REDACTED] → json.Marshal — O(n).
// Empty/nil body: skip body redaction, no error.
// Non-JSON body: returned as-is (non-fatal).
// Missing paths: silently skipped.
func (r *Redactor) Redact(headers http.Header, body []byte) (RedactResult, error) {
	// Redact headers.
	// We iterate map keys directly because http.Header.Get/Set/Del all
	// canonicalize the lookup key, but non-canonical map keys (e.g. from
	// JSON deserialization) would be missed, leaving sensitive values intact.
	if headers != nil {
		headers = headers.Clone()
		for _, name := range r.headerFields {
			lower := strings.ToLower(name)
			for k := range headers {
				if strings.ToLower(k) == lower {
					headers[k] = []string{RedactedValue}
				}
			}
		}
	}

	// Redact body
	redactedBody, bodyParsed, err := r.redactBody(body)
	if err != nil {
		if errors.Is(err, errNotJSON) {
			return RedactResult{Headers: headers, Body: body, BodyParsed: nil}, nil
		}
		return RedactResult{}, err
	}

	return RedactResult{Headers: headers, Body: redactedBody, BodyParsed: bodyParsed}, nil
}

// errNotJSON is a sentinel indicating the body is not valid JSON (passthrough).
var errNotJSON = errors.New("body is not valid JSON")

// redactBody unmarshals, walks, redacts, and re-marshals the JSON body.
// Returns (marshaledBytes, parsedTree, error).
//   - Empty body: returns (body, nil, nil)
//   - Non-JSON body: returns (nil, nil, errNotJSON)
//   - Valid JSON, no fields to redact: returns (original bytes, parsed tree, nil)
//   - Valid JSON, fields redacted: returns (re-marshaled bytes, mutated tree, nil)
func (r *Redactor) redactBody(body []byte) ([]byte, jsontree.Tree, error) {
	if len(body) == 0 {
		return body, nil, nil
	}

	var root jsontree.Tree
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, nil, errNotJSON
	}

	if len(r.bodyFields) == 0 {
		// No fields to redact, but body is valid JSON — retain the parsed tree.
		return body, root, nil
	}

	for _, path := range r.bodyFields {
		jsontree.WalkPath(root, path, func(parent map[string]any, key string) {
			parent[key] = RedactedValue
		})
	}

	out, err := json.Marshal(root)
	if err != nil {
		// Marshal of an unmarshal-produced tree should never fail, but if it
		// does we must not fall back to returning the original un-redacted body.
		return nil, nil, fmt.Errorf("redact body: marshal failed: %w", err)
	}
	return out, root, nil
}
