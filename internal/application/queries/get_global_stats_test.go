package queries

import (
	"context"
	"errors"
	"testing"

	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockStatsRepositoryForGetGlobalStats struct {
	getGlobalStatsFn func(context.Context) (*traffictesting.GlobalStats, error)
}

func (m *mockStatsRepositoryForGetGlobalStats) GetGlobalStats(ctx context.Context) (*traffictesting.GlobalStats, error) {
	if m.getGlobalStatsFn != nil {
		return m.getGlobalStatsFn(ctx)
	}
	return nil, errors.New("not implemented")
}

func (m *mockStatsRepositoryForGetGlobalStats) GetStatsByGateIDs(ctx context.Context, ids []traffictesting.GateID) (map[traffictesting.GateID]traffictesting.GateStats, error) {
	return nil, errors.New("not implemented")
}

func TestGetGlobalStatsHandler_Handle_success(t *testing.T) {
	// Arrange
	expectedStats := &traffictesting.GlobalStats{
		TotalGates:       12,
		TotalRequests24h: 3480,
		TotalDiffRate:    4.2,
	}
	repo := &mockStatsRepositoryForGetGlobalStats{
		getGlobalStatsFn: func(ctx context.Context) (*traffictesting.GlobalStats, error) {
			return expectedStats, nil
		},
	}
	handler := NewGetGlobalStatsHandler(repo)

	// Act
	stats, err := handler.Handle(context.Background())

	// Assert
	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, expectedStats.TotalGates, stats.TotalGates)
	assert.Equal(t, expectedStats.TotalRequests24h, stats.TotalRequests24h)
	assert.InDelta(t, expectedStats.TotalDiffRate, stats.TotalDiffRate, 0.001)
}

func TestGetGlobalStatsHandler_Handle_repository_error(t *testing.T) {
	// Arrange
	expectedErr := errors.New("database connection failed")
	repo := &mockStatsRepositoryForGetGlobalStats{
		getGlobalStatsFn: func(ctx context.Context) (*traffictesting.GlobalStats, error) {
			return nil, expectedErr
		},
	}
	handler := NewGetGlobalStatsHandler(repo)

	// Act
	stats, err := handler.Handle(context.Background())

	// Assert
	require.Error(t, err)
	assert.Nil(t, stats)
	assert.ErrorIs(t, err, expectedErr)
}
