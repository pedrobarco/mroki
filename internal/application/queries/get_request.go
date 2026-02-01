package queries

import (
	"context"

	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
)

// GetRequestQuery represents a query to retrieve a single request
type GetRequestQuery struct {
	ID     string
	GateID string
}

// GetRequestHandler handles the GetRequest query
type GetRequestHandler struct {
	repo traffictesting.RequestRepository
}

// NewGetRequestHandler creates a new GetRequestHandler
func NewGetRequestHandler(repo traffictesting.RequestRepository) *GetRequestHandler {
	return &GetRequestHandler{repo: repo}
}

// Handle executes the GetRequest query
func (h *GetRequestHandler) Handle(ctx context.Context, q GetRequestQuery) (*traffictesting.Request, error) {
	// Parse and validate IDs
	id, err := traffictesting.ParseRequestID(q.ID)
	if err != nil {
		return nil, err
	}

	gateID, err := traffictesting.ParseGateID(q.GateID)
	if err != nil {
		return nil, err
	}

	// Retrieve from repository
	return h.repo.GetByID(ctx, id, gateID)
}
