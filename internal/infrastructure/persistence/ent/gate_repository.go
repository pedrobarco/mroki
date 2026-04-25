package ent

import (
	"context"
	"fmt"
	"strings"

	"entgo.io/ent/dialect/sql"
	"github.com/pedrobarco/mroki/ent"
	"github.com/pedrobarco/mroki/ent/gate"
	"github.com/pedrobarco/mroki/ent/predicate"
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
		SetName(g.Name.String()).
		SetLiveURL(g.LiveURL.String()).
		SetShadowURL(g.ShadowURL.String()).
		SetCreatedAt(g.CreatedAt).
		Save(ctx); err != nil {
		if isUniqueConstraintError(err) {
			return classifyGateUniqueViolation(err, g)
		}
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

	g, err := mapGateToDomain(raw)
	if err != nil {
		return nil, err
	}

	return g, nil
}

func (r *gateRepository) GetAll(ctx context.Context, filters traffictesting.GateFilters, sort traffictesting.GateSort, params *pagination.Params) (*pagination.PagedResult[*traffictesting.Gate], error) {
	// Build predicates — shared between count and list
	preds := r.buildPredicates(filters)

	query := r.client.Gate.Query()
	if len(preds) > 0 {
		query = query.Where(gate.And(preds...))
	}

	total, err := query.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count gates: %w", err)
	}

	listQuery := r.client.Gate.Query()
	if len(preds) > 0 {
		listQuery = listQuery.Where(gate.And(preds...))
	}

	rows, err := listQuery.
		Order(r.buildOrderBy(sort)).
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

// buildPredicates composes Ent predicates from domain gate filters.
func (r *gateRepository) buildPredicates(filters traffictesting.GateFilters) []predicate.Gate {
	var preds []predicate.Gate

	if filters.HasNameFilter() {
		preds = append(preds, gate.NameContainsFold(filters.Name()))
	}

	if filters.HasLiveURLFilter() {
		preds = append(preds, gate.LiveURLContainsFold(filters.LiveURL()))
	}

	if filters.HasShadowURLFilter() {
		preds = append(preds, gate.ShadowURLContainsFold(filters.ShadowURL()))
	}

	return preds
}

// buildOrderBy maps domain sort to an Ent order option.
func (r *gateRepository) buildOrderBy(sort traffictesting.GateSort) gate.OrderOption {
	var opts []sql.OrderTermOption
	if sort.Order().IsAsc() {
		opts = append(opts, sql.OrderAsc())
	} else {
		opts = append(opts, sql.OrderDesc())
	}

	field := sort.Field()
	switch {
	case field.IsName():
		return gate.ByName(opts...)
	case field.IsLiveURL():
		return gate.ByLiveURL(opts...)
	case field.IsShadowURL():
		return gate.ByShadowURL(opts...)
	case field.IsCreatedAt():
		return gate.ByCreatedAt(opts...)
	default:
		return gate.ByID(opts...)
	}
}

// isUniqueConstraintError checks if an error is a unique constraint violation.
func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	return ent.IsConstraintError(err)
}

// classifyGateUniqueViolation maps a unique constraint error to the appropriate domain error
// by inspecting the error message for constraint/column names.
func classifyGateUniqueViolation(err error, g *traffictesting.Gate) error {
	msg := err.Error()
	if strings.Contains(msg, "name") {
		return fmt.Errorf("%w: %s", traffictesting.ErrDuplicateGateName, g.Name)
	}
	if strings.Contains(msg, "live_url") || strings.Contains(msg, "shadow_url") {
		return fmt.Errorf("%w: %s -> %s", traffictesting.ErrDuplicateGateURLs, g.LiveURL, g.ShadowURL)
	}
	// Fallback to name error for unrecognized constraint
	return fmt.Errorf("%w: %s", traffictesting.ErrDuplicateGateName, g.Name)
}
