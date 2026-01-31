package config

import (
	"fmt"
	"strings"
)

// ValidationError accumulates multiple validation errors and formats them
// for display. This allows catching all configuration errors at once rather
// than forcing users to fix them one at a time.
type ValidationError struct {
	Errors []error
}

// Error implements the error interface, formatting multiple errors as a numbered list.
func (e *ValidationError) Error() string {
	if len(e.Errors) == 0 {
		return "no validation errors"
	}
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}

	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("%d validation errors:\n", len(e.Errors)))
	for i, err := range e.Errors {
		buf.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err.Error()))
	}
	return buf.String()
}

// Add appends an error to the validation error list.
// Nil errors are ignored.
func (e *ValidationError) Add(err error) {
	if err != nil {
		e.Errors = append(e.Errors, err)
	}
}

// HasErrors returns true if there are any validation errors.
func (e *ValidationError) HasErrors() bool {
	return len(e.Errors) > 0
}
