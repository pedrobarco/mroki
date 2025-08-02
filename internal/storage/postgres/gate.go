package postgres

import (
	"context"
	"fmt"
	"net/url"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pedrobarco/mroki/internal/domain/diffing"
	"github.com/pedrobarco/mroki/internal/storage/postgres/db"
)

type gateRepository struct {
	queries *db.Queries
}

var _ diffing.GateRepository = (*gateRepository)(nil)

func NewGateRepository(queries *db.Queries) *gateRepository {
	return &gateRepository{
		queries: queries,
	}
}

func (g *gateRepository) GetAll(ctx context.Context) ([]*diffing.Gate, error) {
	gates, err := g.queries.GetAllGates(ctx)
	if err != nil {
		return nil, err
	}

	var result []*diffing.Gate
	for _, raw := range gates {
		gate, err := g.toDomain(raw)
		if err != nil {
			return nil, fmt.Errorf("failed to convert gate: %w", err)
		}
		result = append(result, gate)
	}
	return result, nil
}

func (g *gateRepository) GetByID(ctx context.Context, id uuid.UUID) (*diffing.Gate, error) {
	pid := pgtype.UUID{
		Bytes: id,
		Valid: true,
	}

	raw, err := g.queries.GetGateByID(ctx, pid)
	if err != nil {
		return nil, fmt.Errorf("failed to get gate by ID: %w", err)
	}

	gate, err := g.toDomain(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to convert gate: %w", err)
	}

	return gate, nil
}

func (g *gateRepository) Save(ctx context.Context, gate *diffing.Gate) error {
	if err := g.queries.SaveGate(ctx, db.SaveGateParams{
		ID:        pgtype.UUID{Bytes: gate.ID, Valid: true},
		LiveUrl:   pgtype.Text{String: gate.LiveURL.String(), Valid: true},
		ShadowUrl: pgtype.Text{String: gate.ShadowURL.String(), Valid: true},
	}); err != nil {
		return fmt.Errorf("failed to save gate: %w", err)
	}
	return nil
}

func (g *gateRepository) toDomain(raw db.Gate) (*diffing.Gate, error) {
	live, err := url.Parse(raw.LiveUrl.String)
	if err != nil {
		return nil, err
	}

	shadow, err := url.Parse(raw.ShadowUrl.String)
	if err != nil {
		return nil, err
	}

	gate, err := diffing.NewGate(live, shadow)
	if err != nil {
		return nil, err
	}

	return gate, nil
}
