package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/pedrobarco/mroki/internal/domain/pagination"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
)

// ListRequestsQuery represents a query to list requests for a gate with pagination, filters, and sorting
type ListRequestsQuery struct {
	// Gate identification
	GateID string

	// Pagination
	Limit  int
	Offset int

	// Filters (all optional, empty/nil means no filter)
	Methods     []string   // HTTP methods to filter by (e.g., ["GET", "POST"])
	PathPattern string     // Path pattern with wildcards (e.g., "/api/users/*")
	FromDate    *time.Time // Filter requests created after this date
	ToDate      *time.Time // Filter requests created before this date
	AgentID     string     // Filter by agent ID
	HasDiff     *bool      // Filter by diff existence (true = has diff, false = no diff, nil = no filter)

	// Sorting (optional, defaults applied if empty)
	SortField string // Field to sort by: "created_at", "method", "path"
	SortOrder string // Sort direction: "asc", "desc"
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

	// Create pagination parameters
	params, err := pagination.NewParams(q.Limit, q.Offset)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", traffictesting.ErrInvalidPagination, err)
	}

	// Create filter value objects explicitly (service layer responsibility)
	filters, err := h.buildFilters(q)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", traffictesting.ErrInvalidFilters, err)
	}

	// Create sort value objects explicitly (service layer responsibility)
	sort, err := h.buildSort(q)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", traffictesting.ErrInvalidSort, err)
	}

	// Retrieve from repository with filters and sort
	return h.repo.GetAllByGateID(ctx, gateID, filters, sort, params)
}

// buildFilters creates RequestFilters value object from query primitives
func (h *ListRequestsHandler) buildFilters(q ListRequestsQuery) (traffictesting.RequestFilters, error) {
	// Create HTTPMethod value objects
	methods := make([]traffictesting.HTTPMethod, 0, len(q.Methods))
	for _, methodStr := range q.Methods {
		method, err := traffictesting.NewHTTPMethod(methodStr)
		if err != nil {
			return traffictesting.RequestFilters{}, fmt.Errorf("invalid HTTP method %q: %w", methodStr, err)
		}
		methods = append(methods, method)
	}

	// Create PathPattern value object
	pathPattern, err := traffictesting.NewPathPattern(q.PathPattern)
	if err != nil {
		return traffictesting.RequestFilters{}, fmt.Errorf("invalid path pattern: %w", err)
	}

	// Create DateRange value object
	dateRange, err := traffictesting.NewDateRange(q.FromDate, q.ToDate)
	if err != nil {
		return traffictesting.RequestFilters{}, fmt.Errorf("invalid date range: %w", err)
	}

	// Compose RequestFilters (agentID and hasDiff are plain types)
	return traffictesting.NewRequestFilters(methods, pathPattern, dateRange, q.AgentID, q.HasDiff), nil
}

// buildSort creates RequestSort value object from query primitives
func (h *ListRequestsHandler) buildSort(q ListRequestsQuery) (traffictesting.RequestSort, error) {
	// Create SortField value object
	sortField, err := traffictesting.NewRequestSortField(q.SortField)
	if err != nil {
		return traffictesting.RequestSort{}, fmt.Errorf("invalid sort field: %w", err)
	}

	// Create SortOrder value object
	sortOrder, err := traffictesting.NewSortOrder(q.SortOrder)
	if err != nil {
		return traffictesting.RequestSort{}, fmt.Errorf("invalid sort order: %w", err)
	}

	// Compose RequestSort
	return traffictesting.NewRequestSort(sortField, sortOrder), nil
}
