package ent

import (
	"context"
	"fmt"

	"github.com/pedrobarco/mroki/ent"
	"github.com/pedrobarco/mroki/ent/gate"
	"github.com/pedrobarco/mroki/internal/domain/pagination"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
)

type gateRepository struct {
	client *ent.Client
}

var _ traffictesting.GateRepository = (*gateRepository)(nil)

func NewGateRepository(client *ent.Client) *gateRepository {
	return &gateRepository{client: client}
}

func (r *gateRepository) Save(ctx context.Context, g *traffictesting.Gate) error {
	if _, err := r.client.Gate.Create().
		SetID(g.ID.UUID()).
		SetLiveURL(g.LiveURL.String()).
		SetShadowURL(g.ShadowURL.String()).
		Save(ctx); err != nil {
		return fmt.Errorf("failed to save gate: %w", err)
	}
	return nil
}

func (r *gateRepository) GetByID(ctx context.Context, id traffictesting.GateID) (*traffictesting.Gate, error) {
	raw, err := r.client.Gate.Get(ctx, id.UUID())
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("%w: %s", traffictesting.ErrGateNotFound, id)
		}
		return nil, fmt.Errorf("failed to get gate by ID: %w", err)
	}

	return mapGateToDomain(raw)
}

func (r *gateRepository) GetAll(ctx context.Context, params *pagination.Params) (*pagination.PagedResult[*traffictesting.Gate], error) {
	total, err := r.client.Gate.Query().Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count gates: %w", err)
	}

	rows, err := r.client.Gate.Query().
		Order(gate.ByID()).
		Limit(params.Limit()).
		Offset(params.Offset()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gates: %w", err)
	}

	gates := make([]*traffictesting.Gate, 0, len(rows))
	for _, raw := range rows {
		g, err := mapGateToDomain(raw)
		if err != nil {
			return nil, fmt.Errorf("failed to convert gate: %w", err)
		}
		gates = append(gates, g)
	}

	return pagination.NewPagedResult(gates, int64(total), params), nil
}
