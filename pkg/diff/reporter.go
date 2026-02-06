package diff

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"

	"github.com/google/go-cmp/cmp"
)

// cleanReporter is a custom cmp.Reporter that produces clean, human-readable
// diff output without Go type annotations (map[string]any, float64(), etc.)
type cleanReporter struct {
	buf      bytes.Buffer
	allSteps []cmp.PathStep // all steps from root to current
	hasDiff  bool
}

// cleanReporter implements the cmp.Reporter interface

// PushStep is called when traversing down the value tree
func (r *cleanReporter) PushStep(ps cmp.PathStep) {
	r.allSteps = append(r.allSteps, ps)
}

// Report is called on leaf nodes to report differences
func (r *cleanReporter) Report(rs cmp.Result) {
	if rs.Equal() {
		return
	}

	r.hasDiff = true

	// Get the last step to extract values
	var vx, vy reflect.Value
	if len(r.allSteps) > 0 {
		vx, vy = r.allSteps[len(r.allSteps)-1].Values()
	}

	// Build the clean path string from all steps
	pathStr := r.buildCleanPath()

	// Write diff line
	r.buf.WriteString("  ")
	if pathStr != "" {
		r.buf.WriteString(pathStr)
		r.buf.WriteString(":\n  ")
	}
	r.buf.WriteString("- ")
	r.buf.WriteString(r.formatValue(vx))
	r.buf.WriteString("\n  ")
	r.buf.WriteString("+ ")
	r.buf.WriteString(r.formatValue(vy))
	r.buf.WriteString("\n")
}

// PopStep is called when traversing up the value tree
func (r *cleanReporter) PopStep() {
	if len(r.allSteps) > 0 {
		r.allSteps = r.allSteps[:len(r.allSteps)-1]
	}
}

// String returns the formatted diff output
func (r *cleanReporter) String() string {
	if !r.hasDiff {
		return ""
	}
	return r.buf.String()
}

// buildCleanPath builds a path string, filtering out type conversion steps
func (r *cleanReporter) buildCleanPath() string {
	var parts []string

	for _, step := range r.allSteps {
		part := r.stepToString(step)
		if part != "" {
			parts = append(parts, part)
		}
	}

	if len(parts) == 0 {
		return ""
	}

	// Join parts intelligently
	var result strings.Builder
	for i, part := range parts {
		if strings.HasPrefix(part, "[") {
			// Array/map index - no separator needed
			result.WriteString(part)
		} else {
			// Object key - add dot separator if not first
			if i > 0 {
				// Only add dot if previous wasn't an index
				if !strings.HasSuffix(parts[i-1], "]") {
					result.WriteString(".")
				}
			}
			result.WriteString(part)
		}
	}

	return result.String()
}

// stepToString converts a PathStep to a string, or "" if it should be skipped
func (r *cleanReporter) stepToString(ps cmp.PathStep) string {
	switch s := ps.(type) {
	case cmp.MapIndex:
		key := s.Key()
		if key.Kind() == reflect.String {
			return key.String()
		}
		return fmt.Sprintf("[%v]", r.formatValue(key))
	case cmp.SliceIndex:
		return fmt.Sprintf("[%d]", s.Key())
	case cmp.StructField:
		return s.Name()
	default:
		// Skip type conversions, root types, etc.
		return ""
	}
}

// formatValue formats a value without Go type annotations
func (r *cleanReporter) formatValue(v reflect.Value) string {
	if !v.IsValid() {
		return "null"
	}

	switch v.Kind() {
	case reflect.String:
		return fmt.Sprintf(`"%s"`, v.String())

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", v.Int())

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", v.Uint())

	case reflect.Float32, reflect.Float64:
		f := v.Float()
		// Format floats without decimals if they're whole numbers
		if f == float64(int64(f)) {
			return fmt.Sprintf("%d", int64(f))
		}
		return fmt.Sprintf("%g", f)

	case reflect.Bool:
		return fmt.Sprintf("%t", v.Bool())

	case reflect.Slice, reflect.Array:
		return r.formatSlice(v)

	case reflect.Map:
		return r.formatMap(v)

	case reflect.Interface:
		if v.IsNil() {
			return "null"
		}
		return r.formatValue(v.Elem())

	case reflect.Pointer:
		if v.IsNil() {
			return "null"
		}
		return r.formatValue(v.Elem())

	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}

// formatSlice formats a slice/array without type annotations
func (r *cleanReporter) formatSlice(v reflect.Value) string {
	if v.Len() == 0 {
		return "[]"
	}

	var buf strings.Builder
	buf.WriteString("[")

	for i := 0; i < v.Len(); i++ {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(r.formatValue(v.Index(i)))
	}

	buf.WriteString("]")
	return buf.String()
}

// formatMap formats a map without type annotations
func (r *cleanReporter) formatMap(v reflect.Value) string {
	if v.Len() == 0 {
		return "{}"
	}

	var buf strings.Builder
	buf.WriteString("{")

	keys := v.MapKeys()
	for i, key := range keys {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(r.formatValue(key))
		buf.WriteString(": ")
		buf.WriteString(r.formatValue(v.MapIndex(key)))
	}

	buf.WriteString("}")
	return buf.String()
}
