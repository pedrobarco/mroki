package traffictesting

//go:generate go tool mockgen -destination=../../mocks/traffictesting/mock_gate_repository.go -package=mocks github.com/pedrobarco/mroki/internal/domain/traffictesting GateRepository

import (
	"context"

	"github.com/pedrobarco/mroki/internal/domain/pagination"
)

// GateRepository defines the contract for gate persistence operations
type GateRepository interface {
	Save(ctx context.Context, gate *Gate) error
	GetByID(ctx context.Context, id GateID) (*Gate, error)
	GetAll(ctx context.Context, params *pagination.Params) (*pagination.PagedResult[*Gate], error)
}
