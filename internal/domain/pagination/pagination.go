package pagination

import "fmt"

const (
	// defaultLimit is the default number of items per page
	defaultLimit = 50
	// maxLimit is the maximum allowed number of items per page
	maxLimit = 100
	// minLimit is the minimum number of items per page
	minLimit = 1
)

// Params is an immutable value object representing pagination parameters
type Params struct {
	limit  int
	offset int
}

// NewParams creates new pagination parameters with validation
// Pass 0 or negative values to use defaults
func NewParams(limit, offset int) (*Params, error) {
	// Apply default if limit is not set or invalid
	if limit < minLimit {
		limit = defaultLimit
	}

	// Enforce maximum limit
	if limit > maxLimit {
		limit = maxLimit
	}

	// Ensure non-negative offset
	if offset < 0 {
		offset = 0
	}

	return &Params{
		limit:  limit,
		offset: offset,
	}, nil
}

// Limit returns the limit value (immutable)
func (p *Params) Limit() int {
	return p.limit
}

// Offset returns the offset value (immutable)
func (p *Params) Offset() int {
	return p.offset
}

// Validate ensures pagination parameters are valid
func (p *Params) Validate() error {
	if p.limit < minLimit {
		return fmt.Errorf("limit must be at least %d", minLimit)
	}
	if p.limit > maxLimit {
		return fmt.Errorf("limit must not exceed %d", maxLimit)
	}
	if p.offset < 0 {
		return fmt.Errorf("offset must be non-negative")
	}
	return nil
}

// PagedResult encapsulates paginated query results
type PagedResult[T any] struct {
	Items   []T
	Total   int64
	Limit   int
	Offset  int
	HasMore bool
}

// NewPagedResult creates a paged result from raw data
func NewPagedResult[T any](items []T, total int64, params *Params) *PagedResult[T] {
	// Ensure items is never nil for consistent JSON serialization
	if items == nil {
		items = []T{}
	}

	hasMore := int64(params.Offset()+params.Limit()) < total

	return &PagedResult[T]{
		Items:   items,
		Total:   total,
		Limit:   params.Limit(),
		Offset:  params.Offset(),
		HasMore: hasMore,
	}
}
