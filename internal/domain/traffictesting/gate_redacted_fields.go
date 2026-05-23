package traffictesting

import (
	"fmt"
	"strings"
)

// defaultRedactedFields is the default set of fields always redacted (gjson path notation).
var defaultRedactedFields = []string{
	"headers.Authorization",
	"headers.Cookie",
	"headers.Set-Cookie",
	"headers.X-Api-Key",
}

// DefaultRedactedFieldList returns a copy of the default redacted field paths.
// Callers cannot mutate the canonical list.
func DefaultRedactedFieldList() []string {
	out := make([]string, len(defaultRedactedFields))
	copy(out, defaultRedactedFields)
	return out
}

// RedactedFields holds per-gate field redaction settings.
type RedactedFields struct {
	AdditionalFields []string // extra fields beyond the defaults (gjson paths)
}

// NewRedactedFields creates a validated RedactedFields.
// The input slice is copied — the caller's slice is never mutated.
// Each field must be prefixed with "headers." or "body." to indicate what it targets.
func NewRedactedFields(additionalFields []string) (RedactedFields, error) {
	cleaned := make([]string, len(additionalFields))
	for i, f := range additionalFields {
		trimmed := strings.TrimSpace(f)
		if trimmed == "" {
			return RedactedFields{}, fmt.Errorf("%w: redacted_fields[%d] must not be empty", ErrInvalidRedactedFields, i)
		}
		switch {
		case strings.HasPrefix(trimmed, "headers."):
			if trimmed == "headers." {
				return RedactedFields{}, fmt.Errorf("%w: redacted_fields[%d] %q must specify a field name after \"headers.\"", ErrInvalidRedactedFields, i, trimmed)
			}
		case strings.HasPrefix(trimmed, "body."):
			if trimmed == "body." {
				return RedactedFields{}, fmt.Errorf("%w: redacted_fields[%d] %q must specify a field path after \"body.\"", ErrInvalidRedactedFields, i, trimmed)
			}
		default:
			return RedactedFields{}, fmt.Errorf("%w: redacted_fields[%d] %q must start with \"headers.\" or \"body.\"", ErrInvalidRedactedFields, i, trimmed)
		}
		cleaned[i] = trimmed
	}
	return RedactedFields{AdditionalFields: cleaned}, nil
}

// DefaultRedactedFields returns the zero-value config (no additional fields).
func DefaultRedactedFields() RedactedFields {
	return RedactedFields{AdditionalFields: []string{}}
}

// AllFields returns the merged list of default + additional fields (deduplicated).
// Header fields are deduplicated case-insensitively (e.g. "headers.authorization"
// is considered a duplicate of "headers.Authorization"). Body fields use exact match.
func (r RedactedFields) AllFields() []string {
	seen := make(map[string]struct{}, len(defaultRedactedFields)+len(r.AdditionalFields))
	all := make([]string, 0, len(defaultRedactedFields)+len(r.AdditionalFields))

	dedupKey := func(f string) string {
		if strings.HasPrefix(f, "headers.") {
			return "headers." + strings.ToLower(f[len("headers."):])
		}
		return f
	}

	for _, f := range defaultRedactedFields {
		key := dedupKey(f)
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			all = append(all, f)
		}
	}
	for _, f := range r.AdditionalFields {
		key := dedupKey(f)
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			all = append(all, f)
		}
	}
	return all
}
