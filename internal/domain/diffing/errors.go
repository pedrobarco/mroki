package diffing

import "errors"

var (
	// ErrGateNotFound is returned when a gate cannot be found by ID
	ErrGateNotFound = errors.New("gate not found")

	// ErrRequestNotFound is returned when a request cannot be found by ID
	ErrRequestNotFound = errors.New("request not found")

	// ErrInvalidGateID indicates a string cannot be parsed as a gate ID.
	ErrInvalidGateID = errors.New("invalid gate ID format")

	// ErrInvalidRequestID indicates a string cannot be parsed as a request ID.
	ErrInvalidRequestID = errors.New("invalid request ID format")

	// ErrInvalidAgentID indicates a string cannot be parsed as an agent ID.
	ErrInvalidAgentID = errors.New("invalid agent ID format")

	// ErrInvalidGateURL indicates a URL is not valid for gate usage.
	// URLs must use http or https schemes.
	ErrInvalidGateURL = errors.New("invalid gate URL")

	// ErrInvalidPagination indicates invalid pagination parameters
	ErrInvalidPagination = errors.New("invalid pagination parameters")
)
