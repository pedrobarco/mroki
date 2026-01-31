package diffing

import (
	"fmt"

	"github.com/google/uuid"
)

// AgentID is a value object representing a unique identifier for an agent.
// It wraps a UUID to provide type safety and prevent mixing with other ID types.
type AgentID struct {
	value uuid.UUID
}

// ParseAgentID parses a string into an AgentID.
// Returns ErrInvalidAgentID if the string is not a valid UUID.
func ParseAgentID(s string) (AgentID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return AgentID{}, fmt.Errorf("%w: %v", ErrInvalidAgentID, err)
	}
	return AgentID{value: id}, nil
}

// NewAgentID creates a new random AgentID.
func NewAgentID() AgentID {
	return AgentID{value: uuid.New()}
}

// AgentIDFromUUID creates an AgentID from an existing UUID.
// Used for database operations and testing. No validation is performed.
func AgentIDFromUUID(id uuid.UUID) AgentID {
	return AgentID{value: id}
}

// UUID returns the underlying UUID value.
func (a AgentID) UUID() uuid.UUID {
	return a.value
}

// String returns the string representation of the agent ID.
func (a AgentID) String() string {
	return a.value.String()
}

// IsZero returns true if the AgentID is the zero value.
func (a AgentID) IsZero() bool {
	return a.value == uuid.Nil
}
