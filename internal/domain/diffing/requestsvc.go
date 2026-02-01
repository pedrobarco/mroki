package diffing

//go:generate go tool mockgen -destination=mocks/mock_request_repository.go -package=mocks github.com/pedrobarco/mroki/internal/domain/diffing RequestRepository

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/pedrobarco/mroki/internal/domain/pagination"
)

type RequestRepository interface {
	Save(ctx context.Context, request *Request) error
	GetByID(ctx context.Context, id RequestID, gateID GateID) (*Request, error)
	GetAllByGateID(ctx context.Context, gateID GateID, params *pagination.Params) (*pagination.PagedResult[*Request], error)
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
	ID        string
	GateID    string
	AgentID   string
	Method    string
	Path      string
	Headers   map[string][]string
	Body      []byte
	CreatedAt time.Time

	Responses []CreateRequestResponseProps
	Diff      CreateRequestDiffProps
}

type CreateRequestResponseProps struct {
	ID         string
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
	// Parse gate ID
	gateID, err := ParseGateID(props.GateID)
	if err != nil {
		return nil, err
	}

	// Parse request ID (optional)
	var requestID RequestID
	if props.ID != "" {
		requestID, err = ParseRequestID(props.ID)
		if err != nil {
			return nil, err
		}
	}

	// Parse agent ID (optional)
	var agentID AgentID
	if props.AgentID != "" {
		agentID, err = ParseAgentID(props.AgentID)
		if err != nil {
			return nil, err
		}
	}

	// Parse and create responses
	var live, shadow *Response
	var responses []Response
	for _, dto := range props.Responses {
		rtype, err := NewResponseType(dto.Type)
		if err != nil {
			return nil, fmt.Errorf("invalid response type: %w", err)
		}

		// Parse response ID (optional)
		var respID uuid.UUID
		if dto.ID != "" {
			respID, err = uuid.Parse(dto.ID)
			if err != nil {
				return nil, fmt.Errorf("invalid response ID: %w", err)
			}
		}

		resp, err := NewResponse(
			rtype,
			dto.StatusCode,
			dto.Headers,
			dto.Body,
			dto.CreatedAt,
			WithResponseID(respID),
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

	// Create request with parsed gate ID
	request, err := NewRequest(
		gateID,
		props.Method,
		props.Path,
		props.Headers,
		props.Body,
		props.CreatedAt,
		responses,
		*diff,
		WithRequestID(requestID),
		WithAgentID(agentID),
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

// GetAllByGateID retrieves all requests for a gate with pagination
// Accepts optional limit and offset parameters (0 or negative values use defaults)
// Returns PagedResult containing requests, total count, and pagination metadata
func (s *RequestService) GetAllByGateID(ctx context.Context, gateIDStr string, limit, offset int) (*pagination.PagedResult[*Request], error) {
	// Parse gate ID first (domain validation)
	gateID, err := ParseGateID(gateIDStr)
	if err != nil {
		return nil, err
	}

	// Service is responsible for creating the pagination value object
	params, err := pagination.NewParams(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPagination, err)
	}

	result, err := s.repo.GetAllByGateID(ctx, gateID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get all requests: %w", err)
	}

	return result, nil
}
