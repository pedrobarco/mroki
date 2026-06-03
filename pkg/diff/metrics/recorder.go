// Package metrics holds the shared domain comparison metrics recorded by every
// component that diffs live/shadow responses: the mroki-api server (API mode),
// the standalone mroki-proxy, and the caddy module. Defining them once here lets
// every host emit identical series regardless of where the diff is computed, so
// a standalone proxy exposes the same metrics the API exposes in API mode.
package metrics

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/pedrobarco/mroki/pkg/diff"
)

// Comparison outcome values for the result label on mroki_responses_compared_total.
const (
	ResultMatch = "match"
	ResultDiff  = "diff"
	ResultError = "error"
)

// meterName scopes the instruments created by this package.
const meterName = "github.com/pedrobarco/mroki/pkg/diff/metrics"

// diffOperationsBuckets bounds the per-diff JSON-Patch operation-count
// histogram. Values start at 1 because the histogram is only observed when a
// diff exists (result="diff"); a match records nothing here.
var diffOperationsBuckets = []float64{1, 2, 5, 10, 25, 50, 100, 250, 500}

// Recorder records the shared domain comparison metrics via the OTel Meter API:
//
//   - mroki_responses_compared_total{gate,result} — every comparison, by outcome
//   - mroki_diff_operations{gate}                  — JSON-Patch op count, on diff
//
// A nil *Recorder is a no-op so callers can record unconditionally.
type Recorder struct {
	compared       metric.Int64Counter
	diffOperations metric.Int64Histogram
}

// New constructs the comparison Recorder from mp's Meter.
func New(mp metric.MeterProvider) (*Recorder, error) {
	meter := mp.Meter(meterName)

	compared, err := meter.Int64Counter(
		"mroki.responses_compared",
		metric.WithDescription("Total live/shadow responses compared, by gate and result."),
	)
	if err != nil {
		return nil, fmt.Errorf("create responses_compared counter: %w", err)
	}

	diffOperations, err := meter.Int64Histogram(
		"mroki.diff_operations",
		metric.WithDescription("Distribution of JSON-Patch operation counts per differing comparison, by gate."),
		metric.WithExplicitBucketBoundaries(diffOperationsBuckets...),
	)
	if err != nil {
		return nil, fmt.Errorf("create diff_operations histogram: %w", err)
	}

	return &Recorder{compared: compared, diffOperations: diffOperations}, nil
}

// Observe records a single comparison. The result is derived from err and ops:
// a non-nil err records result="error"; a non-empty ops records result="diff"
// and observes len(ops) on the diff_operations histogram; otherwise
// result="match". gate may be empty (e.g. when the caller is not bound to a
// gate). A nil Recorder is a no-op.
func (r *Recorder) Observe(ctx context.Context, gate string, ops []diff.PatchOp, err error) {
	if r == nil {
		return
	}

	result := ResultMatch
	switch {
	case err != nil:
		result = ResultError
	case len(ops) > 0:
		result = ResultDiff
	}

	gateAttr := attribute.String("gate", gate)
	r.compared.Add(ctx, 1, metric.WithAttributes(gateAttr, attribute.String("result", result)))
	if result == ResultDiff {
		r.diffOperations.Record(ctx, int64(len(ops)), metric.WithAttributes(gateAttr))
	}
}
