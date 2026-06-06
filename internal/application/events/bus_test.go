package events_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pedrobarco/mroki/internal/application/events"
	"github.com/pedrobarco/mroki/internal/domain/event"
)

// stubEvent is a minimal event.Event for routing tests.
type stubEvent struct{ name string }

func (e stubEvent) EventName() string         { return e.name }
func (e stubEvent) OccurredAt() (t time.Time) { return t }

func TestBus_Dispatch_RoutesByEventName(t *testing.T) {
	bus := events.NewBus()

	var got []string
	bus.Subscribe("a", func(_ context.Context, e event.Event) {
		got = append(got, "h1:"+e.EventName())
	})
	bus.Subscribe("a", func(_ context.Context, e event.Event) {
		got = append(got, "h2:"+e.EventName())
	})
	bus.Subscribe("b", func(_ context.Context, e event.Event) {
		got = append(got, "h3:"+e.EventName())
	})

	bus.Dispatch(context.Background(), stubEvent{name: "a"})

	assert.Equal(t, []string{"h1:a", "h2:a"}, got, "only handlers for event 'a', in order")
}

func TestBus_Dispatch_NoSubscribersIsNoop(t *testing.T) {
	bus := events.NewBus()
	assert.NotPanics(t, func() {
		bus.Dispatch(context.Background(), stubEvent{name: "unsubscribed"})
	})
}

func TestBus_Dispatch_RecoversHandlerPanic(t *testing.T) {
	// Silence the recovered-panic log line.
	bus := events.NewBus(events.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))))

	var secondRan bool
	bus.Subscribe("a", func(_ context.Context, _ event.Event) {
		panic("boom")
	})
	bus.Subscribe("a", func(_ context.Context, _ event.Event) {
		secondRan = true
	})

	require.NotPanics(t, func() {
		bus.Dispatch(context.Background(), stubEvent{name: "a"})
	})
	assert.True(t, secondRan, "a panicking handler must not stop later handlers")
}

func TestBus_WithLogger_NilKeepsDefault(t *testing.T) {
	// WithLogger(nil) must be ignored, leaving a usable default logger so panic
	// recovery still works rather than dereferencing a nil logger.
	bus := events.NewBus(events.WithLogger(nil))

	bus.Subscribe("a", func(_ context.Context, _ event.Event) {
		panic("boom")
	})

	require.NotPanics(t, func() {
		bus.Dispatch(context.Background(), stubEvent{name: "a"})
	})
}

func TestBus_Dispatch_MultipleEvents(t *testing.T) {
	bus := events.NewBus()

	var count int
	bus.Subscribe("a", func(_ context.Context, _ event.Event) { count++ })

	bus.Dispatch(context.Background(), stubEvent{name: "a"}, stubEvent{name: "a"})

	assert.Equal(t, 2, count)
}
