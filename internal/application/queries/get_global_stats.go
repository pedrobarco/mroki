package queries

import (
	"context"

	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
)

// GetGlobalStatsHandler handles the global stats query
type GetGlobalStatsHandler struct {
	repo traffictesting.StatsRepository
}

// NewGetGlobalStatsHandler creates a new GetGlobalStatsHandler
func NewGetGlobalStatsHandler(repo traffictesting.StatsRepository) *GetGlobalStatsHandler {
	return &GetGlobalStatsHandler{repo: repo}
}

// Handle executes the global stats query
func (h *GetGlobalStatsHandler) Handle(ctx context.Context) (*traffictesting.GlobalStats, error) {
	return h.repo.GetGlobalStats(ctx)
}
