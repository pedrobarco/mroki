package traffictesting

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
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
	Headers http.Header
	Body    []byte
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
	redactedBody, err := r.redactBody(body)
	if err != nil {
		if errors.Is(err, errNotJSON) {
			return RedactResult{Headers: headers, Body: body}, nil
		}
		return RedactResult{}, err
	}

	return RedactResult{Headers: headers, Body: redactedBody}, nil
}

// errNotJSON is a sentinel indicating the body is not valid JSON (passthrough).
var errNotJSON = errors.New("body is not valid JSON")

// redactBody unmarshals, walks, redacts, and re-marshals the JSON body.
func (r *Redactor) redactBody(body []byte) ([]byte, error) {
	if len(body) == 0 || len(r.bodyFields) == 0 {
		return body, nil
	}

	var root interface{}
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, errNotJSON
	}

	for _, path := range r.bodyFields {
		segments := strings.Split(path, ".")
		setRedacted(root, segments)
	}

	out, err := json.Marshal(root)
	if err != nil {
		// Marshal of an unmarshal-produced tree should never fail, but if it
		// does we must not fall back to returning the original un-redacted body.
		return nil, fmt.Errorf("redact body: marshal failed: %w", err)
	}
	return out, nil
}

// setRedacted walks the in-memory JSON tree and replaces the target value.
// Supports nested objects and arrays (via "#" wildcard segment).
func setRedacted(node interface{}, segments []string) {
	if len(segments) == 0 || node == nil {
		return
	}

	key := segments[0]
	rest := segments[1:]

	switch v := node.(type) {
	case map[string]interface{}:
		if len(rest) == 0 {
			// Leaf: redact if key exists
			if _, ok := v[key]; ok {
				v[key] = RedactedValue
			}
			return
		}
		// Recurse into child
		if child, ok := v[key]; ok {
			setRedacted(child, rest)
		}

	case []interface{}:
		// "#" wildcard: apply remaining path to each array element
		if key == "#" {
			for _, elem := range v {
				if len(rest) == 0 {
					// Can't redact array elements directly (no parent key)
					continue
				}
				setRedacted(elem, rest)
			}
		}
	}
}
