package ent

import (
	"context"
	"fmt"
	"time"

	"github.com/pedrobarco/mroki/ent"
	"github.com/pedrobarco/mroki/ent/request"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
)

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
