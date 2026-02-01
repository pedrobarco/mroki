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

	// SQL injection prevention
	if err := validatePathPatternSecurity(trimmed); err != nil {
		return PathPattern{}, err
	}

	return PathPattern{value: trimmed}, nil
}

// validatePathPatternSecurity checks for SQL injection patterns
// Note: We allow single '*' for glob wildcards
func validatePathPatternSecurity(pattern string) error {
	// Check for SQL statement terminators and comments
	if strings.Contains(pattern, ";") {
		return fmt.Errorf("path pattern contains invalid characters or SQL keywords")
	}
	if strings.Contains(pattern, "--") {
		return fmt.Errorf("path pattern contains invalid characters or SQL keywords")
	}

	// Check for SQL string/identifier delimiters
	if strings.Contains(pattern, "'") || strings.Contains(pattern, "\"") {
		return fmt.Errorf("path pattern contains invalid characters or SQL keywords")
	}

	// Check for SQL keywords
	dangerousKeywords := []string{
		"SELECT",
		"DROP",
		"INSERT",
		"UPDATE",
		"DELETE",
	}

	upperPattern := strings.ToUpper(pattern)
	for _, keyword := range dangerousKeywords {
		if strings.Contains(upperPattern, keyword) {
			return fmt.Errorf("path pattern contains invalid characters or SQL keywords")
		}
	}

	return nil
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
