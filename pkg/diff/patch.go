package diff

import (
	"encoding/json"
	"fmt"
	"strings"
)

// PatchOp represents a single RFC 6902 JSON Patch operation.
type PatchOp struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value,omitempty"`
}

// FormatOps returns a human-readable string representation of a list of patch operations.
// This is useful for logging and CLI output.
func FormatOps(ops []PatchOp) string {
	if len(ops) == 0 {
		return ""
	}

	var buf strings.Builder
	for _, op := range ops {
		fmt.Fprintf(&buf, "  %s %s", op.Op, op.Path)
		if op.Value != nil {
			b, err := json.Marshal(op.Value)
			if err == nil {
				fmt.Fprintf(&buf, ": %s", string(b))
			}
		}
		buf.WriteString("\n")
	}
	return buf.String()
}
