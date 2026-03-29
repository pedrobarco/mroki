package traffictesting

import (
	"fmt"
	"strings"
)

// GateName represents a validated gate name (value object).
// Names must be non-empty, trimmed, and at most 255 characters.
type GateName struct {
	value string
}

// ParseGateName creates a GateName from a raw string.
func ParseGateName(s string) (GateName, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return GateName{}, fmt.Errorf("%w: name must not be empty", ErrInvalidGateName)
	}
	if len(trimmed) > 255 {
		return GateName{}, fmt.Errorf("%w: name must be at most 255 characters", ErrInvalidGateName)
	}
	return GateName{value: trimmed}, nil
}

// String returns the gate name value.
func (n GateName) String() string {
	return n.value
}

// IsZero returns true if the name is empty (zero value).
func (n GateName) IsZero() bool {
	return n.value == ""
}

// Equals checks value equality.
func (n GateName) Equals(other GateName) bool {
	return n.value == other.value
}
