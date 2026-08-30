package diff_test

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pedrobarco/mroki/pkg/diff"
	"github.com/pedrobarco/mroki/pkg/jsontree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mustMarshal returns the JSON encoding of v, failing the test on error.
// Map keys are emitted in sorted order by encoding/json, so byte equality of
// two snapshots reliably detects any array reordering or key mutation.
func mustMarshal(t *testing.T, v jsontree.Tree) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

// TestParsed_does_not_mutate_input_trees locks in the invariant documented on
// NormalizeTree: the tree it returns is always a fresh copy, so SortArraysInTree
// in Parsed() only ever sorts that copy and never the caller's input. Each case
// exercises a different normalizer path (no-filter, included-only, ignored-only,
// both) with WithSortArrays(true) and unsorted arrays in retained fields, then
// asserts both inputs are byte-identical before and after the call.
func TestParsed_does_not_mutate_input_trees(t *testing.T) {
	tests := []struct {
		name string
		a    jsontree.Tree
		b    jsontree.Tree
		opts []diff.Option
	}{
		{
			name: "no-filter",
			a:    map[string]any{"items": []any{float64(3), float64(1), float64(2)}, "nested": map[string]any{"tags": []any{"b", "a", "c"}}},
			b:    map[string]any{"items": []any{float64(2), float64(3), float64(1)}, "nested": map[string]any{"tags": []any{"c", "b", "a"}}},
			opts: []diff.Option{diff.WithSortArrays(true)},
		},
		{
			name: "included-only",
			a:    map[string]any{"items": []any{float64(3), float64(1), float64(2)}, "other": "x"},
			b:    map[string]any{"items": []any{float64(2), float64(3), float64(1)}, "other": "y"},
			opts: []diff.Option{diff.WithSortArrays(true), diff.WithIncludedFields("items")},
		},
		{
			name: "ignored-only",
			a:    map[string]any{"items": []any{float64(3), float64(1), float64(2)}, "ts": "2024-01-01"},
			b:    map[string]any{"items": []any{float64(2), float64(3), float64(1)}, "ts": "2024-01-02"},
			opts: []diff.Option{diff.WithSortArrays(true), diff.WithIgnoredFields("ts")},
		},
		{
			name: "both",
			a:    map[string]any{"user": map[string]any{"tags": []any{float64(3), float64(1), float64(2)}, "ssn": "111-11-1111"}},
			b:    map[string]any{"user": map[string]any{"tags": []any{float64(2), float64(3), float64(1)}, "ssn": "222-22-2222"}},
			opts: []diff.Option{diff.WithSortArrays(true), diff.WithIncludedFields("user"), diff.WithIgnoredFields("user.ssn")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beforeA := mustMarshal(t, tt.a)
			beforeB := mustMarshal(t, tt.b)

			_, err := diff.Parsed(tt.a, tt.b, tt.opts...)
			require.NoError(t, err)

			assert.Equal(t, string(beforeA), string(mustMarshal(t, tt.a)),
				"first input tree must not be mutated by Parsed()")
			assert.Equal(t, string(beforeB), string(mustMarshal(t, tt.b)),
				"second input tree must not be mutated by Parsed()")
		})
	}
}

// TestParsed_reused_input_yields_identical_ops verifies there is no cross-call
// contamination: reusing the same input trees across two Parsed() calls with
// sortArrays enabled produces identical, correct RFC 6902 ops. If the first call
// had sorted the caller's arrays in place, the second call would see already
// sorted inputs and could yield different results.
func TestParsed_reused_input_yields_identical_ops(t *testing.T) {
	t.Run("equal sets produce no ops on both calls", func(t *testing.T) {
		a := map[string]any{"items": []any{float64(3), float64(1), float64(2)}}
		b := map[string]any{"items": []any{float64(2), float64(3), float64(1)}}

		ops1, err := diff.Parsed(a, b, diff.WithSortArrays(true))
		require.NoError(t, err)
		ops2, err := diff.Parsed(a, b, diff.WithSortArrays(true))
		require.NoError(t, err)

		assert.Empty(t, ops1, "sorted arrays with same elements should produce no diff")
		assert.True(t, cmp.Equal(ops1, ops2),
			"reusing the same input across calls must yield identical ops:\n%s", cmp.Diff(ops1, ops2))
	})

	t.Run("differing sets produce identical ops on both calls", func(t *testing.T) {
		a := map[string]any{"items": []any{float64(3), float64(1), float64(2)}}
		b := map[string]any{"items": []any{float64(9), float64(1), float64(2)}}

		ops1, err := diff.Parsed(a, b, diff.WithSortArrays(true))
		require.NoError(t, err)
		ops2, err := diff.Parsed(a, b, diff.WithSortArrays(true))
		require.NoError(t, err)

		assert.NotEmpty(t, ops1, "differing array elements should produce a diff")
		assert.True(t, cmp.Equal(ops1, ops2),
			"reusing the same input across calls must yield identical ops:\n%s", cmp.Diff(ops1, ops2))
	})
}
