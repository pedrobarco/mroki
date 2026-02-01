package queries

import (
	"context"
	"fmt"

	"github.com/pedrobarco/mroki/internal/domain/pagination"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
)

// ListGatesQuery represents a query to list gates with pagination
type ListGatesQuery struct {
	Limit  int
	Offset int
}

// ListGatesHandler handles the ListGates query
type ListGatesHandler struct {
	repo traffictesting.GateRepository
}

// NewListGatesHandler creates a new ListGatesHandler
func NewListGatesHandler(repo traffictesting.GateRepository) *ListGatesHandler {
	return &ListGatesHandler{repo: repo}
}

// Handle executes the ListGates query
func (h *ListGatesHandler) Handle(ctx context.Context, q ListGatesQuery) (*pagination.PagedResult[*traffictesting.Gate], error) {
	// Create and validate pagination parameters
	params, err := pagination.NewParams(q.Limit, q.Offset)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", traffictesting.ErrInvalidPagination, err)
	}

	// Retrieve from repository
	return h.repo.GetAll(ctx, params)
}
