package queries

import (
	"context"
	"fmt"

	"github.com/pedrobarco/mroki/internal/domain/pagination"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
)

// ListGatesQuery represents a query to list gates with pagination, filters, and sorting
type ListGatesQuery struct {
	// Pagination
	Limit  int
	Offset int

	// Filters (all optional, empty means no filter)
	LiveURL   string // Substring match on live URL
	ShadowURL string // Substring match on shadow URL

	// Sorting (optional, defaults applied if empty)
	SortField string // Field to sort by: "id", "live_url", "shadow_url"
	SortOrder string // Sort direction: "asc", "desc"
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

	// Create filter value objects
	filters := traffictesting.NewGateFilters(q.LiveURL, q.ShadowURL)

	// Create sort value objects
	sort, err := h.buildSort(q)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", traffictesting.ErrInvalidGateSort, err)
	}

	// Retrieve from repository with filters and sort
	return h.repo.GetAll(ctx, filters, sort, params)
}

// buildSort creates GateSort value object from query primitives
func (h *ListGatesHandler) buildSort(q ListGatesQuery) (traffictesting.GateSort, error) {
	// Create SortField value object
	sortField, err := traffictesting.NewGateSortField(q.SortField)
	if err != nil {
		return traffictesting.GateSort{}, fmt.Errorf("invalid sort field: %w", err)
	}

	// Create SortOrder value object
	sortOrder, err := traffictesting.NewSortOrder(q.SortOrder)
	if err != nil {
		return traffictesting.GateSort{}, fmt.Errorf("invalid sort order: %w", err)
	}

	// Compose GateSort
	return traffictesting.NewGateSort(sortField, sortOrder), nil
}
