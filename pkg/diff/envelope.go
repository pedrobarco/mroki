package diff

import (
	"net/http"

	"github.com/pedrobarco/mroki/pkg/jsontree"
)

// BuildEnvelope constructs the synthetic diff envelope as a Go value tree:
//
//	{"statusCode": N, "headers": {...}, "body": ...}
//
// Type conventions (must match json.Unmarshal output for cmp.Equal compatibility):
//   - statusCode: float64 (not int) — json.Unmarshal decodes numbers as float64
//   - headers: map[string]any where values are []any{string, ...} — matches
//     the shape produced by json.Marshal(http.Header) → json.Unmarshal.
//     nil headers produce nil (matching json.Marshal(nil) → "null").
//   - body: as-is from json.Unmarshal (map[string]any, []any, string, float64, etc.)
//     Mutable types (map/slice) are aliased, not copied. Callers must not mutate.
func BuildEnvelope(statusCode int, headers http.Header, bodyParsed jsontree.Tree) map[string]any {
	// Convert http.Header → map[string]any with []any values.
	// json.Marshal(http.Header) produces arrays for all values (even single),
	// and json.Unmarshal parses them as []any. We must match that shape.
	// nil headers stay nil to match json.Marshal(nil) → "null" → json.Unmarshal → nil.
	var h any
	if headers != nil {
		hm := make(map[string]any, len(headers))
		for k, vs := range headers {
			vals := make([]any, len(vs))
			for i, v := range vs {
				vals[i] = v
			}
			hm[k] = vals
		}
		h = hm
	}

	envelope := map[string]any{
		"statusCode": float64(statusCode),
		"headers":    h,
		"body":       bodyParsed, // nil becomes JSON null via cmp.Equal
	}
	return envelope
}
