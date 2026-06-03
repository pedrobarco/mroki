package metrics

import (
	"context"
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pedrobarco/mroki/pkg/diff"
	"github.com/pedrobarco/mroki/pkg/metrics"
)

// newTestRecorder builds a Recorder bridged onto a fresh Prometheus registry so
// tests (and benchmarks) can scrape the resulting exposition.
func newTestRecorder(tb testing.TB) (*Recorder, *prometheus.Registry) {
	tb.Helper()
	reg := prometheus.NewRegistry()
	mp, err := metrics.NewMeterProvider(reg)
	require.NoError(tb, err)
	r, err := New(mp)
	require.NoError(tb, err)
	return r, reg
}

func findFamily(t *testing.T, fams []*dto.MetricFamily, name string) *dto.MetricFamily {
	t.Helper()
	for _, f := range fams {
		if f.GetName() == name {
			return f
		}
	}
	t.Fatalf("metric family %q not found (have %d families)", name, len(fams))
	return nil
}

// counterValue returns the value of the counter sample whose labels match all
// of want, or fails when no such sample exists.
func counterValue(t *testing.T, f *dto.MetricFamily, want map[string]string) float64 {
	t.Helper()
	for _, m := range f.GetMetric() {
		if labelsMatch(m, want) {
			return m.GetCounter().GetValue()
		}
	}
	t.Fatalf("no %s sample with labels %v", f.GetName(), want)
	return 0
}

func labelsMatch(m *dto.Metric, want map[string]string) bool {
	got := make(map[string]string, len(m.GetLabel()))
	for _, l := range m.GetLabel() {
		got[l.GetName()] = l.GetValue()
	}
	for k, v := range want {
		if got[k] != v {
			return false
		}
	}
	return true
}

func TestRecorder_Observe_RecordsOutcomes(t *testing.T) {
	r, reg := newTestRecorder(t)
	ctx := context.Background()

	r.Observe(ctx, "gateA", nil, nil)                            // match
	r.Observe(ctx, "gateA", []diff.PatchOp{{}, {}, {}}, nil)     // diff, 3 ops
	r.Observe(ctx, "gateA", nil, errors.New("redaction failed")) // error

	fams, err := reg.Gather()
	require.NoError(t, err)

	compared := findFamily(t, fams, "mroki_responses_compared_total")
	assert.Equal(t, 1.0, counterValue(t, compared, map[string]string{"gate": "gateA", "result": ResultMatch}))
	assert.Equal(t, 1.0, counterValue(t, compared, map[string]string{"gate": "gateA", "result": ResultDiff}))
	assert.Equal(t, 1.0, counterValue(t, compared, map[string]string{"gate": "gateA", "result": ResultError}))

	ops := findFamily(t, fams, "mroki_diff_operations")
	require.Len(t, ops.GetMetric(), 1)
	h := ops.GetMetric()[0].GetHistogram()
	assert.Equal(t, uint64(1), h.GetSampleCount(), "histogram observed once (only on diff)")
	assert.Equal(t, 3.0, h.GetSampleSum(), "histogram sum equals the diff op count")
}

func TestRecorder_Observe_NilIsNoop(t *testing.T) {
	var r *Recorder
	assert.NotPanics(t, func() {
		r.Observe(context.Background(), "gateA", []diff.PatchOp{{}}, nil)
	})
}

// BenchmarkRecorder_Observe measures the per-comparison cost of the domain
// recorder with metrics disabled (a nil *Recorder, the no-op fast path taken
// when metrics are disabled) versus enabled, split by outcome: a match touches
// only the counter, while a diff also records the op-count histogram. It
// quantifies what enabling domain metrics adds to each stored comparison.
func BenchmarkRecorder_Observe(b *testing.B) {
	ctx := context.Background()
	ops := make([]diff.PatchOp, 5) // representative small diff

	b.Run("disabled", func(b *testing.B) {
		var r *Recorder // nil: metrics disabled
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			r.Observe(ctx, "gateA", ops, nil)
		}
	})

	b.Run("enabled/match", func(b *testing.B) {
		r, _ := newTestRecorder(b)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			r.Observe(ctx, "gateA", nil, nil)
		}
	})

	b.Run("enabled/diff", func(b *testing.B) {
		r, _ := newTestRecorder(b)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			r.Observe(ctx, "gateA", ops, nil)
		}
	})
}
