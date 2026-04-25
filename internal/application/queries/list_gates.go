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
	Name      string // Substring match on name
	LiveURL   string // Substring match on live URL
	ShadowURL string // Substring match on shadow URL

	// Sorting (optional, defaults applied if empty)
	SortField string // Field to sort by: "id", "name", "live_url", "shadow_url", "created_at"
	SortOrder string // Sort direction: "asc", "desc"
}

// ListGatesHandler handles the ListGates query
type ListGatesHandler struct {
	repo      traffictesting.GateRepository
	statsRepo traffictesting.StatsRepository
}

// NewListGatesHandler creates a new ListGatesHandler
func NewListGatesHandler(repo traffictesting.GateRepository, statsRepo traffictesting.StatsRepository) *ListGatesHandler {
	return &ListGatesHandler{repo: repo, statsRepo: statsRepo}
}

// Handle executes the ListGates query
func (h *ListGatesHandler) Handle(ctx context.Context, q ListGatesQuery) (*pagination.PagedResult[*GateWithStats], error) {
	// Create and validate pagination parameters
	params, err := pagination.NewParams(q.Limit, q.Offset)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", traffictesting.ErrInvalidPagination, err)
	}

	// Create filter value objects
	filters := traffictesting.NewGateFilters(q.Name, q.LiveURL, q.ShadowURL)

	// Create sort value objects
	sort, err := h.buildSort(q)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", traffictesting.ErrInvalidGateSort, err)
	}

	// Retrieve gate aggregates
	result, err := h.repo.GetAll(ctx, filters, sort, params)
	if err != nil {
		return nil, err
	}

	// Collect gate IDs for batch stats fetch
	gateIDs := make([]traffictesting.GateID, len(result.Items))
	for i, g := range result.Items {
		gateIDs[i] = g.ID
	}

	statsMap, err := h.statsRepo.GetStatsByGateIDs(ctx, gateIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get gate stats: %w", err)
	}

	// Compose GateWithStats items
	items := make([]*GateWithStats, len(result.Items))
	for i, g := range result.Items {
		items[i] = &GateWithStats{Gate: g, Stats: statsMap[g.ID]}
	}

	return pagination.NewPagedResult(items, result.Total, params), nil
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
