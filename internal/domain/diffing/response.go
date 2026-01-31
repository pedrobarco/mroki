package diffing

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type ResponseType string

const (
	ResponseTypeLive   ResponseType = "live"
	ResponseTypeShadow ResponseType = "shadow"
)

func NewResponseType(s string) (ResponseType, error) {
	switch s {
	case string(ResponseTypeLive), string(ResponseTypeShadow):
		return ResponseType(s), nil
	default:
		return "", fmt.Errorf("invalid response type: %s", s)
	}
}

type Response struct {
	ID         uuid.UUID
	Type       ResponseType
	StatusCode int
	Headers    http.Header
	Body       []byte
	CreatedAt  time.Time
}

type responseOption func(*Response)

func WithResponseID(id uuid.UUID) responseOption {
	return func(r *Response) {
		r.ID = id
	}
}

func NewResponse(
	responseType ResponseType,
	statusCode int,
	headers http.Header,
	body []byte,
	createdAt time.Time,
	opts ...responseOption,
) (*Response, error) {
	response := &Response{
		Type:       responseType,
		StatusCode: statusCode,
		Headers:    headers,
		Body:       body,
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
