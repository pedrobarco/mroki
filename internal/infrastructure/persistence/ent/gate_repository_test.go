package ent_test

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pedrobarco/mroki/ent/enttest"
	"github.com/pedrobarco/mroki/internal/domain/pagination"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/pedrobarco/mroki/internal/infrastructure/persistence/ent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGateRepository_Save_success(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	repo := ent.NewGateRepository(client)

	liveURL, _ := traffictesting.ParseGateURL("http://live.example.com")
	shadowURL, _ := traffictesting.ParseGateURL("http://shadow.example.com")
	gate, _ := traffictesting.NewGate(liveURL, shadowURL)

	err := repo.Save(context.Background(), gate)

	assert.NoError(t, err)
}

func TestGateRepository_Save_database_error(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	repo := ent.NewGateRepository(client)

	liveURL, _ := traffictesting.ParseGateURL("http://live.example.com")
	shadowURL, _ := traffictesting.ParseGateURL("http://shadow.example.com")
	gate, _ := traffictesting.NewGate(liveURL, shadowURL)

	// Save twice should fail (duplicate ID)
	err := repo.Save(context.Background(), gate)
	require.NoError(t, err)

	err = repo.Save(context.Background(), gate)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save gate")
}

func TestGateRepository_GetByID_success(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer func() { _ = client.Close() }()

	repo := ent.NewGateRepository(client)

	liveURL, _ := traffictesting.ParseGateURL("http://live.example.com")
	shadowURL, _ := traffictesting.ParseGateURL("http://shadow.example.com")
	gate, _ := traffictesting.NewGate(liveURL, shadowURL)

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
	gate1, _ := traffictesting.NewGate(liveURL1, shadowURL1)

	liveURL2, _ := traffictesting.ParseGateURL("http://live2.example.com")
	shadowURL2, _ := traffictesting.ParseGateURL("http://shadow2.example.com")
	gate2, _ := traffictesting.NewGate(liveURL2, shadowURL2)

	require.NoError(t, repo.Save(context.Background(), gate1))
	require.NoError(t, repo.Save(context.Background(), gate2))

	params, _ := pagination.NewParams(50, 0)
	result, err := repo.GetAll(context.Background(), params)

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
	result, err := repo.GetAll(context.Background(), params)

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

	// Create 3 gates
	for i := 0; i < 3; i++ {
		liveURL, _ := traffictesting.ParseGateURL("http://live.example.com")
		shadowURL, _ := traffictesting.ParseGateURL("http://shadow.example.com")
		gate, _ := traffictesting.NewGate(liveURL, shadowURL)
		require.NoError(t, repo.Save(context.Background(), gate))
	}

	// Get first page (limit 2)
	params, _ := pagination.NewParams(2, 0)
	result, err := repo.GetAll(context.Background(), params)

	assert.NoError(t, err)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, int64(3), result.Total)
	assert.True(t, result.HasMore)
}
