package traffictesting

import (
	"fmt"
	"strings"
)

// PathPattern represents a glob-style path pattern for filtering
// Supports wildcard '*' character (e.g., "/api/users/*")
type PathPattern struct {
	value string
}

const maxPathPatternLength = 500

// NewPathPattern creates a validated PathPattern value object
// Empty string is valid (means no filtering)
func NewPathPattern(pattern string) (PathPattern, error) {
	trimmed := strings.TrimSpace(pattern)

	// Empty is valid (no filter)
	if trimmed == "" {
		return PathPattern{value: ""}, nil
	}

	// Length validation
	if len(trimmed) > maxPathPatternLength {
		return PathPattern{}, fmt.Errorf(
			"path pattern exceeds maximum length of %d characters",
			maxPathPatternLength,
		)
	}

	// Validate the pattern is a structurally valid URL path.
	if err := validatePathPatternChars(trimmed); err != nil {
		return PathPattern{}, err
	}

	return PathPattern{value: trimmed}, nil
}

// validatePathPatternChars ensures the pattern only contains characters that
// are legal in a URL path (RFC 3986 path grammar) plus the '*' wildcard, which
// is itself an RFC 3986 sub-delim and so needs no special handling.
//
// SQL safety does not depend on this check: the repository binds the pattern as
// a query parameter and escapes LIKE metacharacters. This validation only
// rejects characters that cannot appear unencoded in a URL path, such as
// spaces, quotes, angle brackets, control characters, and URL delimiters.
func validatePathPatternChars(pattern string) error {
	for i := 0; i < len(pattern); i++ {
		if !isPathChar(pattern[i]) {
			return fmt.Errorf(
				"path pattern contains invalid character %q for a URL path",
				rune(pattern[i]),
			)
		}
	}
	return nil
}

// isPathChar reports whether b is allowed in a URL path per RFC 3986
// (pchar / "/"): unreserved characters, sub-delims, ":", "@", "/", and "%" for
// percent-encoding.
func isPathChar(b byte) bool {
	switch {
	case b >= 'A' && b <= 'Z',
		b >= 'a' && b <= 'z',
		b >= '0' && b <= '9':
		return true
	}
	switch b {
	case '-', '.', '_', '~', // unreserved
		'!', '$', '&', '\'', '(', ')', '*', '+', ',', ';', '=', // sub-delims
		':', '@', // pchar
		'/', // path separator
		'%': // percent-encoding
		return true
	}
	return false
}

// EmptyPathPattern returns a pattern with no filtering
func EmptyPathPattern() PathPattern {
	return PathPattern{value: ""}
}

// String returns the pattern value
func (p PathPattern) String() string {
	return p.value
}

// IsEmpty returns true if no pattern is set
func (p PathPattern) IsEmpty() bool {
	return p.value == ""
}

// Equals checks value equality
func (p PathPattern) Equals(other PathPattern) bool {
	return p.value == other.value
}
