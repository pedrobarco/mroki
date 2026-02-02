package traffictesting

import (
	"time"
)

type RequestMethod string

type Request struct {
	ID        RequestID
	GateID    GateID
	Method    HTTPMethod
	Path      Path
	Headers   Headers
	Body      []byte
	AgentID   AgentID
	CreatedAt time.Time

	Responses []Response
	Diff      Diff
}

type requestOption func(*Request)

func WithRequestID(id RequestID) requestOption {
	return func(r *Request) {
		r.ID = id
	}
}

func WithAgentID(id AgentID) requestOption {
	return func(r *Request) {
		r.AgentID = id
	}
}

func NewRequest(
	gateID GateID,
	method HTTPMethod,
	path Path,
	headers Headers,
	body []byte,
	createdAt time.Time,
	responses []Response,
	diff Diff,
	opts ...requestOption,
) (*Request, error) {
	request := &Request{
		GateID:    gateID,
		Method:    method,
		Path:      path,
		Headers:   headers,
		Body:      body,
		CreatedAt: createdAt,
		Responses: responses,
		Diff:      diff,
	}

	for _, o := range opts {
		o(request)
	}

	if request.ID.IsZero() {
		request.ID = NewRequestID()
	}

	return request, nil
}
