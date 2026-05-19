package traffictesting

import "net/http"

// Headers wraps http.Header for domain layer type safety
type Headers http.Header

// NewHeaders creates a Headers wrapper from http.Header
func NewHeaders(h http.Header) Headers {
	if h == nil {
		return make(Headers)
	}
	return Headers(h)
}

// HTTPHeader returns the underlying http.Header
func (h Headers) HTTPHeader() http.Header {
	return http.Header(h)
}

// RedactedValue is the replacement value for scrubbed headers.
const RedactedValue = "[REDACTED]"

// Scrub replaces the values of the given header names with RedactedValue.
// Matching is case-insensitive (uses http.CanonicalHeaderKey).
// Returns a new Headers instance; the original is not modified.
func (h Headers) Scrub(names []string) Headers {
	out := make(http.Header, len(h))
	for k, vv := range http.Header(h) {
		dst := make([]string, len(vv))
		copy(dst, vv)
		out[k] = dst
	}
	for _, name := range names {
		canonical := http.CanonicalHeaderKey(name)
		if _, ok := out[canonical]; ok {
			out[canonical] = []string{RedactedValue}
		}
	}
	return Headers(out)
}
