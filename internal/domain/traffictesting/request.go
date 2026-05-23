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
	Headers   Headers
	Body      json.RawMessage
	CreatedAt time.Time

	LiveResponse   Response
	ShadowResponse Response
	Diff           Diff
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
