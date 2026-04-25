package ent_test

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pedrobarco/mroki/ent/enttest"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/pedrobarco/mroki/internal/infrastructure/persistence/ent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatsRepository_GetGlobalStats_empty(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	repo := ent.NewStatsRepository(client)

	stats, err := repo.GetGlobalStats(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, int64(0), stats.TotalGates)
	assert.Equal(t, int64(0), stats.TotalRequests24h)
	assert.Equal(t, float64(0), stats.TotalDiffRate)
}

func setupGateWithURLs(t *testing.T, repo traffictesting.GateRepository, liveHost, shadowHost string) traffictesting.GateID {
	t.Helper()
	liveURL, _ := traffictesting.ParseGateURL("http://" + liveHost)
	shadowURL, _ := traffictesting.ParseGateURL("http://" + shadowHost)
	gate, _ := traffictesting.NewGate(nextGateName(), liveURL, shadowURL)
	require.NoError(t, repo.Save(context.Background(), gate))
	return gate.ID
}

func TestStatsRepository_GetGlobalStats_with_data(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	gateRepo := ent.NewGateRepository(client)
	reqRepo := ent.NewRequestRepository(client)
	statsRepo := ent.NewStatsRepository(client)

	// Create 2 gates with unique URLs
	gateID1 := setupGateWithURLs(t, gateRepo, "live1.stats.example.com", "shadow1.stats.example.com")
	gateID2 := setupGateWithURLs(t, gateRepo, "live2.stats.example.com", "shadow2.stats.example.com")

	// gate1: 2 requests with diffs
	req1 := newTestRequest(t, gateID1)
	req2 := newTestRequest(t, gateID1)
	require.NoError(t, reqRepo.Save(context.Background(), req1))
	require.NoError(t, reqRepo.Save(context.Background(), req2))

	// gate2: 1 request without diff
	req3 := newTestRequestWithoutDiff(t, gateID2)
	require.NoError(t, reqRepo.Save(context.Background(), req3))

	stats, err := statsRepo.GetGlobalStats(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, int64(2), stats.TotalGates)
	assert.Equal(t, int64(3), stats.TotalRequests24h)
	assert.InDelta(t, 66.66, stats.TotalDiffRate, 0.1)
}

func TestStatsRepository_GetGlobalStats_no_diffs(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	gateRepo := ent.NewGateRepository(client)
	reqRepo := ent.NewRequestRepository(client)
	statsRepo := ent.NewStatsRepository(client)

	gateID := setupGateWithURLs(t, gateRepo, "live.nodiff.example.com", "shadow.nodiff.example.com")

	req1 := newTestRequestWithoutDiff(t, gateID)
	req2 := newTestRequestWithoutDiff(t, gateID)
	require.NoError(t, reqRepo.Save(context.Background(), req1))
	require.NoError(t, reqRepo.Save(context.Background(), req2))

	stats, err := statsRepo.GetGlobalStats(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, int64(1), stats.TotalGates)
	assert.Equal(t, int64(2), stats.TotalRequests24h)
	assert.Equal(t, float64(0), stats.TotalDiffRate)
}


func TestStatsRepository_GetStatsByGateIDs_no_requests(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	gateRepo := ent.NewGateRepository(client)
	statsRepo := ent.NewStatsRepository(client)

	gateID := setupGateWithURLs(t, gateRepo, "live.noreq.example.com", "shadow.noreq.example.com")

	result, err := statsRepo.GetStatsByGateIDs(context.Background(), []traffictesting.GateID{gateID})

	assert.NoError(t, err)
	stats := result[gateID]
	assert.Equal(t, int64(0), stats.RequestCount24h)
	assert.Equal(t, int64(0), stats.DiffCount24h)
	assert.Equal(t, float64(0), stats.DiffRate)
	assert.Nil(t, stats.LastActive)
}

func TestStatsRepository_GetStatsByGateIDs_with_requests_and_diffs(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	gateRepo := ent.NewGateRepository(client)
	reqRepo := ent.NewRequestRepository(client)
	statsRepo := ent.NewStatsRepository(client)

	gateID := setupGateWithURLs(t, gateRepo, "live.withreq.example.com", "shadow.withreq.example.com")

	// Create 3 requests: 2 with diffs, 1 without
	req1 := newTestRequest(t, gateID)
	req2 := newTestRequest(t, gateID)
	req3 := newTestRequestWithoutDiff(t, gateID)
	require.NoError(t, reqRepo.Save(context.Background(), req1))
	require.NoError(t, reqRepo.Save(context.Background(), req2))
	require.NoError(t, reqRepo.Save(context.Background(), req3))

	result, err := statsRepo.GetStatsByGateIDs(context.Background(), []traffictesting.GateID{gateID})

	assert.NoError(t, err)
	stats := result[gateID]
	assert.Equal(t, int64(3), stats.RequestCount24h)
	assert.Equal(t, int64(2), stats.DiffCount24h)
	assert.InDelta(t, 66.66, stats.DiffRate, 0.1)
	assert.NotNil(t, stats.LastActive)
}

func TestStatsRepository_GetStatsByGateIDs_multiple_gates(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	gateRepo := ent.NewGateRepository(client)
	reqRepo := ent.NewRequestRepository(client)
	statsRepo := ent.NewStatsRepository(client)

	gateID1 := setupGateWithURLs(t, gateRepo, "live1.multi.example.com", "shadow1.multi.example.com")
	gateID2 := setupGateWithURLs(t, gateRepo, "live2.multi.example.com", "shadow2.multi.example.com")

	// gate1: 2 requests, 1 diff
	req1 := newTestRequest(t, gateID1)
	req2 := newTestRequestWithoutDiff(t, gateID1)
	require.NoError(t, reqRepo.Save(context.Background(), req1))
	require.NoError(t, reqRepo.Save(context.Background(), req2))

	// gate2: 0 requests

	result, err := statsRepo.GetStatsByGateIDs(context.Background(), []traffictesting.GateID{gateID1, gateID2})

	assert.NoError(t, err)

	// gate1 stats
	g1Stats := result[gateID1]
	assert.Equal(t, int64(2), g1Stats.RequestCount24h)
	assert.Equal(t, int64(1), g1Stats.DiffCount24h)
	assert.InDelta(t, 50.0, g1Stats.DiffRate, 0.1)
	assert.NotNil(t, g1Stats.LastActive)

	// gate2 stats (no requests)
	g2Stats := result[gateID2]
	assert.Equal(t, int64(0), g2Stats.RequestCount24h)
	assert.Equal(t, int64(0), g2Stats.DiffCount24h)
	assert.Equal(t, float64(0), g2Stats.DiffRate)
	assert.Nil(t, g2Stats.LastActive)
}

func TestStatsRepository_GetStatsByGateIDs_empty_ids(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	statsRepo := ent.NewStatsRepository(client)

	result, err := statsRepo.GetStatsByGateIDs(context.Background(), []traffictesting.GateID{})

	assert.NoError(t, err)
	assert.Empty(t, result)
}