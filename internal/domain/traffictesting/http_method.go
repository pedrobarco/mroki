package traffictesting

import (
	"fmt"
	"strings"
)

// HTTPMethod represents a valid HTTP method
type HTTPMethod struct {
	value string
}

var validHTTPMethods = map[string]bool{
	"GET":     true,
	"POST":    true,
	"PUT":     true,
	"DELETE":  true,
	"PATCH":   true,
	"HEAD":    true,
	"OPTIONS": true,
}

// NewHTTPMethod creates a validated HTTPMethod value object
func NewHTTPMethod(method string) (HTTPMethod, error) {
	normalized := strings.ToUpper(strings.TrimSpace(method))

	if normalized == "" {
		return HTTPMethod{}, fmt.Errorf("HTTP method cannot be empty")
	}

	if !validHTTPMethods[normalized] {
		return HTTPMethod{}, fmt.Errorf(
			"invalid HTTP method: '%s' (valid: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS)",
			method,
		)
	}

	return HTTPMethod{value: normalized}, nil
}

// Factory methods for common HTTP methods
func GET() HTTPMethod     { return HTTPMethod{value: "GET"} }
func POST() HTTPMethod    { return HTTPMethod{value: "POST"} }
func PUT() HTTPMethod     { return HTTPMethod{value: "PUT"} }
func DELETE() HTTPMethod  { return HTTPMethod{value: "DELETE"} }
func PATCH() HTTPMethod   { return HTTPMethod{value: "PATCH"} }
func HEAD() HTTPMethod    { return HTTPMethod{value: "HEAD"} }
func OPTIONS() HTTPMethod { return HTTPMethod{value: "OPTIONS"} }

// String returns the HTTP method (uppercase)
func (m HTTPMethod) String() string {
	return m.value
}

// Equals checks value equality
func (m HTTPMethod) Equals(other HTTPMethod) bool {
	return m.value == other.value
}

// Predicates for common checks
func (m HTTPMethod) IsGET() bool     { return m.value == "GET" }
func (m HTTPMethod) IsPOST() bool    { return m.value == "POST" }
func (m HTTPMethod) IsPUT() bool     { return m.value == "PUT" }
func (m HTTPMethod) IsDELETE() bool  { return m.value == "DELETE" }
func (m HTTPMethod) IsPATCH() bool   { return m.value == "PATCH" }
func (m HTTPMethod) IsHEAD() bool    { return m.value == "HEAD" }
func (m HTTPMethod) IsOPTIONS() bool { return m.value == "OPTIONS" }
