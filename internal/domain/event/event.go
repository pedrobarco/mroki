// Package event defines the neutral domain-event contract shared by every
// bounded context and the in-process event bus. It depends only on the standard
// library so any domain context can implement events and any dispatcher can
// route them without coupling to a specific bounded context.
package event

import "time"

// Event is the contract every domain event implements. EventName identifies the
// event for routing; OccurredAt records when the fact happened.
type Event interface {
	EventName() string
	OccurredAt() time.Time
}
