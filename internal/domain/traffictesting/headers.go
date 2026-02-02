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
