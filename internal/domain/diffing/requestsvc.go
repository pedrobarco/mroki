package diffing

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type RequestRepository interface {
	Save(ctx context.Context, request *Request) error
	GetByID(ctx context.Context, id uuid.UUID, gateID uuid.UUID) (*Request, error)
	GetAllByGateID(ctx context.Context, gateID uuid.UUID) ([]*Request, error)
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
	ID        uuid.UUID
	GateID    uuid.UUID
	Method    string
	Path      string
	Headers   map[string][]string
	Body      []byte
	CreatedAt time.Time
}

func (s *RequestService) Create(ctx context.Context, props CreateRequestProps) (*Request, error) {
	var opts []requestOption
	if props.ID != uuid.Nil {
		opts = append(opts, WithRequestID(props.ID))
	}

	request, err := NewRequest(
		props.GateID,
		props.Method,
		props.Path,
		props.Headers,
		props.Body,
		props.CreatedAt,
		opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if err := s.repo.Save(ctx, request); err != nil {
		return nil, fmt.Errorf("failed to save request: %w", err)
	}

	return request, nil
}

func (s *RequestService) GetByID(ctx context.Context, id uuid.UUID, gateID uuid.UUID) (*Request, error) {
	request, err := s.repo.GetByID(ctx, id, gateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get request by ID: %w", err)
	}
	return request, nil
}

func (s *RequestService) GetAllByGateID(ctx context.Context, gateID uuid.UUID) ([]*Request, error) {
	requests, err := s.repo.GetAllByGateID(ctx, gateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get all requests: %w", err)
	}
	return requests, nil
}
