package handlers

import (
	"fmt"
	"net/http"
)

const unknownErrorMessage = "An unknown error occurred. Please try again later."

type APIError struct {
	StatusCode int    `json:"status_code"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	Err        error  `json:"-"`
}

var _ error = (*APIError)(nil)

func NewError(statusCode int, message string, err error) *APIError {
	return &APIError{
		StatusCode: statusCode,
		Code:       http.StatusText(statusCode),
		Message:    message,
		Err:        err,
	}
}

func (a *APIError) Error() string {
	return a.Err.Error()
}

func InvalidRequestBody(err error) *APIError {
	return NewError(http.StatusBadRequest, "invalid request body", err)
}

func InvalidResponseBody(err error) *APIError {
	return NewError(http.StatusInternalServerError, "failed to encode response body", err)
}

func MissingBodyProperty(path string) error {
	err := fmt.Errorf("missing required body property '%s'", path)
	return NewError(http.StatusBadRequest, err.Error(), err)
}

func MissingPathParam(param string) *APIError {
	err := fmt.Errorf("missing required path parameter '%s'", param)
	return NewError(http.StatusBadRequest, err.Error(), err)
}
