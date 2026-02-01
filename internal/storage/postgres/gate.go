package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pedrobarco/mroki/internal/domain/diffing"
	"github.com/pedrobarco/mroki/internal/domain/pagination"
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

func (r *gateRepository) GetAll(ctx context.Context, params *pagination.Params) (*pagination.PagedResult[*diffing.Gate], error) {
	// Get total count
	total, err := r.queries.CountGates(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count gates: %w", err)
	}

	// Get paginated results using getters
	rows, err := r.queries.GetAllGates(ctx, db.GetAllGatesParams{
		Limit:  int32(params.Limit()),
		Offset: int32(params.Offset()),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get gates: %w", err)
	}

	// Handle empty results - return empty slice
	gates := make([]*diffing.Gate, 0, len(rows))
	for _, raw := range rows {
		gate, err := r.toDomain(raw)
		if err != nil {
			return nil, fmt.Errorf("failed to convert gate: %w", err)
		}
		gates = append(gates, gate)
	}

	// Use factory to create PagedResult
	return pagination.NewPagedResult(gates, total, params), nil
}

func (r *gateRepository) GetByID(ctx context.Context, id diffing.GateID) (*diffing.Gate, error) {
	pid := pgtype.UUID{
		Bytes: id.UUID(),
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
		ID:        pgtype.UUID{Bytes: gate.ID.UUID(), Valid: true},
		LiveUrl:   pgtype.Text{String: gate.LiveURL.String(), Valid: true},
		ShadowUrl: pgtype.Text{String: gate.ShadowURL.String(), Valid: true},
	}); err != nil {
		return fmt.Errorf("failed to save gate: %w", err)
	}
	return nil
}

func (r *gateRepository) toDomain(raw db.Gate) (*diffing.Gate, error) {
	live, err := diffing.ParseGateURL(raw.LiveUrl.String)
	if err != nil {
		return nil, fmt.Errorf("invalid live URL in database: %w", err)
	}

	shadow, err := diffing.ParseGateURL(raw.ShadowUrl.String)
	if err != nil {
		return nil, fmt.Errorf("invalid shadow URL in database: %w", err)
	}

	id, err := diffing.ParseGateID(raw.ID.String())
	if err != nil {
		return nil, fmt.Errorf("invalid gate ID in database: %w", err)
	}

	gate, err := diffing.NewGate(live, shadow, diffing.WithGateID(id))
	if err != nil {
		return nil, fmt.Errorf("failed to create gate: %w", err)
	}

	return gate, nil
}
