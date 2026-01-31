package diffing

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type RequestRepository interface {
	Save(ctx context.Context, request *Request) error
	GetByID(ctx context.Context, id RequestID, gateID GateID) (*Request, error)
	GetAllByGateID(ctx context.Context, gateID GateID) ([]*Request, error)
}

type RequestService struct {
	repo RequestRepository
}

func NewRequestService(repo RequestRepository) *RequestService {
	return &RequestService{
		repo: repo,
	}
}

type CreateRequestProps struct {
	ID        RequestID
	GateID    GateID
	Method    string
	Path      string
	Headers   map[string][]string
	Body      []byte
	CreatedAt time.Time

	Responses []CreateRequestResponseProps
	Diff      CreateRequestDiffProps
}

type CreateRequestResponseProps struct {
	ID         uuid.UUID
	Type       string
	StatusCode int
	Headers    http.Header
	Body       []byte
	CreatedAt  time.Time
}

type CreateRequestDiffProps struct {
	Content string
}

func (s *RequestService) Create(ctx context.Context, props CreateRequestProps) (*Request, error) {
	var live, shadow *Response
	var responses []Response
	for _, dto := range props.Responses {
		rtype, err := NewResponseType(dto.Type)
		if err != nil {
			return nil, fmt.Errorf("invalid response type: %w", err)
		}

		resp, err := NewResponse(
			rtype,
			dto.StatusCode,
			dto.Headers,
			dto.Body,
			dto.CreatedAt,
			WithResponseID(dto.ID),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create response: %w", err)
		}
		responses = append(responses, *resp)

		switch resp.Type {
		case ResponseTypeLive:
			live = resp
		case ResponseTypeShadow:
			shadow = resp
		}
	}

	if len(responses) != 2 {
		return nil, fmt.Errorf("exactly two responses are required, got %d", len(responses))
	}

	if live == nil {
		return nil, fmt.Errorf("live response is required")
	}

	if shadow == nil {
		return nil, fmt.Errorf("shadow response is required")
	}

	diff, err := NewDiff(live.ID, shadow.ID, props.Diff.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to create diff: %w", err)
	}

	request, err := NewRequest(
		props.GateID,
		props.Method,
		props.Path,
		props.Headers,
		props.Body,
		props.CreatedAt,
		responses,
		*diff,
		WithRequestID(props.ID),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if err := s.repo.Save(ctx, request); err != nil {
		return nil, fmt.Errorf("failed to save request: %w", err)
	}

	return request, nil
}

func (s *RequestService) GetByID(ctx context.Context, idStr string, gateIDStr string) (*Request, error) {
	id, err := ParseRequestID(idStr)
	if err != nil {
		return nil, err
	}

	gateID, err := ParseGateID(gateIDStr)
	if err != nil {
		return nil, err
	}

	request, err := s.repo.GetByID(ctx, id, gateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get request by ID: %w", err)
	}
	return request, nil
}

func (s *RequestService) GetAllByGateID(ctx context.Context, gateIDStr string) ([]*Request, error) {
	gateID, err := ParseGateID(gateIDStr)
	if err != nil {
		return nil, err
	}

	requests, err := s.repo.GetAllByGateID(ctx, gateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get all requests: %w", err)
	}
	return requests, nil
}
