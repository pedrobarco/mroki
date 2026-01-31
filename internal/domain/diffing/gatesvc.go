package diffing

//go:generate go tool mockgen -destination=mocks/mock_gate_repository.go -package=mocks github.com/pedrobarco/mroki/internal/domain/diffing GateRepository

import (
	"context"
	"fmt"
)

type GateRepository interface {
	Save(ctx context.Context, gate *Gate) error
	GetByID(ctx context.Context, id GateID) (*Gate, error)
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
	liveURL, err := ParseGateURL(live)
	if err != nil {
		return nil, err
	}

	shadowURL, err := ParseGateURL(shadow)
	if err != nil {
		return nil, err
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

func (s *GateService) GetByID(ctx context.Context, idStr string) (*Gate, error) {
	id, err := ParseGateID(idStr)
	if err != nil {
		return nil, err
	}

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
