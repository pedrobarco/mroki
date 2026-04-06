package ent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/pedrobarco/mroki/ent"
	"github.com/pedrobarco/mroki/ent/gate"
	"github.com/pedrobarco/mroki/ent/predicate"
	"github.com/pedrobarco/mroki/ent/request"
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

	if err := r.attachStats(ctx, []*traffictesting.Gate{g}); err != nil {
		return nil, fmt.Errorf("failed to attach gate stats: %w", err)
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

	if err := r.attachStats(ctx, gates); err != nil {
		return nil, fmt.Errorf("failed to attach gate stats: %w", err)
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


// gateStatRow holds a grouped aggregation result for a gate.
type gateStatRow struct {
	GateID uuid.UUID `json:"gate_id"`
	Count  int       `json:"count"`
}

// gateLastActiveRow holds the last active timestamp for a gate.
// LastActive is a string because ent's Scan with aggregation returns
// time values as strings (especially with SQLite).
type gateLastActiveRow struct {
	GateID     uuid.UUID `json:"gate_id"`
	LastActive string    `json:"last_active"`
}

// attachStats runs batch aggregation queries and populates Stats on each gate.
func (r *gateRepository) attachStats(ctx context.Context, gates []*traffictesting.Gate) error {
	if len(gates) == 0 {
		return nil
	}

	gateIDs := make([]uuid.UUID, len(gates))
	for i, g := range gates {
		gateIDs[i] = g.ID.UUID()
	}
	since24h := time.Now().Add(-24 * time.Hour)

	// Query 1: request_count_24h per gate
	var requestCounts []gateStatRow
	err := r.client.Request.Query().
		Where(request.GateIDIn(gateIDs...), request.CreatedAtGTE(since24h)).
		GroupBy(request.FieldGateID).
		Aggregate(ent.Count()).
		Scan(ctx, &requestCounts)
	if err != nil {
		return fmt.Errorf("failed to query request counts: %w", err)
	}

	// Query 2: diff_count_24h per gate (requests that have a diff)
	var diffCounts []gateStatRow
	err = r.client.Request.Query().
		Where(request.GateIDIn(gateIDs...), request.CreatedAtGTE(since24h), request.HasDiff()).
		GroupBy(request.FieldGateID).
		Aggregate(ent.Count()).
		Scan(ctx, &diffCounts)
	if err != nil {
		return fmt.Errorf("failed to query diff counts: %w", err)
	}

	// Query 3: last_active per gate (no time filter)
	var lastActives []gateLastActiveRow
	err = r.client.Request.Query().
		Where(request.GateIDIn(gateIDs...)).
		GroupBy(request.FieldGateID).
		Aggregate(ent.As(ent.Max(request.FieldCreatedAt), "last_active")).
		Scan(ctx, &lastActives)
	if err != nil {
		return fmt.Errorf("failed to query last active: %w", err)
	}

	// Build lookup maps
	reqCountMap := make(map[uuid.UUID]int64, len(requestCounts))
	for _, r := range requestCounts {
		reqCountMap[r.GateID] = int64(r.Count)
	}
	diffCountMap := make(map[uuid.UUID]int64, len(diffCounts))
	for _, r := range diffCounts {
		diffCountMap[r.GateID] = int64(r.Count)
	}
	lastActiveMap := make(map[uuid.UUID]time.Time, len(lastActives))
	for _, r := range lastActives {
		if r.LastActive == "" {
			continue
		}
		// Try common time formats (RFC3339, then SQLite/ent default)
		t, err := time.Parse(time.RFC3339Nano, r.LastActive)
		if err != nil {
			t, err = time.Parse("2006-01-02T15:04:05Z", r.LastActive)
		}
		if err != nil {
			t, err = time.Parse("2006-01-02 15:04:05-07:00", r.LastActive)
		}
		if err != nil {
			t, err = time.Parse("2006-01-02 15:04:05+00:00", r.LastActive)
		}
		if err == nil {
			lastActiveMap[r.GateID] = t
		}
	}

	// Attach stats to each gate
	for _, g := range gates {
		id := g.ID.UUID()
		reqCount := reqCountMap[id]
		diffCount := diffCountMap[id]

		var diffRate float64
		if reqCount > 0 {
			diffRate = float64(diffCount) / float64(reqCount) * 100
		}

		var lastActive *time.Time
		if t, ok := lastActiveMap[id]; ok {
			lastActive = &t
		}

		g.Stats = traffictesting.GateStats{
			RequestCount24h: reqCount,
			DiffCount24h:    diffCount,
			DiffRate:        diffRate,
			LastActive:      lastActive,
		}
	}

	return nil
}