package diff

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/google/go-cmp/cmp"
)

// patchReporter is a custom cmp.Reporter that collects RFC 6902 JSON Patch
// operations describing differences between two values.
type patchReporter struct {
	ops      []PatchOp
	allSteps []cmp.PathStep // all steps from root to current
}

// PushStep is called when traversing down the value tree.
func (r *patchReporter) PushStep(ps cmp.PathStep) {
	r.allSteps = append(r.allSteps, ps)
}

// Report is called on leaf nodes to report differences.
func (r *patchReporter) Report(rs cmp.Result) {
	if rs.Equal() {
		return
	}

	var vx, vy reflect.Value
	if len(r.allSteps) > 0 {
		vx, vy = r.allSteps[len(r.allSteps)-1].Values()
	}

	path := r.buildJSONPointer()

	switch {
	case !vx.IsValid():
		// Value only in b → add
		r.ops = append(r.ops, PatchOp{
			Op:    "add",
			Path:  path,
			Value: extractValue(vy),
		})
	case !vy.IsValid():
		// Value only in a → remove
		r.ops = append(r.ops, PatchOp{
			Op:   "remove",
			Path: path,
		})
	default:
		// Both present but different → replace
		r.ops = append(r.ops, PatchOp{
			Op:    "replace",
			Path:  path,
			Value: extractValue(vy),
		})
	}
}

// PopStep is called when traversing up the value tree.
func (r *patchReporter) PopStep() {
	if len(r.allSteps) > 0 {
		r.allSteps = r.allSteps[:len(r.allSteps)-1]
	}
}

// Ops returns the collected patch operations.
func (r *patchReporter) Ops() []PatchOp {
	return r.ops
}

// buildJSONPointer builds an RFC 6901 JSON Pointer from the current path steps.
func (r *patchReporter) buildJSONPointer() string {
	var parts []string
	for _, step := range r.allSteps {
		part := r.stepToPointerToken(step)
		if part != "" {
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return "/"
	}
	return "/" + strings.Join(parts, "/")
}

// stepToPointerToken converts a PathStep to a JSON Pointer token (RFC 6901),
// or "" if it should be skipped.
func (r *patchReporter) stepToPointerToken(ps cmp.PathStep) string {
	switch s := ps.(type) {
	case cmp.MapIndex:
		key := s.Key()
		if key.Kind() == reflect.String {
			// Escape ~ and / per RFC 6901
			token := key.String()
			token = strings.ReplaceAll(token, "~", "~0")
			token = strings.ReplaceAll(token, "/", "~1")
			return token
		}
		return fmt.Sprintf("%v", key.Interface())
	case cmp.SliceIndex:
		return fmt.Sprintf("%d", s.Key())
	case cmp.StructField:
		return s.Name()
	default:
		return ""
	}
}

// extractValue converts a reflect.Value to a plain Go value suitable for JSON marshaling.
func extractValue(v reflect.Value) any {
	if !v.IsValid() {
		return nil
	}

	switch v.Kind() {
	case reflect.Interface, reflect.Pointer:
		if v.IsNil() {
			return nil
		}
		return extractValue(v.Elem())
	default:
		return v.Interface()
	}
}
