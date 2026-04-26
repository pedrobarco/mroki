package transport

import "net/http"

// authRoundTripper sets Authorization and Content-Type headers on every request.
type authRoundTripper struct {
	next   http.RoundTripper
	apiKey string
}

// NewAuthRoundTripper returns a RoundTripper that injects Bearer auth and
// Content-Type: application/json headers before delegating to next.
func NewAuthRoundTripper(next http.RoundTripper, apiKey string) http.RoundTripper {
	return &authRoundTripper{next: next, apiKey: apiKey}
}

func (rt *authRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context()) // don't mutate caller's request
	req.Header.Set("Authorization", "Bearer "+rt.apiKey)
	req.Header.Set("Content-Type", "application/json")
	return rt.next.RoundTrip(req)
}
