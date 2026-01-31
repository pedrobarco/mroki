package config

import (
	"errors"
	"strings"
	"testing"
)

func TestValidationError_Error_no_errors(t *testing.T) {
	ve := &ValidationError{}

	result := ve.Error()
	expected := "no validation errors"

	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestValidationError_Error_single_error(t *testing.T) {
	ve := &ValidationError{
		Errors: []error{errors.New("field is required")},
	}

	result := ve.Error()
	expected := "field is required"

	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestValidationError_Error_multiple_errors(t *testing.T) {
	ve := &ValidationError{
		Errors: []error{
			errors.New("field1 is required"),
			errors.New("field2 is invalid"),
			errors.New("field3 must be positive"),
		},
	}

	result := ve.Error()

	if !strings.Contains(result, "3 validation errors:") {
		t.Errorf("expected header '3 validation errors:', got: %s", result)
	}

	if !strings.Contains(result, "1. field1 is required") {
		t.Errorf("expected error 1, got: %s", result)
	}

	if !strings.Contains(result, "2. field2 is invalid") {
		t.Errorf("expected error 2, got: %s", result)
	}

	if !strings.Contains(result, "3. field3 must be positive") {
		t.Errorf("expected error 3, got: %s", result)
	}
}

func TestValidationError_Add_non_nil(t *testing.T) {
	ve := &ValidationError{}

	ve.Add(errors.New("error 1"))
	ve.Add(errors.New("error 2"))

	if len(ve.Errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(ve.Errors))
	}
}

func TestValidationError_Add_nil(t *testing.T) {
	ve := &ValidationError{}

	ve.Add(errors.New("error 1"))
	ve.Add(nil)
	ve.Add(errors.New("error 2"))

	if len(ve.Errors) != 2 {
		t.Errorf("expected 2 errors (nil ignored), got %d", len(ve.Errors))
	}
}

func TestValidationError_HasErrors_true(t *testing.T) {
	ve := &ValidationError{
		Errors: []error{errors.New("some error")},
	}

	if !ve.HasErrors() {
		t.Error("expected HasErrors() to return true")
	}
}

func TestValidationError_HasErrors_false(t *testing.T) {
	ve := &ValidationError{}

	if ve.HasErrors() {
		t.Error("expected HasErrors() to return false")
	}
}
