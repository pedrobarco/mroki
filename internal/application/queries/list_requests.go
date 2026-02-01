package queries

import (
	"context"
	"fmt"

	"github.com/pedrobarco/mroki/internal/domain/pagination"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
)

// ListRequestsQuery represents a query to list requests for a gate with pagination
type ListRequestsQuery struct {
	GateID string
	Limit  int
	Offset int
}

// ListRequestsHandler handles the ListRequests query
type ListRequestsHandler struct {
	repo traffictesting.RequestRepository
}

// NewListRequestsHandler creates a new ListRequestsHandler
func NewListRequestsHandler(repo traffictesting.RequestRepository) *ListRequestsHandler {
	return &ListRequestsHandler{repo: repo}
}

// Handle executes the ListRequests query
func (h *ListRequestsHandler) Handle(ctx context.Context, q ListRequestsQuery) (*pagination.PagedResult[*traffictesting.Request], error) {
	// Parse and validate gate ID
	gateID, err := traffictesting.ParseGateID(q.GateID)
	if err != nil {
		return nil, err
	}

	// Create and validate pagination parameters
	params, err := pagination.NewParams(q.Limit, q.Offset)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", traffictesting.ErrInvalidPagination, err)
	}

	// Retrieve from repository
	return h.repo.GetAllByGateID(ctx, gateID, params)
}
