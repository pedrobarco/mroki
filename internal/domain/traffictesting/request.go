package traffictesting

import (
	"encoding/json"
	"time"
)

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

	events []DomainEvent
}

// RecordCompared raises a RequestCompared domain event capturing the outcome of
// comparing this request's live and shadow responses. It is invoked by the use
// case on the create/compare transition and must be called at most once per
// request; events are later drained via PullEvents and dispatched once the
// request is persisted.
func (r *Request) RecordCompared() {
	r.events = append(r.events, RequestCompared{
		gateID:          r.GateID,
		diffContent:     r.Diff.Content,
		liveStatus:      r.LiveResponse.StatusCode.Int(),
		shadowStatus:    r.ShadowResponse.StatusCode.Int(),
		liveLatencyMs:   r.LiveResponse.LatencyMs,
		shadowLatencyMs: r.ShadowResponse.LatencyMs,
		occurredAt:      r.CreatedAt,
	})
}

// PullEvents returns the domain events recorded on this request and clears them,
// so each event is dispatched at most once.
func (r *Request) PullEvents() []DomainEvent {
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

	return request, nil
}
