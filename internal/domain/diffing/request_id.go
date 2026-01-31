package diffing

import (
	"fmt"

	"github.com/google/uuid"
)

// RequestID is a value object representing a request identifier.
// It encapsulates UUID validation and ensures type safety, preventing
// accidental mixing of request IDs with gate IDs.
type RequestID struct {
	value uuid.UUID
}

// ParseRequestID parses a string into a RequestID value object.
// Returns ErrInvalidRequestID if the string is not a valid UUID.
func ParseRequestID(s string) (RequestID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return RequestID{}, fmt.Errorf("%w: %s", ErrInvalidRequestID, s)
	}
	return RequestID{value: id}, nil
}

// NewRequestID creates a new random RequestID.
func NewRequestID() RequestID {
	return RequestID{value: uuid.New()}
}

// RequestIDFromUUID creates a RequestID from an existing UUID.
// Used for database operations and testing.
func RequestIDFromUUID(id uuid.UUID) RequestID {
	return RequestID{value: id}
}

// UUID returns the underlying UUID value.
// Used for database operations and serialization.
func (r RequestID) UUID() uuid.UUID {
	return r.value
}

// String returns the string representation of the RequestID.
func (r RequestID) String() string {
	return r.value.String()
}

// IsZero returns true if the RequestID is the zero value.
func (r RequestID) IsZero() bool {
	return r.value == uuid.Nil
}
