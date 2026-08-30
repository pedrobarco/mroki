package traffictesting

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pedrobarco/mroki/internal/domain/event"
)

// maxClockSkew bounds how far a request's CreatedAt may exceed the current
// time before it is rejected on ingest. It absorbs minor clock differences
// between clients/proxies and the API while preventing far-future timestamps
// from corrupting stats and TTL-based retention cleanup.
const maxClockSkew = 5 * time.Minute

type RequestMethod string

type Request struct {
	ID        RequestID
	GateID    GateID
	Method    HTTPMethod
	Path      Path
	RawQuery  string
	Headers   Headers
	Body      json.RawMessage
	CreatedAt time.Time

	LiveResponse   Response
	ShadowResponse Response
	Diff           Diff

	events []event.Event
}

// raise records a domain event on the aggregate. Events are buffered in memory
// and later drained via PullEvents; raising never performs I/O so the domain
// stays free of infrastructure concerns.
func (r *Request) raise(e event.Event) {
	r.events = append(r.events, e)
}

// PullEvents returns the domain events recorded on this request and clears them,
// so each event is dispatched at most once.
func (r *Request) PullEvents() []event.Event {
	events := r.events
	r.events = nil
	return events
}

type requestOption func(*Request)

func WithRequestID(id RequestID) requestOption {
	return func(r *Request) {
		r.ID = id
	}
}

func NewRequest(
	gateID GateID,
	method HTTPMethod,
	path Path,
	rawQuery string,
	headers Headers,
	body json.RawMessage,
	createdAt time.Time,
	liveResponse Response,
	shadowResponse Response,
	diff Diff,
	opts ...requestOption,
) (*Request, error) {
	if createdAt.After(time.Now().Add(maxClockSkew)) {
		return nil, fmt.Errorf("%w: %s is in the future", ErrCreatedAtInFuture, createdAt)
	}

	request := &Request{
		GateID:         gateID,
		Method:         method,
		Path:           path,
		RawQuery:       rawQuery,
		Headers:        headers,
		Body:           body,
		CreatedAt:      createdAt,
		LiveResponse:   liveResponse,
		ShadowResponse: shadowResponse,
		Diff:           diff,
	}

	for _, o := range opts {
		o(request)
	}

	if request.ID.IsZero() {
		request.ID = NewRequestID()
	}

	// Constructing a Request is the compare transition, so raise the comparison
	// fact here: every path that creates one is instrumented uniformly. Loading
	// from persistence rebuilds the struct directly and therefore never raises.
	request.raise(RequestCompared{
		gateID:          request.GateID,
		diffContent:     request.Diff.Content,
		liveStatus:      request.LiveResponse.StatusCode.Int(),
		shadowStatus:    request.ShadowResponse.StatusCode.Int(),
		liveLatencyMs:   request.LiveResponse.LatencyMs,
		shadowLatencyMs: request.ShadowResponse.LatencyMs,
		occurredAt:      request.CreatedAt,
	})

	return request, nil
}
