package traffictesting

//go:generate go tool mockgen -destination=../../mocks/traffictesting/mock_request_repository.go -package=mocks github.com/pedrobarco/mroki/internal/domain/traffictesting RequestRepository

import (
	"context"

	"github.com/pedrobarco/mroki/internal/domain/pagination"
)

// RequestRepository defines the contract for request persistence operations
type RequestRepository interface {
	Save(ctx context.Context, request *Request) error
	GetByID(ctx context.Context, id RequestID, gateID GateID) (*Request, error)
	GetAllByGateID(
		ctx context.Context,
		gateID GateID,
		filters RequestFilters,
		sort RequestSort,
		params *pagination.Params,
	) (*pagination.PagedResult[*Request], error)
}
