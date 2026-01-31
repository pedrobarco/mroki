package diffing

import "errors"

var (
	// ErrGateNotFound is returned when a gate cannot be found by ID
	ErrGateNotFound = errors.New("gate not found")

	// ErrRequestNotFound is returned when a request cannot be found by ID
	ErrRequestNotFound = errors.New("request not found")
)
