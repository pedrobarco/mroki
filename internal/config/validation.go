package config

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/pedrobarco/mroki/pkg/logger"
)

// ValidateCORSOrigins appends error-severity findings for CORS origin
// misconfiguration. Every non-wildcard origin must be a valid http or https
// URL.
//
// The wildcard "*" is only permitted when allowAuthorization is false: a
// wildcard origin combined with an allowed Authorization header would let any
// origin drive authenticated cross-origin requests, so it is rejected at
// startup rather than silently producing an unsafe policy.
func ValidateCORSOrigins(verr *ValidationError, origins []string, allowAuthorization bool) {
	for _, origin := range origins {
		if origin == "*" {
			if allowAuthorization {
				verr.Add(SeverityError, `cors_origins cannot be "*" (wildcard) when the Authorization header is allowed; set explicit origins (e.g. https://hub.example.com) instead`)
			}
			continue
		}

		u, err := url.Parse(origin)
		if err != nil {
			verr.Add(SeverityError, fmt.Sprintf("cors_origins contains invalid URL %q: %v", origin, err))
		} else if u.Scheme != "http" && u.Scheme != "https" {
			verr.Add(SeverityError, fmt.Sprintf("cors_origins entry %q must use http or https scheme, got %q", origin, u.Scheme))
		}
	}
}

// ValidateTrustedProxies appends error-severity findings for trusted-proxy
// entries that are neither a valid CIDR (e.g. "10.0.0.0/8") nor a bare IP
// address (e.g. "192.168.1.1"). Each entry decides whether a request's
// X-Forwarded-For header may be trusted, so an unparseable entry is rejected at
// startup rather than silently ignored.
func ValidateTrustedProxies(verr *ValidationError, entries []string) {
	for _, entry := range entries {
		if _, _, err := net.ParseCIDR(entry); err == nil {
			continue
		}
		if net.ParseIP(entry) != nil {
			continue
		}
		verr.Add(SeverityError, fmt.Sprintf("trusted_proxies entry %q must be a valid IP or CIDR", entry))
	}
}

// ValidateLogSettings appends error-severity findings for unrecognised log
// level or format values, using pkg/logger as the single source of truth for
// the accepted values. Empty values are treated as valid so that the env
// defaults apply.
func ValidateLogSettings(verr *ValidationError, level, format string) {
	if level != "" {
		if _, err := logger.ParseLevel(level); err != nil {
			verr.Add(SeverityError, err.Error())
		}
	}
	if format != "" {
		if _, err := logger.ParseFormat(format); err != nil {
			verr.Add(SeverityError, err.Error())
		}
	}
}

// ValidateAppEnv appends an error-severity finding when the application
// environment is set to an unrecognised value. An empty value is treated as
// valid because the config loader resolves an unset APP_ENV to development
// before validation runs.
func ValidateAppEnv(verr *ValidationError, env AppEnv) {
	if env != "" && !env.IsValid() {
		verr.Add(SeverityError, fmt.Sprintf("APP_ENV must be one of development, production, got %q", env))
	}
}

// Severity indicates whether a validation finding is a hard error or a warning.
type Severity int

const (
	// SeverityError indicates a configuration problem that prevents startup.
	SeverityError Severity = iota
	// SeverityWarning indicates a suboptimal configuration that may cause
	// unexpected behaviour but does not prevent startup.
	SeverityWarning
)

// String returns a human-readable label for the severity.
func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "error"
	case SeverityWarning:
		return "warning"
	default:
		return "unknown"
	}
}

// FieldError represents a single validation finding with a severity.
type FieldError struct {
	Severity Severity
	Message  string
}

// Error implements the error interface.
func (e FieldError) Error() string { return e.Message }

// ValidationError accumulates validation findings (errors and warnings)
// and formats them for display. This allows catching all configuration
// issues at once rather than forcing users to fix them one at a time.
type ValidationError struct {
	Entries []FieldError
}

// Add appends a finding with the given severity and message.
func (e *ValidationError) Add(severity Severity, msg string) {
	e.Entries = append(e.Entries, FieldError{Severity: severity, Message: msg})
}

// HasEntries returns true if there are any findings (errors or warnings).
func (e *ValidationError) HasEntries() bool {
	return len(e.Entries) > 0
}

// HasErrors returns true if there are any error-severity findings.
func (e *ValidationError) HasErrors() bool {
	for _, fe := range e.Entries {
		if fe.Severity == SeverityError {
			return true
		}
	}
	return false
}

// HasWarnings returns true if there are any warning-severity findings.
func (e *ValidationError) HasWarnings() bool {
	for _, fe := range e.Entries {
		if fe.Severity == SeverityWarning {
			return true
		}
	}
	return false
}

// Errors returns only the error-severity findings.
func (e *ValidationError) Errors() []FieldError {
	var errs []FieldError
	for _, fe := range e.Entries {
		if fe.Severity == SeverityError {
			errs = append(errs, fe)
		}
	}
	return errs
}

// Warnings returns only the warning-severity findings.
func (e *ValidationError) Warnings() []FieldError {
	var warns []FieldError
	for _, fe := range e.Entries {
		if fe.Severity == SeverityWarning {
			warns = append(warns, fe)
		}
	}
	return warns
}

// Error implements the error interface, formatting only error-severity
// findings as a numbered list. Warnings are excluded.
func (e *ValidationError) Error() string {
	errs := e.Errors()
	if len(errs) == 0 {
		return "no validation errors"
	}
	if len(errs) == 1 {
		return errs[0].Message
	}

	var buf strings.Builder
	fmt.Fprintf(&buf, "%d validation errors:\n", len(errs))
	for i, fe := range errs {
		fmt.Fprintf(&buf, "  %d. %s\n", i+1, fe.Message)
	}
	return buf.String()
}
