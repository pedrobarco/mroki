package diffing

import (
	"fmt"

	"github.com/google/uuid"
)

// GateID is a value object representing a gate identifier.
// It encapsulates UUID validation and ensures type safety, preventing
// accidental mixing of gate IDs with request IDs.
type GateID struct {
	value uuid.UUID
}

// ParseGateID parses a string into a GateID value object.
// Returns ErrInvalidGateID if the string is not a valid UUID.
func ParseGateID(s string) (GateID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return GateID{}, fmt.Errorf("%w: %s", ErrInvalidGateID, s)
	}
	return GateID{value: id}, nil
}

// NewGateID creates a new random GateID.
func NewGateID() GateID {
	return GateID{value: uuid.New()}
}

// GateIDFromUUID creates a GateID from an existing UUID.
// Used for database operations and testing.
func GateIDFromUUID(id uuid.UUID) GateID {
	return GateID{value: id}
}

// UUID returns the underlying UUID value.
// Used for database operations and serialization.
func (g GateID) UUID() uuid.UUID {
	return g.value
}

// String returns the string representation of the GateID.
func (g GateID) String() string {
	return g.value.String()
}

// IsZero returns true if the GateID is the zero value.
func (g GateID) IsZero() bool {
	return g.value == uuid.Nil
}
