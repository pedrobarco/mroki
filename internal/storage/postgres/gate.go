package postgres

import (
	"context"
	"fmt"
	"net/url"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

func (r *gateRepository) GetAll(ctx context.Context) ([]*diffing.Gate, error) {
	rows, err := r.queries.GetAllGates(ctx)
	if err != nil {
		return nil, err
	}

	var gates []*diffing.Gate
	for _, raw := range rows {
		gate, err := r.toDomain(raw)
		if err != nil {
			return nil, fmt.Errorf("failed to convert gate: %w", err)
		}
		gates = append(gates, gate)
	}
	return gates, nil
}

func (r *gateRepository) GetByID(ctx context.Context, id uuid.UUID) (*diffing.Gate, error) {
	pid := pgtype.UUID{
		Bytes: id,
		Valid: true,
	}

	raw, err := r.queries.GetGateByID(ctx, pid)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("%w: %s", diffing.ErrGateNotFound, id)
		}
		return nil, fmt.Errorf("failed to get gate by ID: %w", err)
	}

	gate, err := r.toDomain(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to convert gate: %w", err)
	}

	return gate, nil
}

func (r *gateRepository) Save(ctx context.Context, gate *diffing.Gate) error {
	if err := r.queries.SaveGate(ctx, db.SaveGateParams{
		ID:        pgtype.UUID{Bytes: gate.ID, Valid: true},
		LiveUrl:   pgtype.Text{String: gate.LiveURL.String(), Valid: true},
		ShadowUrl: pgtype.Text{String: gate.ShadowURL.String(), Valid: true},
	}); err != nil {
		return fmt.Errorf("failed to save gate: %w", err)
	}
	return nil
}

func (r *gateRepository) toDomain(raw db.Gate) (*diffing.Gate, error) {
	live, err := url.Parse(raw.LiveUrl.String)
	if err != nil {
		return nil, fmt.Errorf("invalid live URL: %w", err)
	}

	shadow, err := url.Parse(raw.ShadowUrl.String)
	if err != nil {
		return nil, fmt.Errorf("invalid shadow URL: %w", err)
	}

	id, err := uuid.Parse(raw.ID.String())
	if err != nil {
		return nil, fmt.Errorf("invalid UUID format: %w", err)
	}

	gate, err := diffing.NewGate(live, shadow, diffing.WithGateID(id))
	if err != nil {
		return nil, fmt.Errorf("failed to create gate: %w", err)
	}

	return gate, nil
}
