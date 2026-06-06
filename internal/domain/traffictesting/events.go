package traffictesting

import (
	"time"

	"github.com/pedrobarco/mroki/internal/domain/event"
	"github.com/pedrobarco/mroki/pkg/diff"
)

// RequestCompared implements the shared event.Event contract.
var _ event.Event = RequestCompared{}

// EventRequestCompared is the EventName of the RequestCompared domain event.
const EventRequestCompared = "traffictesting.request_compared"

// RequestCompared is a domain event recording that a request's live and shadow
// responses were compared for a gate. It is an immutable fact raised by the
// Request aggregate once a comparison is established; subscribers react to it
// (e.g. to record business metrics) without coupling the domain to their
// concerns.
type RequestCompared struct {
	gateID          GateID
	diffContent     []diff.PatchOp
	liveStatus      int
	shadowStatus    int
	liveLatencyMs   int64
	shadowLatencyMs int64
	occurredAt      time.Time
}

// EventName implements event.Event.
func (e RequestCompared) EventName() string { return EventRequestCompared }

// OccurredAt implements event.Event.
func (e RequestCompared) OccurredAt() time.Time { return e.occurredAt }

// GateID returns the gate the compared request belongs to.
func (e RequestCompared) GateID() GateID { return e.gateID }

// DiffContent returns the JSON-Patch operations describing the differences
// between the live and shadow responses (empty when they matched).
func (e RequestCompared) DiffContent() []diff.PatchOp { return e.diffContent }

// HasDiff reports whether the compared responses differed.
func (e RequestCompared) HasDiff() bool { return len(e.diffContent) > 0 }

// LiveStatus returns the live response HTTP status code.
func (e RequestCompared) LiveStatus() int { return e.liveStatus }

// ShadowStatus returns the shadow response HTTP status code.
func (e RequestCompared) ShadowStatus() int { return e.shadowStatus }

// LiveLatencyMs returns the live response latency in milliseconds.
func (e RequestCompared) LiveLatencyMs() int64 { return e.liveLatencyMs }

// ShadowLatencyMs returns the shadow response latency in milliseconds.
func (e RequestCompared) ShadowLatencyMs() int64 { return e.shadowLatencyMs }
