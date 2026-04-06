package traffictesting

import "context"

// StatsRepository defines the contract for statistics queries.
type StatsRepository interface {
	GetGlobalStats(ctx context.Context) (*GlobalStats, error)
}
