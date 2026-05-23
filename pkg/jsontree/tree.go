// Package jsontree provides types and utilities for working with parsed JSON
// value trees — the Go representation produced by json.Unmarshal into any.
//
// A Tree is a type alias for any, constrained by convention to the shapes
// that encoding/json produces:
//
//   - map[string]any  (JSON object)
//   - []any           (JSON array)
//   - string          (JSON string)
//   - float64         (JSON number)
//   - bool            (JSON boolean)
//   - nil             (JSON null)
//
// The tree shape is not JSON-specific — any format that can be parsed into
// map[string]any / []any (YAML, MessagePack, etc.) is compatible with the
// utilities in this package.
package jsontree

// Tree is a Go value tree produced by json.Unmarshal into any.
// Valid shapes: map[string]any (object), []any (array), string, float64, bool, nil.
//
// This is a type alias (not a defined type), so it is fully interchangeable
// with any at call sites — no casting or wrapping required.
type Tree = any
