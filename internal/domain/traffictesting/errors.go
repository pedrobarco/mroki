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

	// Validation errors
	ErrInvalidStatusCode = errors.New("invalid HTTP status code")
	ErrInvalidPath       = errors.New("invalid path")

	// Filtering and sorting errors
	ErrInvalidFilters     = errors.New("invalid request filters")
	ErrInvalidSort        = errors.New("invalid request sort")
	ErrInvalidGateFilters = errors.New("invalid gate filters")
	ErrInvalidGateSort    = errors.New("invalid gate sort")
	ErrInvalidSortField = errors.New("invalid sort field")
	ErrInvalidSortOrder = errors.New("invalid sort order")

	// Pagination errors
	ErrInvalidPagination = errors.New("invalid pagination parameters")
)
