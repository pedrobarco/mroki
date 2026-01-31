package diffing

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// AgentID is a value object representing a unique identifier for an agent.
// Format: {hostname}-{8-hex-chars}
// Example: web-server-a1b2c3d4, api-prod-550e8400
//
// This hybrid format provides both human readability (via hostname) and
// uniqueness (via UUID segment). The hostname is limited to 50 characters
// and may only contain alphanumeric characters and hyphens.
type AgentID struct {
	value string
}

var agentIDPattern = regexp.MustCompile(`^[a-zA-Z0-9-]{1,50}-[0-9a-fA-F]{8}$`)

// ParseAgentID parses a string into an AgentID.
// Returns ErrInvalidAgentID if the string doesn't match the required format.
//
// Valid format: {hostname}-{8-hex-chars}
// - hostname: 1-50 alphanumeric characters or hyphens
// - 8-hex-chars: exactly 8 hexadecimal characters (lowercase)
func ParseAgentID(s string) (AgentID, error) {
	if s == "" {
		return AgentID{}, fmt.Errorf("%w: empty string", ErrInvalidAgentID)
	}

	if !agentIDPattern.MatchString(s) {
		return AgentID{}, fmt.Errorf("%w: must match format {hostname}-{8-hex-chars}, got %q", ErrInvalidAgentID, s)
	}

	return AgentID{value: s}, nil
}

// NewAgentID creates a new random AgentID with "agent" as the hostname prefix.
// For custom hostnames, use NewAgentIDWithHostname.
func NewAgentID() AgentID {
	return NewAgentIDWithHostname("agent")
}

// NewAgentIDWithHostname creates a new AgentID with a custom hostname.
// The hostname will be sanitized to match the allowed format.
func NewAgentIDWithHostname(hostname string) AgentID {
	// Sanitize hostname: keep only alphanumeric and hyphens
	hostname = sanitizeHostname(hostname)

	// Truncate if too long
	if len(hostname) > 50 {
		hostname = hostname[:50]
	}

	// Generate UUID and take first segment (8 hex chars)
	id := uuid.New()
	shortID := strings.Split(id.String(), "-")[0]

	return AgentID{value: fmt.Sprintf("%s-%s", hostname, shortID)}
}

// sanitizeHostname replaces invalid characters with hyphens
func sanitizeHostname(hostname string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return '-'
	}, hostname)
}

// String returns the string representation of the agent ID.
func (a AgentID) String() string {
	return a.value
}

// IsZero returns true if the AgentID is the zero value.
func (a AgentID) IsZero() bool {
	return a.value == ""
}
