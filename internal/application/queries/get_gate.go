package queries

import (
	"context"

	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
)

// GetGateQuery represents a query to retrieve a single gate
type GetGateQuery struct {
	ID string
}

// GetGateHandler handles the GetGate query
type GetGateHandler struct {
	repo traffictesting.GateRepository
}

// NewGetGateHandler creates a new GetGateHandler
func NewGetGateHandler(repo traffictesting.GateRepository) *GetGateHandler {
	return &GetGateHandler{repo: repo}
}

// Handle executes the GetGate query
func (h *GetGateHandler) Handle(ctx context.Context, q GetGateQuery) (*traffictesting.Gate, error) {
	// Parse and validate ID
	id, err := traffictesting.ParseGateID(q.ID)
	if err != nil {
		return nil, err
	}

	// Retrieve from repository
	return h.repo.GetByID(ctx, id)
}
