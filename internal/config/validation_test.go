package config

import (
	"strings"
	"testing"
)

func TestSeverity_String(t *testing.T) {
	if SeverityError.String() != "error" {
		t.Errorf("expected %q, got %q", "error", SeverityError.String())
	}
	if SeverityWarning.String() != "warning" {
		t.Errorf("expected %q, got %q", "warning", SeverityWarning.String())
	}
	if Severity(99).String() != "unknown" {
		t.Errorf("expected %q, got %q", "unknown", Severity(99).String())
	}
}

func TestFieldError_Error(t *testing.T) {
	fe := FieldError{Severity: SeverityError, Message: "bad value"}
	if fe.Error() != "bad value" {
		t.Errorf("expected %q, got %q", "bad value", fe.Error())
	}
}

func TestValidationError_Add(t *testing.T) {
	ve := &ValidationError{}
	ve.Add(SeverityError, "err1")
	ve.Add(SeverityWarning, "warn1")

	if len(ve.Entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(ve.Entries))
	}
	if ve.Entries[0].Severity != SeverityError || ve.Entries[0].Message != "err1" {
		t.Errorf("unexpected first entry: %v", ve.Entries[0])
	}
	if ve.Entries[1].Severity != SeverityWarning || ve.Entries[1].Message != "warn1" {
		t.Errorf("unexpected second entry: %v", ve.Entries[1])
	}
}

func TestValidationError_HasEntries(t *testing.T) {
	ve := &ValidationError{}
	if ve.HasEntries() {
		t.Error("expected HasEntries() false on empty")
	}
	ve.Add(SeverityWarning, "w")
	if !ve.HasEntries() {
		t.Error("expected HasEntries() true after add")
	}
}

func TestValidationError_HasErrors(t *testing.T) {
	ve := &ValidationError{}
	if ve.HasErrors() {
		t.Error("expected HasErrors() false on empty")
	}

	ve.Add(SeverityWarning, "warning only")
	if ve.HasErrors() {
		t.Error("expected HasErrors() false with only warnings")
	}

	ve.Add(SeverityError, "an error")
	if !ve.HasErrors() {
		t.Error("expected HasErrors() true after adding error")
	}
}

func TestValidationError_HasWarnings(t *testing.T) {
	ve := &ValidationError{}
	if ve.HasWarnings() {
		t.Error("expected HasWarnings() false on empty")
	}

	ve.Add(SeverityError, "error only")
	if ve.HasWarnings() {
		t.Error("expected HasWarnings() false with only errors")
	}

	ve.Add(SeverityWarning, "a warning")
	if !ve.HasWarnings() {
		t.Error("expected HasWarnings() true after adding warning")
	}
}

func TestValidationError_Errors_and_Warnings(t *testing.T) {
	ve := &ValidationError{}
	ve.Add(SeverityError, "err1")
	ve.Add(SeverityWarning, "warn1")
	ve.Add(SeverityError, "err2")
	ve.Add(SeverityWarning, "warn2")

	errs := ve.Errors()
	if len(errs) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(errs))
	}
	if errs[0].Message != "err1" || errs[1].Message != "err2" {
		t.Errorf("unexpected errors: %v", errs)
	}

	warns := ve.Warnings()
	if len(warns) != 2 {
		t.Fatalf("expected 2 warnings, got %d", len(warns))
	}
	if warns[0].Message != "warn1" || warns[1].Message != "warn2" {
		t.Errorf("unexpected warnings: %v", warns)
	}
}

func TestValidationError_Error_no_errors(t *testing.T) {
	ve := &ValidationError{}
	if ve.Error() != "no validation errors" {
		t.Errorf("expected %q, got %q", "no validation errors", ve.Error())
	}
}

func TestValidationError_Error_single_error(t *testing.T) {
	ve := &ValidationError{}
	ve.Add(SeverityError, "field is required")
	if ve.Error() != "field is required" {
		t.Errorf("expected %q, got %q", "field is required", ve.Error())
	}
}

func TestValidationError_Error_excludes_warnings(t *testing.T) {
	ve := &ValidationError{}
	ve.Add(SeverityWarning, "just a warning")
	if ve.Error() != "no validation errors" {
		t.Errorf("expected %q, got %q", "no validation errors", ve.Error())
	}
}

func TestValidationError_Error_multiple_errors(t *testing.T) {
	ve := &ValidationError{}
	ve.Add(SeverityError, "field1 is required")
	ve.Add(SeverityWarning, "some warning")
	ve.Add(SeverityError, "field2 is invalid")
	ve.Add(SeverityError, "field3 must be positive")

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
	if strings.Contains(result, "some warning") {
		t.Errorf("Error() should not include warnings, got: %s", result)
	}
}
