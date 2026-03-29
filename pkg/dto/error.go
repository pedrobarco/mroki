package dto

import (
	"errors"
	"fmt"
	"net/http"
)

const unknownErrorMessage = "An unknown error occurred. Please try again later."

// APIError represents an RFC 7807 Problem Detail error response.
// It is serialized directly as the error response body.
type APIError struct {
	Type     string `json:"type"`               // URI reference identifying the error type
	Title    string `json:"title"`              // Short, human-readable summary
	Status   int    `json:"status"`             // HTTP status code
	Detail   string `json:"detail,omitempty"`   // Human-readable explanation
	Instance string `json:"instance,omitempty"` // URI reference to error occurrence
	Err      error  `json:"-"`                  // Internal error (not serialized)
}

// Error implements the error interface
func (a *APIError) Error() string {
	if a.Err != nil {
		return a.Err.Error()
	}
	return a.Detail
}

// Generic HTTP error type constants (relative paths)
const (
	// Request body errors
	ErrorTypeInvalidRequestBody = "/errors/invalid-request-body" // Malformed JSON or validation failures

	// Parameter errors (by source)
	ErrorTypeMissingPathParam  = "/errors/missing-path-param"  // Required path parameter missing
	ErrorTypeMissingQueryParam = "/errors/missing-query-param" // Required query parameter missing
	ErrorTypeMissingHeader     = "/errors/missing-header"      // Required header missing
	ErrorTypeMissingBodyField  = "/errors/missing-body-field"  // Required body field missing
	ErrorTypeInvalidQueryParam = "/errors/invalid-query-param" // Query parameter invalid

	// Authentication errors
	ErrorTypeUnauthorized = "/errors/unauthorized" // Missing or invalid credentials

	// Rate limiting errors
	ErrorTypeRateLimitExceeded = "/errors/rate-limit-exceeded" // Too many requests

	// Resource errors
	ErrorTypeNotFound = "/errors/not-found" // Resource doesn't exist
	ErrorTypeConflict = "/errors/conflict"  // Resource already exists

	// Server errors
	ErrorTypeInternalError = "/errors/internal-error" // Server error
)

// Static authentication errors
var (
	ErrMissingAuthHeader = &APIError{
		Type:   ErrorTypeUnauthorized,
		Title:  "Missing Authorization Header",
		Status: http.StatusUnauthorized,
		Detail: "Authorization header is required",
		Err:    errors.New("missing authorization header"),
	}

	ErrInvalidAuthFormat = &APIError{
		Type:   ErrorTypeUnauthorized,
		Title:  "Invalid Authorization Format",
		Status: http.StatusUnauthorized,
		Detail: "Authorization header must use format: Bearer <token>",
		Err:    errors.New("invalid authorization format"),
	}

	ErrInvalidAPIKey = &APIError{
		Type:   ErrorTypeUnauthorized,
		Title:  "Invalid API Key",
		Status: http.StatusUnauthorized,
		Detail: "The provided API key is not valid",
		Err:    errors.New("invalid api key"),
	}

	ErrRateLimitExceeded = &APIError{
		Type:   ErrorTypeRateLimitExceeded,
		Title:  "Rate Limit Exceeded",
		Status: http.StatusTooManyRequests,
		Detail: "Too many requests. Please slow down and try again later.",
		Err:    errors.New("rate limit exceeded"),
	}
)

// NewError creates a new APIError with RFC 7807 fields.
func NewError(status int, errorType, title, detail string, err error) *APIError {
	return &APIError{
		Type:   errorType,
		Title:  title,
		Status: status,
		Detail: detail,
		Err:    err,
	}
}

// Generic error constructors

// InvalidRequestBody returns an RFC 7807 error for malformed request bodies.
// Used when the request body cannot be parsed as valid JSON.
func InvalidRequestBody(err error) *APIError {
	return NewError(
		http.StatusBadRequest,
		ErrorTypeInvalidRequestBody,
		"Invalid Request Body",
		"request body must be valid JSON",
		err,
	)
}

// InvalidResponseBody returns an RFC 7807 error for response encoding failures.
// Used when the server cannot encode a response (internal error).
func InvalidResponseBody(err error) *APIError {
	return NewError(
		http.StatusInternalServerError,
		ErrorTypeInternalError,
		"Internal Server Error",
		unknownErrorMessage,
		err,
	)
}

// MissingBodyProperty returns an RFC 7807 error for missing required fields.
// Used when a required field in the request body is missing.
func MissingBodyProperty(field string) *APIError {
	return NewError(
		http.StatusBadRequest,
		ErrorTypeMissingBodyField,
		"Missing Required Field",
		fmt.Sprintf("%s is required", field),
		fmt.Errorf("missing required field: %s", field),
	)
}

// MissingPathParam returns an RFC 7807 error for missing path parameters.
// Used when a required path parameter is not provided.
func MissingPathParam(param string) *APIError {
	return NewError(
		http.StatusBadRequest,
		ErrorTypeMissingPathParam,
		"Missing Path Parameter",
		fmt.Sprintf("%s is required", param),
		fmt.Errorf("missing path parameter: %s", param),
	)
}

// MissingQueryParam returns an RFC 7807 error for missing query parameters.
// Used when a required query parameter is not provided.
func MissingQueryParam(param string) *APIError {
	return NewError(
		http.StatusBadRequest,
		ErrorTypeMissingQueryParam,
		"Missing Query Parameter",
		fmt.Sprintf("%s is required", param),
		fmt.Errorf("missing query parameter: %s", param),
	)
}

// MissingHeader returns an RFC 7807 error for missing required headers.
// Used when a required HTTP header is not provided.
func MissingHeader(header string) *APIError {
	return NewError(
		http.StatusBadRequest,
		ErrorTypeMissingHeader,
		"Missing Required Header",
		fmt.Sprintf("%s header is required", header),
		fmt.Errorf("missing required header: %s", header),
	)
}

// Gate-specific error constructors

// InvalidGateID returns an RFC 7807 error for invalid gate ID format.
// Used when gate_id path parameter is not a valid UUID.
func InvalidGateID(id string) *APIError {
	return NewError(
		http.StatusBadRequest,
		ErrorTypeInvalidRequestBody,
		"Invalid Gate ID",
		fmt.Sprintf("gate_id must be a valid UUID, got %q", id),
		nil,
	)
}

// GateNotFound returns an RFC 7807 error for non-existent gates.
// Used when a gate with the given ID does not exist in the system.
func GateNotFound(id string) *APIError {
	return NewError(
		http.StatusNotFound,
		ErrorTypeNotFound,
		"Gate Not Found",
		fmt.Sprintf("gate with id %q does not exist", id),
		nil,
	)
}

// InvalidGateURL returns an RFC 7807 error for invalid gate URL format.
// Used when live_url or shadow_url use invalid schemes (must be http/https).
func InvalidGateURL(err error) *APIError {
	detail := "live_url and shadow_url must use http or https scheme"
	if err != nil {
		detail = fmt.Sprintf("%s: %v", detail, err)
	}
	return NewError(
		http.StatusBadRequest,
		ErrorTypeInvalidRequestBody,
		"Invalid Gate URL",
		detail,
		err,
	)
}

// InvalidGateName returns an RFC 7807 error for invalid gate name.
// Used when the name field is empty or exceeds the maximum length.
func InvalidGateName(err error) *APIError {
	detail := "name must be a non-empty string of at most 255 characters"
	if err != nil {
		detail = fmt.Sprintf("%s: %v", detail, err)
	}
	return NewError(
		http.StatusBadRequest,
		ErrorTypeInvalidRequestBody,
		"Invalid Gate Name",
		detail,
		err,
	)
}

// DuplicateGateName returns an RFC 7807 error for duplicate gate names.
// Used when a gate with the same name already exists.
func DuplicateGateName(err error) *APIError {
	detail := "a gate with this name already exists"
	if err != nil {
		detail = fmt.Sprintf("%s: %v", detail, err)
	}
	return NewError(
		http.StatusConflict,
		ErrorTypeConflict,
		"Duplicate Gate Name",
		detail,
		err,
	)
}

// DuplicateGateURLs returns an RFC 7807 error for duplicate gate URL pairs.
// Used when a gate with the same live_url + shadow_url combination already exists.
func DuplicateGateURLs(err error) *APIError {
	detail := "a gate with this live_url and shadow_url pair already exists"
	if err != nil {
		detail = fmt.Sprintf("%s: %v", detail, err)
	}
	return NewError(
		http.StatusConflict,
		ErrorTypeConflict,
		"Duplicate Gate URLs",
		detail,
		err,
	)
}

// InvalidGatePagination returns an RFC 7807 error for invalid pagination parameters.
// Used when limit or offset query parameters are invalid (e.g., negative values).
func InvalidGatePagination(err error) *APIError {
	detail := "limit and offset must be non-negative integers"
	if err != nil {
		detail = fmt.Sprintf("%s: %v", detail, err)
	}
	return NewError(
		http.StatusBadRequest,
		ErrorTypeInvalidQueryParam,
		"Invalid Query Parameter",
		detail,
		err,
	)
}

// InvalidGateSort returns an RFC 7807 error for invalid gate sort parameters.
// Used when sort or order query parameters are invalid for gate listing.
func InvalidGateSort(err error) *APIError {
	detail := "sort must be 'id', 'name', 'live_url', 'shadow_url', or 'created_at'; order must be 'asc' or 'desc'"
	if err != nil {
		detail = fmt.Sprintf("%s: %v", detail, err)
	}
	return NewError(
		http.StatusBadRequest,
		ErrorTypeInvalidQueryParam,
		"Invalid Query Parameter",
		detail,
		err,
	)
}

// Request-specific error constructors

// InvalidRequestID returns an RFC 7807 error for invalid request ID format.
// Used when request_id path parameter is not a valid UUID.
func InvalidRequestID(id string) *APIError {
	return NewError(
		http.StatusBadRequest,
		ErrorTypeInvalidRequestBody,
		"Invalid Request ID",
		fmt.Sprintf("request_id must be a valid UUID, got %q", id),
		nil,
	)
}

// RequestNotFound returns an RFC 7807 error for non-existent requests.
// Used when a request with the given ID does not exist in the system.
func RequestNotFound(id string) *APIError {
	return NewError(
		http.StatusNotFound,
		ErrorTypeNotFound,
		"Request Not Found",
		fmt.Sprintf("request with id %q does not exist", id),
		nil,
	)
}

// InvalidRequestPagination returns an RFC 7807 error for invalid pagination parameters.
// Used when limit or offset query parameters are invalid for request listing.
func InvalidRequestPagination(err error) *APIError {
	detail := "limit and offset must be non-negative integers"
	if err != nil {
		detail = fmt.Sprintf("%s: %v", detail, err)
	}
	return NewError(
		http.StatusBadRequest,
		ErrorTypeInvalidQueryParam,
		"Invalid Query Parameter",
		detail,
		err,
	)
}

// InvalidRequestFilters returns an RFC 7807 error for invalid filter parameters.
// Used when request filter query parameters (methods, path_pattern, from_date, to_date, or has_diff) are invalid.
func InvalidRequestFilters(err error) *APIError {
	detail := "invalid filter parameters: methods, path_pattern, from_date, to_date, or has_diff"
	if err != nil {
		detail = fmt.Sprintf("%s: %v", detail, err)
	}
	return NewError(
		http.StatusBadRequest,
		ErrorTypeInvalidQueryParam,
		"Invalid Query Parameter",
		detail,
		err,
	)
}

// InvalidRequestSort returns an RFC 7807 error for invalid sort parameters.
// Used when sort_by or sort_order query parameters are invalid.
func InvalidRequestSort(err error) *APIError {
	detail := "sort_by must be 'created_at', 'method', or 'path'; sort_order must be 'asc' or 'desc'"
	if err != nil {
		detail = fmt.Sprintf("%s: %v", detail, err)
	}
	return NewError(
		http.StatusBadRequest,
		ErrorTypeInvalidQueryParam,
		"Invalid Query Parameter",
		detail,
		err,
	)
}
