package queries

import (
	"context"
	"fmt"

	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
)

// GateWithStats composes a Gate aggregate with its read-side statistics.
type GateWithStats struct {
	Gate  *traffictesting.Gate
	Stats traffictesting.GateStats
}

// GetGateQuery represents a query to retrieve a single gate
type GetGateQuery struct {
	ID string
}

// GetGateHandler handles the GetGate query
type GetGateHandler struct {
	repo      traffictesting.GateRepository
	statsRepo traffictesting.StatsRepository
}

// NewGetGateHandler creates a new GetGateHandler
func NewGetGateHandler(repo traffictesting.GateRepository, statsRepo traffictesting.StatsRepository) *GetGateHandler {
	return &GetGateHandler{repo: repo, statsRepo: statsRepo}
}

// Handle executes the GetGate query
func (h *GetGateHandler) Handle(ctx context.Context, q GetGateQuery) (*GateWithStats, error) {
	// Parse and validate ID
	id, err := traffictesting.ParseGateID(q.ID)
	if err != nil {
		return nil, err
	}

	// Retrieve gate aggregate
	gate, err := h.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Fetch stats from read side
	statsMap, err := h.statsRepo.GetStatsByGateIDs(ctx, []traffictesting.GateID{gate.ID})
	if err != nil {
		return nil, fmt.Errorf("failed to get gate stats: %w", err)
	}

	return &GateWithStats{Gate: gate, Stats: statsMap[gate.ID]}, nil
}
