package traffictesting

import (
	"time"

	"github.com/google/uuid"
)

type Response struct {
	ID         uuid.UUID
	StatusCode StatusCode
	Headers    Headers
	Body       []byte
	LatencyMs  int64
	CreatedAt  time.Time
}

type ResponseOption func(*Response)

func WithResponseID(id uuid.UUID) ResponseOption {
	return func(r *Response) {
		r.ID = id
	}
}

func NewResponse(
	statusCode StatusCode,
	headers Headers,
	body []byte,
	latencyMs int64,
	createdAt time.Time,
	opts ...ResponseOption,
) (*Response, error) {
	response := &Response{
		StatusCode: statusCode,
		Headers:    headers,
		Body:       body,
		LatencyMs:  latencyMs,
		CreatedAt:  createdAt,
	}

	for _, o := range opts {
		o(response)
	}

	if response.ID == uuid.Nil {
		response.ID = uuid.New()
	}

	return response, nil
}
