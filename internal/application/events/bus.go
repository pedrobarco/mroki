// Package events provides a minimal in-process domain event bus. Aggregates
// raise domain events (see internal/domain/traffictesting); the use case
// dispatches them through this bus after persistence, and subscribers react
// without coupling the domain/application layers to their concerns.
package events

import (
	"context"
	"log/slog"

	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
)

// Handler reacts to a dispatched domain event. Handlers run synchronously and
// must be cheap and best-effort: a handler must never assume it can fail the
// originating operation.
type Handler func(ctx context.Context, event traffictesting.DomainEvent)

// Dispatcher dispatches domain events to subscribed handlers. The application
// layer depends only on this interface.
type Dispatcher interface {
	Dispatch(ctx context.Context, events ...traffictesting.DomainEvent)
}

// Bus is a synchronous, in-process event bus. Handlers are invoked in
// subscription order on the caller's goroutine; a panicking handler is recovered
// and logged so one subscriber can never fail or block the dispatching
// operation (best-effort delivery).
//
// The Bus is not safe for concurrent Subscribe/Dispatch: all handlers must be
// registered during composition (before the first Dispatch), after which the
// handler set is read-only and Dispatch may be called concurrently.
type Bus struct {
	handlers map[string][]Handler
	logger   *slog.Logger
}

// BusOption configures a Bus.
type BusOption func(*Bus)

// WithLogger sets the logger used to report recovered handler panics.
func WithLogger(l *slog.Logger) BusOption {
	return func(b *Bus) {
		if l != nil {
			b.logger = l
		}
	}
}

// NewBus constructs an empty Bus.
func NewBus(opts ...BusOption) *Bus {
	b := &Bus{
		handlers: make(map[string][]Handler),
		logger:   slog.Default(),
	}
	for _, o := range opts {
		o(b)
	}
	return b
}

// Subscribe registers h to receive events whose EventName equals eventName. It
// must be called during composition, before the first Dispatch (see Bus).
func (b *Bus) Subscribe(eventName string, h Handler) {
	b.handlers[eventName] = append(b.handlers[eventName], h)
}

// Dispatch delivers each event to every handler subscribed to its EventName.
// Delivery is synchronous and best-effort: a handler panic is recovered and
// logged, and the remaining handlers still run.
func (b *Bus) Dispatch(ctx context.Context, events ...traffictesting.DomainEvent) {
	for _, evt := range events {
		for _, h := range b.handlers[evt.EventName()] {
			b.deliver(ctx, h, evt)
		}
	}
}

func (b *Bus) deliver(ctx context.Context, h Handler, evt traffictesting.DomainEvent) {
	defer func() {
		if r := recover(); r != nil {
			b.logger.Error("event handler panicked",
				"event", evt.EventName(),
				"panic", r,
			)
		}
	}()
	h(ctx, evt)
}
