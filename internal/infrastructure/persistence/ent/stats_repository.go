package ent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pedrobarco/mroki/ent"
	"github.com/pedrobarco/mroki/ent/request"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
)

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

type statsRepository struct {
	client *ent.Client
}

var _ traffictesting.StatsRepository = (*statsRepository)(nil)

func NewStatsRepository(client *ent.Client) *statsRepository {
	return &statsRepository{client: client}
}

func (r *statsRepository) GetGlobalStats(ctx context.Context) (*traffictesting.GlobalStats, error) {
	since24h := time.Now().Add(-24 * time.Hour)

	// Query 1: total_gates
	totalGates, err := r.client.Gate.Query().Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count gates: %w", err)
	}

	// Query 2: total_requests_24h
	totalReqs, err := r.client.Request.Query().
		Where(request.CreatedAtGTE(since24h)).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count requests: %w", err)
	}

	// Query 3: total diffs 24h
	totalDiffs, err := r.client.Request.Query().
		Where(request.CreatedAtGTE(since24h), request.HasDiff()).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count diffs: %w", err)
	}

	var diffRate float64
	if totalReqs > 0 {
		diffRate = float64(totalDiffs) / float64(totalReqs) * 100
	}

	return &traffictesting.GlobalStats{
		TotalGates:       int64(totalGates),
		TotalRequests24h: int64(totalReqs),
		TotalDiffRate:    diffRate,
	}, nil
}

func (r *statsRepository) GetStatsByGateIDs(ctx context.Context, ids []traffictesting.GateID) (map[traffictesting.GateID]traffictesting.GateStats, error) {
	if len(ids) == 0 {
		return map[traffictesting.GateID]traffictesting.GateStats{}, nil
	}

	uuids := make([]uuid.UUID, len(ids))
	for i, id := range ids {
		uuids[i] = id.UUID()
	}
	since24h := time.Now().Add(-24 * time.Hour)

	// Query 1: request_count_24h per gate
	var requestCounts []gateStatRow
	err := r.client.Request.Query().
		Where(request.GateIDIn(uuids...), request.CreatedAtGTE(since24h)).
		GroupBy(request.FieldGateID).
		Aggregate(ent.Count()).
		Scan(ctx, &requestCounts)
	if err != nil {
		return nil, fmt.Errorf("failed to query request counts: %w", err)
	}

	// Query 2: diff_count_24h per gate (requests that have a diff)
	var diffCounts []gateStatRow
	err = r.client.Request.Query().
		Where(request.GateIDIn(uuids...), request.CreatedAtGTE(since24h), request.HasDiff()).
		GroupBy(request.FieldGateID).
		Aggregate(ent.Count()).
		Scan(ctx, &diffCounts)
	if err != nil {
		return nil, fmt.Errorf("failed to query diff counts: %w", err)
	}

	// Query 3: last_active per gate (no time filter)
	var lastActives []gateLastActiveRow
	err = r.client.Request.Query().
		Where(request.GateIDIn(uuids...)).
		GroupBy(request.FieldGateID).
		Aggregate(ent.As(ent.Max(request.FieldCreatedAt), "last_active")).
		Scan(ctx, &lastActives)
	if err != nil {
		return nil, fmt.Errorf("failed to query last active: %w", err)
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

	// Compose results
	result := make(map[traffictesting.GateID]traffictesting.GateStats, len(ids))
	for _, id := range ids {
		uid := id.UUID()
		reqCount := reqCountMap[uid]
		diffCount := diffCountMap[uid]
		var diffRate float64
		if reqCount > 0 {
			diffRate = float64(diffCount) / float64(reqCount) * 100
		}
		var lastActive *time.Time
		if t, ok := lastActiveMap[uid]; ok {
			lastActive = &t
		}
		result[id] = traffictesting.GateStats{
			RequestCount24h: reqCount,
			DiffCount24h:    diffCount,
			DiffRate:        diffRate,
			LastActive:      lastActive,
		}
	}
	return result, nil
}
