package postgres_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/pedrobarco/mroki/internal/domain/pagination"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/pedrobarco/mroki/internal/infrastructure/persistence/postgres"
	"github.com/pedrobarco/mroki/internal/infrastructure/persistence/postgres/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGateRepository_Save_success(t *testing.T) {
	mock, err := pgxmock.NewConn()
	require.NoError(t, err)
	defer func() { _ = mock.Close(context.Background()) }()

	queries := db.New(mock)
	repo := postgres.NewGateRepository(queries)

	liveURL, _ := traffictesting.ParseGateURL("http://live.example.com")
	shadowURL, _ := traffictesting.ParseGateURL("http://shadow.example.com")
	gate, _ := traffictesting.NewGate(liveURL, shadowURL)

	// Expect SaveGate query
	mock.ExpectExec("INSERT INTO gates").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = repo.Save(context.Background(), gate)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGateRepository_Save_database_error(t *testing.T) {
	mock, err := pgxmock.NewConn()
	require.NoError(t, err)
	defer func() { _ = mock.Close(context.Background()) }()

	queries := db.New(mock)
	repo := postgres.NewGateRepository(queries)

	liveURL, _ := traffictesting.ParseGateURL("http://live.example.com")
	shadowURL, _ := traffictesting.ParseGateURL("http://shadow.example.com")
	gate, _ := traffictesting.NewGate(liveURL, shadowURL)

	// Expect SaveGate query to fail
	mock.ExpectExec("INSERT INTO gates").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(pgx.ErrTxClosed)

	err = repo.Save(context.Background(), gate)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save gate")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGateRepository_GetByID_success(t *testing.T) {
	mock, err := pgxmock.NewConn()
	require.NoError(t, err)
	defer func() { _ = mock.Close(context.Background()) }()

	queries := db.New(mock)
	repo := postgres.NewGateRepository(queries)

	gateID := traffictesting.NewGateID()
	liveURL := "http://live.example.com"
	shadowURL := "http://shadow.example.com"

	// Expect GetGateByID query
	rows := pgxmock.NewRows([]string{"id", "live_url", "shadow_url"}).
		AddRow(
			pgtype.UUID{Bytes: gateID.UUID(), Valid: true},
			pgtype.Text{String: liveURL, Valid: true},
			pgtype.Text{String: shadowURL, Valid: true},
		)

	mock.ExpectQuery("SELECT (.+) FROM gates WHERE id").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(rows)

	gate, err := repo.GetByID(context.Background(), gateID)

	assert.NoError(t, err)
	assert.NotNil(t, gate)
	assert.Equal(t, gateID.String(), gate.ID.String())
	assert.Equal(t, liveURL, gate.LiveURL.String())
	assert.Equal(t, shadowURL, gate.ShadowURL.String())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGateRepository_GetByID_not_found(t *testing.T) {
	mock, err := pgxmock.NewConn()
	require.NoError(t, err)
	defer func() { _ = mock.Close(context.Background()) }()

	queries := db.New(mock)
	repo := postgres.NewGateRepository(queries)

	gateID := traffictesting.NewGateID()

	// Expect GetGateByID query to return no rows
	mock.ExpectQuery("SELECT (.+) FROM gates WHERE id").
		WithArgs(pgxmock.AnyArg()).
		WillReturnError(pgx.ErrNoRows)

	gate, err := repo.GetByID(context.Background(), gateID)

	assert.Error(t, err)
	assert.Nil(t, gate)
	assert.ErrorIs(t, err, traffictesting.ErrGateNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGateRepository_GetByID_invalid_live_url(t *testing.T) {
	mock, err := pgxmock.NewConn()
	require.NoError(t, err)
	defer func() { _ = mock.Close(context.Background()) }()

	queries := db.New(mock)
	repo := postgres.NewGateRepository(queries)

	gateID := traffictesting.NewGateID()

	// Return invalid URL
	rows := pgxmock.NewRows([]string{"id", "live_url", "shadow_url"}).
		AddRow(
			pgtype.UUID{Bytes: gateID.UUID(), Valid: true},
			pgtype.Text{String: "ftp://invalid.com", Valid: true}, // Invalid scheme
			pgtype.Text{String: "http://shadow.example.com", Valid: true},
		)

	mock.ExpectQuery("SELECT (.+) FROM gates WHERE id").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(rows)

	gate, err := repo.GetByID(context.Background(), gateID)

	assert.Error(t, err)
	assert.Nil(t, gate)
	assert.Contains(t, err.Error(), "invalid live URL")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGateRepository_GetAll_success(t *testing.T) {
	mock, err := pgxmock.NewConn()
	require.NoError(t, err)
	defer func() { _ = mock.Close(context.Background()) }()

	queries := db.New(mock)
	repo := postgres.NewGateRepository(queries)

	gateID1 := traffictesting.NewGateID()
	gateID2 := traffictesting.NewGateID()

	params, _ := pagination.NewParams(50, 0)

	// Expect CountGates query
	countRows := pgxmock.NewRows([]string{"count"}).AddRow(int64(2))
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(countRows)

	// Expect GetAllGates query
	rows := pgxmock.NewRows([]string{"id", "live_url", "shadow_url"}).
		AddRow(
			pgtype.UUID{Bytes: gateID1.UUID(), Valid: true},
			pgtype.Text{String: "http://live1.example.com", Valid: true},
			pgtype.Text{String: "http://shadow1.example.com", Valid: true},
		).
		AddRow(
			pgtype.UUID{Bytes: gateID2.UUID(), Valid: true},
			pgtype.Text{String: "http://live2.example.com", Valid: true},
			pgtype.Text{String: "http://shadow2.example.com", Valid: true},
		)

	mock.ExpectQuery("SELECT (.+) FROM gates").
		WithArgs(int32(0), int32(50)).
		WillReturnRows(rows)

	result, err := repo.GetAll(context.Background(), params)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, int64(2), result.Total)
	assert.Equal(t, 50, result.Limit)
	assert.Equal(t, 0, result.Offset)
	assert.False(t, result.HasMore)
	assert.Equal(t, gateID1.String(), result.Items[0].ID.String())
	assert.Equal(t, gateID2.String(), result.Items[1].ID.String())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGateRepository_GetAll_empty_result(t *testing.T) {
	mock, err := pgxmock.NewConn()
	require.NoError(t, err)
	defer func() { _ = mock.Close(context.Background()) }()

	queries := db.New(mock)
	repo := postgres.NewGateRepository(queries)

	params, _ := pagination.NewParams(50, 0)

	// Expect CountGates query
	countRows := pgxmock.NewRows([]string{"count"}).AddRow(int64(0))
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(countRows)

	// Expect GetAllGates query with no rows
	rows := pgxmock.NewRows([]string{"id", "live_url", "shadow_url"})

	mock.ExpectQuery("SELECT (.+) FROM gates").
		WithArgs(int32(0), int32(50)).
		WillReturnRows(rows)

	result, err := repo.GetAll(context.Background(), params)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Items)
	assert.Equal(t, int64(0), result.Total)
	assert.False(t, result.HasMore)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGateRepository_GetAll_database_error(t *testing.T) {
	mock, err := pgxmock.NewConn()
	require.NoError(t, err)
	defer func() { _ = mock.Close(context.Background()) }()

	queries := db.New(mock)
	repo := postgres.NewGateRepository(queries)

	params, _ := pagination.NewParams(50, 0)

	// Expect CountGates query to fail
	mock.ExpectQuery("SELECT COUNT").
		WillReturnError(pgx.ErrTxClosed)

	result, err := repo.GetAll(context.Background(), params)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}
