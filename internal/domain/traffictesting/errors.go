package traffictesting

import "errors"

// Domain errors for traffictesting
var (
	// Gate errors
	ErrGateNotFound   = errors.New("gate not found")
	ErrInvalidGateID  = errors.New("invalid gate ID")
	ErrInvalidGateURL = errors.New("invalid gate URL")

	// Request errors
	ErrRequestNotFound  = errors.New("request not found")
	ErrInvalidRequestID = errors.New("invalid request ID")
	ErrInvalidAgentID   = errors.New("invalid agent ID")

	// Pagination errors
	ErrInvalidPagination = errors.New("invalid pagination parameters")
)
