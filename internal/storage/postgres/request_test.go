package postgres_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/pedrobarco/mroki/internal/domain/diffing"
	"github.com/pedrobarco/mroki/internal/storage/postgres"
	"github.com/pedrobarco/mroki/internal/storage/postgres/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestRepository_Save_success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := db.New(mock)
	repo := postgres.NewRequestRepository(queries, mock)

	// Create test request with responses and diff
	gateID := diffing.NewGateID()
	requestID := diffing.NewRequestID()
	liveRespID := uuid.New()
	shadowRespID := uuid.New()
	diffID := uuid.New()

	request := &diffing.Request{
		ID:        requestID,
		GateID:    gateID,
		Method:    "POST",
		Path:      "/api/test",
		Headers:   http.Header{"Content-Type": []string{"application/json"}},
		Body:      []byte(`{"test":"data"}`),
		CreatedAt: time.Now(),
		Responses: []diffing.Response{
			{
				ID:         liveRespID,
				Type:       diffing.ResponseTypeLive,
				StatusCode: 200,
				Headers:    http.Header{},
				Body:       []byte(`{"status":"ok"}`),
				CreatedAt:  time.Now(),
			},
			{
				ID:         shadowRespID,
				Type:       diffing.ResponseTypeShadow,
				StatusCode: 200,
				Headers:    http.Header{},
				Body:       []byte(`{"status":"ok"}`),
				CreatedAt:  time.Now(),
			},
		},
		Diff: diffing.Diff{
			ID:             diffID,
			FromResponseID: liveRespID,
			ToResponseID:   shadowRespID,
			Content:        "no differences",
		},
	}

	// Expect transaction
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO requests").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec("INSERT INTO responses").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec("INSERT INTO responses").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec("INSERT INTO diffs").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()
	mock.ExpectRollback() // Deferred rollback

	err = repo.Save(context.Background(), request)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRequestRepository_Save_transaction_begin_error(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := db.New(mock)
	repo := postgres.NewRequestRepository(queries, mock)

	gateID := diffing.NewGateID()
	request := &diffing.Request{
		ID:        diffing.NewRequestID(),
		GateID:    gateID,
		Method:    "POST",
		Path:      "/test",
		Headers:   http.Header{},
		Body:      []byte{},
		CreatedAt: time.Now(),
		Responses: []diffing.Response{},
		Diff:      diffing.Diff{},
	}

	// Expect transaction begin to fail
	mock.ExpectBegin().WillReturnError(pgx.ErrTxClosed)

	err = repo.Save(context.Background(), request)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to begin transaction")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRequestRepository_Save_request_insert_error(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := db.New(mock)
	repo := postgres.NewRequestRepository(queries, mock)

	gateID := diffing.NewGateID()
	request := &diffing.Request{
		ID:        diffing.NewRequestID(),
		GateID:    gateID,
		Method:    "POST",
		Path:      "/test",
		Headers:   http.Header{},
		Body:      []byte{},
		CreatedAt: time.Now(),
		Responses: []diffing.Response{},
		Diff:      diffing.Diff{},
	}

	// Expect transaction and request insert to fail
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO requests").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(pgx.ErrTxClosed)
	mock.ExpectRollback()

	err = repo.Save(context.Background(), request)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save request")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRequestRepository_GetByID_success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := db.New(mock)
	repo := postgres.NewRequestRepository(queries, mock)

	requestID := diffing.NewRequestID()
	gateID := diffing.NewGateID()
	liveRespID := uuid.New()
	shadowRespID := uuid.New()
	diffID := uuid.New()
	now := time.Now()

	// Mock the complex joined query result
	rows := pgxmock.NewRows([]string{
		"request_id", "request_gate_id", "request_agent_id", "request_method", "request_path",
		"request_headers", "request_body", "request_created_at",
		"response_id", "response_type", "response_status_code", "response_headers", "response_body", "response_created_at",
		"diff_id", "diff_from_response_id", "diff_to_response_id", "diff_content",
	}).
		AddRow(
			pgtype.UUID{Bytes: requestID.UUID(), Valid: true},
			pgtype.UUID{Bytes: gateID.UUID(), Valid: true},
			pgtype.Text{String: "", Valid: false},
			pgtype.Text{String: "GET", Valid: true},
			pgtype.Text{String: "/test", Valid: true},
			[]byte(`{}`),
			[]byte(`{"test":"data"}`),
			pgtype.Timestamptz{Time: now, Valid: true},
			pgtype.UUID{Bytes: liveRespID, Valid: true},
			pgtype.Text{String: "live", Valid: true},
			pgtype.Int4{Int32: 200, Valid: true},
			[]byte(`{}`),
			[]byte(`{"status":"ok"}`),
			pgtype.Timestamptz{Time: now, Valid: true},
			pgtype.UUID{Bytes: diffID, Valid: true},
			pgtype.UUID{Bytes: liveRespID, Valid: true},
			pgtype.UUID{Bytes: shadowRespID, Valid: true},
			[]byte("no diff"),
		).
		AddRow(
			pgtype.UUID{Bytes: requestID.UUID(), Valid: true},
			pgtype.UUID{Bytes: gateID.UUID(), Valid: true},
			pgtype.Text{String: "", Valid: false},
			pgtype.Text{String: "GET", Valid: true},
			pgtype.Text{String: "/test", Valid: true},
			[]byte(`{}`),
			[]byte(`{"test":"data"}`),
			pgtype.Timestamptz{Time: now, Valid: true},
			pgtype.UUID{Bytes: shadowRespID, Valid: true},
			pgtype.Text{String: "shadow", Valid: true},
			pgtype.Int4{Int32: 200, Valid: true},
			[]byte(`{}`),
			[]byte(`{"status":"ok"}`),
			pgtype.Timestamptz{Time: now, Valid: true},
			pgtype.UUID{Bytes: diffID, Valid: true},
			pgtype.UUID{Bytes: liveRespID, Valid: true},
			pgtype.UUID{Bytes: shadowRespID, Valid: true},
			[]byte("no diff"),
		)

	mock.ExpectQuery("SELECT (.+) FROM requests").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(rows)

	request, err := repo.GetByID(context.Background(), requestID, gateID)

	assert.NoError(t, err)
	assert.NotNil(t, request)
	assert.Equal(t, requestID.String(), request.ID.String())
	assert.Equal(t, gateID.String(), request.GateID.String())
	assert.Equal(t, "GET", request.Method)
	assert.Len(t, request.Responses, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRequestRepository_GetByID_not_found(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := db.New(mock)
	repo := postgres.NewRequestRepository(queries, mock)

	requestID := diffing.NewRequestID()
	gateID := diffing.NewGateID()

	// Return empty result
	rows := pgxmock.NewRows([]string{
		"request_id", "request_gate_id", "request_agent_id", "request_method", "request_path",
		"request_headers", "request_body", "request_created_at",
		"response_id", "response_type", "response_status_code", "response_headers", "response_body", "response_created_at",
		"diff_id", "diff_from_response_id", "diff_to_response_id", "diff_content",
	})

	mock.ExpectQuery("SELECT (.+) FROM requests").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(rows)

	request, err := repo.GetByID(context.Background(), requestID, gateID)

	assert.Error(t, err)
	assert.Nil(t, request)
	assert.ErrorIs(t, err, diffing.ErrRequestNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRequestRepository_GetAllByGateID_success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := db.New(mock)
	repo := postgres.NewRequestRepository(queries, mock)

	gateID := diffing.NewGateID()
	req1ID := diffing.NewRequestID()
	req2ID := diffing.NewRequestID()
	now := time.Now()

	rows := pgxmock.NewRows([]string{
		"id", "gate_id", "agent_id", "method", "path", "headers", "body", "created_at",
	}).
		AddRow(
			pgtype.UUID{Bytes: req1ID.UUID(), Valid: true},
			pgtype.UUID{Bytes: gateID.UUID(), Valid: true},
			pgtype.Text{String: "", Valid: false},
			pgtype.Text{String: "GET", Valid: true},
			pgtype.Text{String: "/test1", Valid: true},
			[]byte(`{}`),
			[]byte(`{}`),
			pgtype.Timestamptz{Time: now, Valid: true},
		).
		AddRow(
			pgtype.UUID{Bytes: req2ID.UUID(), Valid: true},
			pgtype.UUID{Bytes: gateID.UUID(), Valid: true},
			pgtype.Text{String: "", Valid: false},
			pgtype.Text{String: "POST", Valid: true},
			pgtype.Text{String: "/test2", Valid: true},
			[]byte(`{}`),
			[]byte(`{}`),
			pgtype.Timestamptz{Time: now, Valid: true},
		)

	mock.ExpectQuery("SELECT (.+) FROM requests WHERE gate_id").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(rows)

	requests, err := repo.GetAllByGateID(context.Background(), gateID)

	assert.NoError(t, err)
	assert.Len(t, requests, 2)
	assert.Equal(t, req1ID.String(), requests[0].ID.String())
	assert.Equal(t, req2ID.String(), requests[1].ID.String())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRequestRepository_GetAllByGateID_empty_result(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := db.New(mock)
	repo := postgres.NewRequestRepository(queries, mock)

	gateID := diffing.NewGateID()

	rows := pgxmock.NewRows([]string{
		"id", "gate_id", "agent_id", "method", "path", "headers", "body", "created_at",
	})

	mock.ExpectQuery("SELECT (.+) FROM requests WHERE gate_id").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(rows)

	requests, err := repo.GetAllByGateID(context.Background(), gateID)

	assert.NoError(t, err)
	assert.Empty(t, requests)
	assert.NoError(t, mock.ExpectationsWereMet())
}
