package diffing

import (
	"net/http"
	"time"

	"github.com/google/uuid"
)

type RequestMethod string

type Request struct {
	ID        RequestID
	GateID    GateID
	Method    string
	Path      string
	Headers   http.Header
	Body      []byte
	AgentID   uuid.UUID
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

func NewRequest(
	gateID GateID,
	method string,
	path string,
	headers http.Header,
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
