package traffictesting

//go:generate go tool mockgen -destination=../../mocks/traffictesting/mock_gate_repository.go -package=mocks github.com/pedrobarco/mroki/internal/domain/traffictesting GateRepository

import (
	"context"

	"github.com/pedrobarco/mroki/internal/domain/pagination"
)

// GateRepository defines the contract for gate persistence operations
type GateRepository interface {
	Save(ctx context.Context, gate *Gate) error
	Update(ctx context.Context, gate *Gate) error
	Delete(ctx context.Context, id GateID) error
	GetByID(ctx context.Context, id GateID) (*Gate, error)
	GetAll(ctx context.Context, filters GateFilters, sort GateSort, params *pagination.Params) (*pagination.PagedResult[*Gate], error)
}
