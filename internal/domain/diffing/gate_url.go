package diffing

import (
	"fmt"
	"net/url"
)

// GateURL is a value object representing a validated gate URL.
// It ensures URLs use http or https schemes.
type GateURL struct {
	value *url.URL
}

// ParseGateURL parses and validates a URL string for use as a gate URL.
// Returns ErrInvalidGateURL if:
// - The URL cannot be parsed
// - The URL scheme is not http or https
func ParseGateURL(s string) (GateURL, error) {
	u, err := url.Parse(s)
	if err != nil {
		return GateURL{}, fmt.Errorf("%w: failed to parse: %v", ErrInvalidGateURL, err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return GateURL{}, fmt.Errorf("%w: scheme must be http or https, got %q", ErrInvalidGateURL, u.Scheme)
	}

	return GateURL{value: u}, nil
}

// GateURLFromURL creates a GateURL from an existing *url.URL without validation.
// Used for database operations and testing. Caller must ensure URL is valid.
func GateURLFromURL(u *url.URL) GateURL {
	return GateURL{value: u}
}

// URL returns the underlying *url.URL value.
func (g GateURL) URL() *url.URL {
	return g.value
}

// String returns the string representation of the URL.
func (g GateURL) String() string {
	if g.value == nil {
		return ""
	}
	return g.value.String()
}
