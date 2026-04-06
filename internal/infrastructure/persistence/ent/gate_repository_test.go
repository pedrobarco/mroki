package ent_test

import (
	"context"
	"fmt"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pedrobarco/mroki/ent/enttest"
	"github.com/pedrobarco/mroki/internal/domain/pagination"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/pedrobarco/mroki/internal/infrastructure/persistence/ent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var gateCounter int

func nextGateName() traffictesting.GateName {
	gateCounter++
	n, _ := traffictesting.ParseGateName(fmt.Sprintf("gate-%d", gateCounter))
	return n
}

func TestGateRepository_Save_success(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	repo := ent.NewGateRepository(client)

	liveURL, _ := traffictesting.ParseGateURL("http://live.example.com")
	shadowURL, _ := traffictesting.ParseGateURL("http://shadow.example.com")
	gate, _ := traffictesting.NewGate(nextGateName(), liveURL, shadowURL)

	err := repo.Save(context.Background(), gate)

	assert.NoError(t, err)
}

func TestGateRepository_Save_database_error(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	repo := ent.NewGateRepository(client)

	liveURL, _ := traffictesting.ParseGateURL("http://live.example.com")
	shadowURL, _ := traffictesting.ParseGateURL("http://shadow.example.com")
	gate, _ := traffictesting.NewGate(nextGateName(), liveURL, shadowURL)

	// Save twice should fail (duplicate ID)
	err := repo.Save(context.Background(), gate)
	require.NoError(t, err)

	err = repo.Save(context.Background(), gate)
	assert.Error(t, err)
}

func TestGateRepository_GetByID_success(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	repo := ent.NewGateRepository(client)

	liveURL, _ := traffictesting.ParseGateURL("http://live.example.com")
	shadowURL, _ := traffictesting.ParseGateURL("http://shadow.example.com")
	gate, _ := traffictesting.NewGate(nextGateName(), liveURL, shadowURL)

	require.NoError(t, repo.Save(context.Background(), gate))

	result, err := repo.GetByID(context.Background(), gate.ID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, gate.ID.String(), result.ID.String())
	assert.Equal(t, "http://live.example.com", result.LiveURL.String())
	assert.Equal(t, "http://shadow.example.com", result.ShadowURL.String())
}

func TestGateRepository_GetByID_not_found(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	repo := ent.NewGateRepository(client)

	gateID := traffictesting.NewGateID()

	gate, err := repo.GetByID(context.Background(), gateID)

	assert.Error(t, err)
	assert.Nil(t, gate)
	assert.ErrorIs(t, err, traffictesting.ErrGateNotFound)
}

func TestGateRepository_GetAll_success(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	repo := ent.NewGateRepository(client)

	liveURL1, _ := traffictesting.ParseGateURL("http://live1.example.com")
	shadowURL1, _ := traffictesting.ParseGateURL("http://shadow1.example.com")
	gate1, _ := traffictesting.NewGate(nextGateName(), liveURL1, shadowURL1)

	liveURL2, _ := traffictesting.ParseGateURL("http://live2.example.com")
	shadowURL2, _ := traffictesting.ParseGateURL("http://shadow2.example.com")
	gate2, _ := traffictesting.NewGate(nextGateName(), liveURL2, shadowURL2)

	require.NoError(t, repo.Save(context.Background(), gate1))
	require.NoError(t, repo.Save(context.Background(), gate2))

	params, _ := pagination.NewParams(50, 0)
	filters := traffictesting.EmptyGateFilters()
	sort := traffictesting.DefaultGateSort()
	result, err := repo.GetAll(context.Background(), filters, sort, params)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, int64(2), result.Total)
	assert.Equal(t, 50, result.Limit)
	assert.Equal(t, 0, result.Offset)
	assert.False(t, result.HasMore)
}

func TestGateRepository_GetAll_empty(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	repo := ent.NewGateRepository(client)

	params, _ := pagination.NewParams(50, 0)
	filters := traffictesting.EmptyGateFilters()
	sort := traffictesting.DefaultGateSort()
	result, err := repo.GetAll(context.Background(), filters, sort, params)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Items)
	assert.Equal(t, int64(0), result.Total)
	assert.False(t, result.HasMore)
}

func TestGateRepository_GetAll_pagination(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	repo := ent.NewGateRepository(client)

	// Create 3 gates with unique URL pairs
	for i := 0; i < 3; i++ {
		liveURL, _ := traffictesting.ParseGateURL(fmt.Sprintf("http://live-%d.example.com", i))
		shadowURL, _ := traffictesting.ParseGateURL(fmt.Sprintf("http://shadow-%d.example.com", i))
		gate, _ := traffictesting.NewGate(nextGateName(), liveURL, shadowURL)
		require.NoError(t, repo.Save(context.Background(), gate))
	}

	// Get first page (limit 2)
	params, _ := pagination.NewParams(2, 0)
	filters := traffictesting.EmptyGateFilters()
	sort := traffictesting.DefaultGateSort()
	result, err := repo.GetAll(context.Background(), filters, sort, params)

	assert.NoError(t, err)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, int64(3), result.Total)
	assert.True(t, result.HasMore)
}

func TestGateRepository_GetAll_filter_by_live_url(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	repo := ent.NewGateRepository(client)

	liveURL1, _ := traffictesting.ParseGateURL("http://api.production.example.com")
	shadowURL1, _ := traffictesting.ParseGateURL("http://shadow1.example.com")
	gate1, _ := traffictesting.NewGate(nextGateName(), liveURL1, shadowURL1)

	liveURL2, _ := traffictesting.ParseGateURL("http://api.staging.example.com")
	shadowURL2, _ := traffictesting.ParseGateURL("http://shadow2.example.com")
	gate2, _ := traffictesting.NewGate(nextGateName(), liveURL2, shadowURL2)

	require.NoError(t, repo.Save(context.Background(), gate1))
	require.NoError(t, repo.Save(context.Background(), gate2))

	params, _ := pagination.NewParams(50, 0)
	sort := traffictesting.DefaultGateSort()

	// Filter for "production" in live_url
	filters := traffictesting.NewGateFilters("", "production", "")
	result, err := repo.GetAll(context.Background(), filters, sort, params)

	assert.NoError(t, err)
	assert.Len(t, result.Items, 1)
	assert.Equal(t, int64(1), result.Total)
	assert.Equal(t, "http://api.production.example.com", result.Items[0].LiveURL.String())
}

func TestGateRepository_GetAll_filter_by_shadow_url(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	repo := ent.NewGateRepository(client)

	liveURL1, _ := traffictesting.ParseGateURL("http://live1.example.com")
	shadowURL1, _ := traffictesting.ParseGateURL("http://shadow-alpha.example.com")
	gate1, _ := traffictesting.NewGate(nextGateName(), liveURL1, shadowURL1)

	liveURL2, _ := traffictesting.ParseGateURL("http://live2.example.com")
	shadowURL2, _ := traffictesting.ParseGateURL("http://shadow-beta.example.com")
	gate2, _ := traffictesting.NewGate(nextGateName(), liveURL2, shadowURL2)

	require.NoError(t, repo.Save(context.Background(), gate1))
	require.NoError(t, repo.Save(context.Background(), gate2))

	params, _ := pagination.NewParams(50, 0)
	sort := traffictesting.DefaultGateSort()

	// Filter for "beta" in shadow_url
	filters := traffictesting.NewGateFilters("", "", "beta")
	result, err := repo.GetAll(context.Background(), filters, sort, params)

	assert.NoError(t, err)
	assert.Len(t, result.Items, 1)
	assert.Equal(t, int64(1), result.Total)
	assert.Equal(t, "http://shadow-beta.example.com", result.Items[0].ShadowURL.String())
}

func TestGateRepository_GetAll_filter_by_both_urls(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	repo := ent.NewGateRepository(client)

	liveURL1, _ := traffictesting.ParseGateURL("http://api.production.example.com")
	shadowURL1, _ := traffictesting.ParseGateURL("http://shadow-alpha.example.com")
	gate1, _ := traffictesting.NewGate(nextGateName(), liveURL1, shadowURL1)

	liveURL2, _ := traffictesting.ParseGateURL("http://api.production.example.com")
	shadowURL2, _ := traffictesting.ParseGateURL("http://shadow-beta.example.com")
	gate2, _ := traffictesting.NewGate(nextGateName(), liveURL2, shadowURL2)

	liveURL3, _ := traffictesting.ParseGateURL("http://api.staging.example.com")
	shadowURL3, _ := traffictesting.ParseGateURL("http://shadow-alpha.example.com")
	gate3, _ := traffictesting.NewGate(nextGateName(), liveURL3, shadowURL3)

	require.NoError(t, repo.Save(context.Background(), gate1))
	require.NoError(t, repo.Save(context.Background(), gate2))
	require.NoError(t, repo.Save(context.Background(), gate3))

	params, _ := pagination.NewParams(50, 0)
	sort := traffictesting.DefaultGateSort()

	// Filter for "production" in live_url AND "alpha" in shadow_url
	filters := traffictesting.NewGateFilters("", "production", "alpha")
	result, err := repo.GetAll(context.Background(), filters, sort, params)

	assert.NoError(t, err)
	assert.Len(t, result.Items, 1)
	assert.Equal(t, int64(1), result.Total)
	assert.Equal(t, "http://api.production.example.com", result.Items[0].LiveURL.String())
	assert.Equal(t, "http://shadow-alpha.example.com", result.Items[0].ShadowURL.String())
}

func TestGateRepository_GetAll_filter_no_match(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	repo := ent.NewGateRepository(client)

	liveURL, _ := traffictesting.ParseGateURL("http://live.example.com")
	shadowURL, _ := traffictesting.ParseGateURL("http://shadow.example.com")
	gate1, _ := traffictesting.NewGate(nextGateName(), liveURL, shadowURL)
	require.NoError(t, repo.Save(context.Background(), gate1))

	params, _ := pagination.NewParams(50, 0)
	sort := traffictesting.DefaultGateSort()

	filters := traffictesting.NewGateFilters("", "nonexistent", "")
	result, err := repo.GetAll(context.Background(), filters, sort, params)

	assert.NoError(t, err)
	assert.Empty(t, result.Items)
	assert.Equal(t, int64(0), result.Total)
}

func TestGateRepository_GetAll_filter_case_insensitive(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	repo := ent.NewGateRepository(client)

	liveURL, _ := traffictesting.ParseGateURL("http://api.Production.example.com")
	shadowURL, _ := traffictesting.ParseGateURL("http://shadow.example.com")
	gate1, _ := traffictesting.NewGate(nextGateName(), liveURL, shadowURL)
	require.NoError(t, repo.Save(context.Background(), gate1))

	params, _ := pagination.NewParams(50, 0)
	sort := traffictesting.DefaultGateSort()

	// Search with lowercase should match uppercase in URL
	filters := traffictesting.NewGateFilters("", "production", "")
	result, err := repo.GetAll(context.Background(), filters, sort, params)

	assert.NoError(t, err)
	assert.Len(t, result.Items, 1)
}

func TestGateRepository_GetAll_sort_by_live_url_asc(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	repo := ent.NewGateRepository(client)

	liveURL1, _ := traffictesting.ParseGateURL("http://charlie.example.com")
	shadowURL1, _ := traffictesting.ParseGateURL("http://shadow.example.com")
	gate1, _ := traffictesting.NewGate(nextGateName(), liveURL1, shadowURL1)

	liveURL2, _ := traffictesting.ParseGateURL("http://alpha.example.com")
	shadowURL2, _ := traffictesting.ParseGateURL("http://shadow.example.com")
	gate2, _ := traffictesting.NewGate(nextGateName(), liveURL2, shadowURL2)

	liveURL3, _ := traffictesting.ParseGateURL("http://bravo.example.com")
	shadowURL3, _ := traffictesting.ParseGateURL("http://shadow.example.com")
	gate3, _ := traffictesting.NewGate(nextGateName(), liveURL3, shadowURL3)

	require.NoError(t, repo.Save(context.Background(), gate1))
	require.NoError(t, repo.Save(context.Background(), gate2))
	require.NoError(t, repo.Save(context.Background(), gate3))

	params, _ := pagination.NewParams(50, 0)
	filters := traffictesting.EmptyGateFilters()
	sort := traffictesting.NewGateSort(traffictesting.SortByLiveURL(), traffictesting.Asc())

	result, err := repo.GetAll(context.Background(), filters, sort, params)

	assert.NoError(t, err)
	assert.Len(t, result.Items, 3)
	assert.Equal(t, "http://alpha.example.com", result.Items[0].LiveURL.String())
	assert.Equal(t, "http://bravo.example.com", result.Items[1].LiveURL.String())
	assert.Equal(t, "http://charlie.example.com", result.Items[2].LiveURL.String())
}

func TestGateRepository_GetAll_sort_by_live_url_desc(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	repo := ent.NewGateRepository(client)

	liveURL1, _ := traffictesting.ParseGateURL("http://charlie.example.com")
	shadowURL1, _ := traffictesting.ParseGateURL("http://shadow.example.com")
	gate1, _ := traffictesting.NewGate(nextGateName(), liveURL1, shadowURL1)

	liveURL2, _ := traffictesting.ParseGateURL("http://alpha.example.com")
	shadowURL2, _ := traffictesting.ParseGateURL("http://shadow.example.com")
	gate2, _ := traffictesting.NewGate(nextGateName(), liveURL2, shadowURL2)

	require.NoError(t, repo.Save(context.Background(), gate1))
	require.NoError(t, repo.Save(context.Background(), gate2))

	params, _ := pagination.NewParams(50, 0)
	filters := traffictesting.EmptyGateFilters()
	sort := traffictesting.NewGateSort(traffictesting.SortByLiveURL(), traffictesting.Desc())

	result, err := repo.GetAll(context.Background(), filters, sort, params)

	assert.NoError(t, err)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, "http://charlie.example.com", result.Items[0].LiveURL.String())
	assert.Equal(t, "http://alpha.example.com", result.Items[1].LiveURL.String())
}

func TestGateRepository_GetAll_sort_by_shadow_url(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	repo := ent.NewGateRepository(client)

	liveURL1, _ := traffictesting.ParseGateURL("http://live.example.com")
	shadowURL1, _ := traffictesting.ParseGateURL("http://zebra.example.com")
	gate1, _ := traffictesting.NewGate(nextGateName(), liveURL1, shadowURL1)

	liveURL2, _ := traffictesting.ParseGateURL("http://live.example.com")
	shadowURL2, _ := traffictesting.ParseGateURL("http://apple.example.com")
	gate2, _ := traffictesting.NewGate(nextGateName(), liveURL2, shadowURL2)

	require.NoError(t, repo.Save(context.Background(), gate1))
	require.NoError(t, repo.Save(context.Background(), gate2))

	params, _ := pagination.NewParams(50, 0)
	filters := traffictesting.EmptyGateFilters()
	sort := traffictesting.NewGateSort(traffictesting.SortByShadowURL(), traffictesting.Asc())

	result, err := repo.GetAll(context.Background(), filters, sort, params)

	assert.NoError(t, err)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, "http://apple.example.com", result.Items[0].ShadowURL.String())
	assert.Equal(t, "http://zebra.example.com", result.Items[1].ShadowURL.String())
}

func TestGateRepository_GetByID_stats_no_requests(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	repo := ent.NewGateRepository(client)

	liveURL, _ := traffictesting.ParseGateURL("http://live.example.com")
	shadowURL, _ := traffictesting.ParseGateURL("http://shadow.example.com")
	gate, _ := traffictesting.NewGate(nextGateName(), liveURL, shadowURL)
	require.NoError(t, repo.Save(context.Background(), gate))

	result, err := repo.GetByID(context.Background(), gate.ID)

	assert.NoError(t, err)
	assert.Equal(t, int64(0), result.Stats.RequestCount24h)
	assert.Equal(t, int64(0), result.Stats.DiffCount24h)
	assert.Equal(t, float64(0), result.Stats.DiffRate)
	assert.Nil(t, result.Stats.LastActive)
}

func TestGateRepository_GetByID_stats_with_requests_and_diffs(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	gateRepo := ent.NewGateRepository(client)
	reqRepo := ent.NewRequestRepository(client)

	liveURL, _ := traffictesting.ParseGateURL("http://live.example.com")
	shadowURL, _ := traffictesting.ParseGateURL("http://shadow.example.com")
	gate, _ := traffictesting.NewGate(nextGateName(), liveURL, shadowURL)
	require.NoError(t, gateRepo.Save(context.Background(), gate))

	// Create 3 requests: 2 with diffs, 1 without
	req1 := newTestRequest(t, gate.ID) // has diff
	req2 := newTestRequest(t, gate.ID) // has diff
	req3 := newTestRequestWithoutDiff(t, gate.ID)
	require.NoError(t, reqRepo.Save(context.Background(), req1))
	require.NoError(t, reqRepo.Save(context.Background(), req2))
	require.NoError(t, reqRepo.Save(context.Background(), req3))

	result, err := gateRepo.GetByID(context.Background(), gate.ID)

	assert.NoError(t, err)
	assert.Equal(t, int64(3), result.Stats.RequestCount24h)
	assert.Equal(t, int64(2), result.Stats.DiffCount24h)
	assert.InDelta(t, 66.66, result.Stats.DiffRate, 0.1)
	assert.NotNil(t, result.Stats.LastActive)
}

func TestGateRepository_GetAll_stats_populated(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	gateRepo := ent.NewGateRepository(client)
	reqRepo := ent.NewRequestRepository(client)

	liveURL1, _ := traffictesting.ParseGateURL("http://live1.example.com")
	shadowURL1, _ := traffictesting.ParseGateURL("http://shadow1.example.com")
	gate1, _ := traffictesting.NewGate(nextGateName(), liveURL1, shadowURL1)

	liveURL2, _ := traffictesting.ParseGateURL("http://live2.example.com")
	shadowURL2, _ := traffictesting.ParseGateURL("http://shadow2.example.com")
	gate2, _ := traffictesting.NewGate(nextGateName(), liveURL2, shadowURL2)

	require.NoError(t, gateRepo.Save(context.Background(), gate1))
	require.NoError(t, gateRepo.Save(context.Background(), gate2))

	// gate1: 2 requests, 1 diff
	req1 := newTestRequest(t, gate1.ID)
	req2 := newTestRequestWithoutDiff(t, gate1.ID)
	require.NoError(t, reqRepo.Save(context.Background(), req1))
	require.NoError(t, reqRepo.Save(context.Background(), req2))

	// gate2: 0 requests

	params, _ := pagination.NewParams(50, 0)
	filters := traffictesting.EmptyGateFilters()
	sort := traffictesting.DefaultGateSort()
	result, err := gateRepo.GetAll(context.Background(), filters, sort, params)

	assert.NoError(t, err)
	assert.Len(t, result.Items, 2)

	// Find gate1 and gate2 in results
	var g1Stats, g2Stats traffictesting.GateStats
	for _, g := range result.Items {
		switch g.ID {
		case gate1.ID:
			g1Stats = g.Stats
		case gate2.ID:
			g2Stats = g.Stats
		}
	}

	// gate1 stats
	assert.Equal(t, int64(2), g1Stats.RequestCount24h)
	assert.Equal(t, int64(1), g1Stats.DiffCount24h)
	assert.InDelta(t, 50.0, g1Stats.DiffRate, 0.1)
	assert.NotNil(t, g1Stats.LastActive)

	// gate2 stats (no requests)
	assert.Equal(t, int64(0), g2Stats.RequestCount24h)
	assert.Equal(t, int64(0), g2Stats.DiffCount24h)
	assert.Equal(t, float64(0), g2Stats.DiffRate)
	assert.Nil(t, g2Stats.LastActive)
}
