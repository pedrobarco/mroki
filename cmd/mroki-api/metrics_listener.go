package main

import (
	"context"

	"github.com/pedrobarco/mroki/internal/application/events"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	diffmetrics "github.com/pedrobarco/mroki/pkg/diff/metrics"
)

// newComparisonMetricsListener returns an event handler that records the shared
// business comparison metrics for every RequestCompared domain event. It bridges
// the domain event onto the diff metrics Recorder so Prometheus stays out of the
// domain and application layers, and so any path that compares responses (not
// just the HTTP handler) is instrumented uniformly. recorder must be non-nil;
// callers subscribe this listener only when metrics are enabled.
func newComparisonMetricsListener(recorder *diffmetrics.Recorder) events.Handler {
	return func(ctx context.Context, event traffictesting.DomainEvent) {
		evt, ok := event.(traffictesting.RequestCompared)
		if !ok {
			return
		}
		recorder.Observe(ctx, evt.GateID().String(), evt.DiffContent(), nil)
	}
}
