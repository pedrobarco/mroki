package diffing

import (
	"context"
	"fmt"
	"net/url"

	"github.com/google/uuid"
)

type GateRepository interface {
	Save(ctx context.Context, gate *Gate) error
	GetByID(ctx context.Context, id uuid.UUID) (*Gate, error)
	GetAll(ctx context.Context) ([]*Gate, error)
}

type GateService struct {
	repo GateRepository
}

func NewGateService(repo GateRepository) *GateService {
	return &GateService{
		repo: repo,
	}
}

func (s *GateService) Create(ctx context.Context, live, shadow string) (*Gate, error) {
	liveURL, err := url.Parse(live)
	if err != nil {
		return nil, fmt.Errorf("invalid live URL: %w", err)
	}

	shadowURL, err := url.Parse(shadow)
	if err != nil {
		return nil, fmt.Errorf("invalid shadow URL: %w", err)
	}

	gate, err := NewGate(liveURL, shadowURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create gate: %w", err)
	}

	if err := s.repo.Save(ctx, gate); err != nil {
		return nil, err
	}

	return gate, nil
}

func (s *GateService) GetByID(ctx context.Context, id uuid.UUID) (*Gate, error) {
	gate, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get gate by ID: %w", err)
	}
	return gate, nil
}

func (s *GateService) GetAll(ctx context.Context) ([]*Gate, error) {
	gates, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all gates: %w", err)
	}
	return gates, nil
}
