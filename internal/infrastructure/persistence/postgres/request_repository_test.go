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
	"github.com/pedrobarco/mroki/internal/domain/pagination"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/pedrobarco/mroki/internal/infrastructure/persistence/postgres"
	"github.com/pedrobarco/mroki/internal/infrastructure/persistence/postgres/db"
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
	gateID := traffictesting.NewGateID()
	requestID := traffictesting.NewRequestID()
	liveRespID := uuid.New()
	shadowRespID := uuid.New()

	method, _ := traffictesting.NewHTTPMethod("POST")
	path, _ := traffictesting.ParsePath("/api/test")
	statusCode, _ := traffictesting.ParseStatusCode(200)

	request := &traffictesting.Request{
		ID:        requestID,
		GateID:    gateID,
		Method:    method,
		Path:      path,
		Headers:   traffictesting.NewHeaders(http.Header{"Content-Type": []string{"application/json"}}),
		Body:      []byte(`{"test":"data"}`),
		CreatedAt: time.Now(),
		Responses: []traffictesting.Response{
			{
				ID:         liveRespID,
				Type:       traffictesting.ResponseTypeLive,
				StatusCode: statusCode,
				Headers:    traffictesting.NewHeaders(http.Header{}),
				Body:       []byte(`{"status":"ok"}`),
				CreatedAt:  time.Now(),
			},
			{
				ID:         shadowRespID,
				Type:       traffictesting.ResponseTypeShadow,
				StatusCode: statusCode,
				Headers:    traffictesting.NewHeaders(http.Header{}),
				Body:       []byte(`{"status":"ok"}`),
				CreatedAt:  time.Now(),
			},
		},
		Diff: traffictesting.Diff{
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
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
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

	gateID := traffictesting.NewGateID()
	method, _ := traffictesting.NewHTTPMethod("POST")
	path, _ := traffictesting.ParsePath("/test")

	request := &traffictesting.Request{
		ID:        traffictesting.NewRequestID(),
		GateID:    gateID,
		Method:    method,
		Path:      path,
		Headers:   traffictesting.NewHeaders(http.Header{}),
		Body:      []byte{},
		CreatedAt: time.Now(),
		Responses: []traffictesting.Response{},
		Diff:      traffictesting.Diff{},
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

	gateID := traffictesting.NewGateID()
	method, _ := traffictesting.NewHTTPMethod("POST")
	path, _ := traffictesting.ParsePath("/test")

	request := &traffictesting.Request{
		ID:        traffictesting.NewRequestID(),
		GateID:    gateID,
		Method:    method,
		Path:      path,
		Headers:   traffictesting.NewHeaders(http.Header{}),
		Body:      []byte{},
		CreatedAt: time.Now(),
		Responses: []traffictesting.Response{},
		Diff:      traffictesting.Diff{},
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

	requestID := traffictesting.NewRequestID()
	gateID := traffictesting.NewGateID()
	liveRespID := uuid.New()
	shadowRespID := uuid.New()
	now := time.Now()

	// Mock the complex joined query result
	rows := pgxmock.NewRows([]string{
		"request_id", "request_gate_id", "request_agent_id", "request_method", "request_path",
		"request_headers", "request_body", "request_created_at",
		"response_id", "response_type", "response_status_code", "response_headers", "response_body", "response_created_at",
		"diff_from_response_id", "diff_to_response_id", "diff_content",
	}).
		AddRow(
			pgtype.UUID{Bytes: requestID.UUID(), Valid: true},
			pgtype.UUID{Bytes: gateID.UUID(), Valid: true},
			pgtype.Text{String: "", Valid: false},
			"GET",
			"/test",
			[]byte(`{}`),
			[]byte(`{"test":"data"}`),
			pgtype.Timestamptz{Time: now, Valid: true},
			pgtype.UUID{Bytes: liveRespID, Valid: true},
			pgtype.Text{String: "live", Valid: true},
			pgtype.Int4{Int32: 200, Valid: true},
			[]byte(`{}`),
			[]byte(`{"status":"ok"}`),
			pgtype.Timestamptz{Time: now, Valid: true},
			pgtype.UUID{Bytes: liveRespID, Valid: true},
			pgtype.UUID{Bytes: shadowRespID, Valid: true},
			pgtype.Text{String: "no diff", Valid: true},
		).
		AddRow(
			pgtype.UUID{Bytes: requestID.UUID(), Valid: true},
			pgtype.UUID{Bytes: gateID.UUID(), Valid: true},
			pgtype.Text{String: "", Valid: false},
			"GET",
			"/test",
			[]byte(`{}`),
			[]byte(`{"test":"data"}`),
			pgtype.Timestamptz{Time: now, Valid: true},
			pgtype.UUID{Bytes: shadowRespID, Valid: true},
			pgtype.Text{String: "shadow", Valid: true},
			pgtype.Int4{Int32: 200, Valid: true},
			[]byte(`{}`),
			[]byte(`{"status":"ok"}`),
			pgtype.Timestamptz{Time: now, Valid: true},
			pgtype.UUID{Bytes: liveRespID, Valid: true},
			pgtype.UUID{Bytes: shadowRespID, Valid: true},
			pgtype.Text{String: "no diff", Valid: true},
		)

	mock.ExpectQuery("SELECT (.+) FROM requests").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(rows)

	request, err := repo.GetByID(context.Background(), requestID, gateID)

	assert.NoError(t, err)
	assert.NotNil(t, request)
	assert.Equal(t, requestID.String(), request.ID.String())
	assert.Equal(t, gateID.String(), request.GateID.String())
	assert.Equal(t, "GET", request.Method.String())
	assert.Len(t, request.Responses, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRequestRepository_GetByID_not_found(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := db.New(mock)
	repo := postgres.NewRequestRepository(queries, mock)

	requestID := traffictesting.NewRequestID()
	gateID := traffictesting.NewGateID()

	// Return empty result
	rows := pgxmock.NewRows([]string{
		"request_id", "request_gate_id", "request_agent_id", "request_method", "request_path",
		"request_headers", "request_body", "request_created_at",
		"response_id", "response_type", "response_status_code", "response_headers", "response_body", "response_created_at",
		"diff_from_response_id", "diff_to_response_id", "diff_content",
	})

	mock.ExpectQuery("SELECT (.+) FROM requests").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(rows)

	request, err := repo.GetByID(context.Background(), requestID, gateID)

	assert.Error(t, err)
	assert.Nil(t, request)
	assert.ErrorIs(t, err, traffictesting.ErrRequestNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRequestRepository_GetAllByGateID_success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := db.New(mock)
	repo := postgres.NewRequestRepository(queries, mock)

	gateID := traffictesting.NewGateID()
	req1ID := traffictesting.NewRequestID()
	req2ID := traffictesting.NewRequestID()
	now := time.Now()

	params, _ := pagination.NewParams(50, 0)
	filters := traffictesting.EmptyRequestFilters()
	sort := traffictesting.DefaultRequestSort()

	// Expect CountFilteredRequests with all filter parameters
	// Parameters: gate_id, methods, path_pattern, from_date, to_date, agent_id, has_diff
	countRows := pgxmock.NewRows([]string{"count"}).AddRow(int64(2))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM requests WHERE`).
		WithArgs(
			pgtype.UUID{Bytes: gateID.UUID(), Valid: true},
			[]string{},                        // methods (empty slice, not nil)
			pgtype.Text{Valid: false},          // path_pattern
			pgtype.Timestamptz{Valid: false},   // from_date
			pgtype.Timestamptz{Valid: false},   // to_date
			pgtype.Text{Valid: false},          // agent_id
			pgtype.Bool{Valid: false},          // has_diff
		).
		WillReturnRows(countRows)

	// Expect GetFilteredRequests with all parameters including sort and pagination
	// Parameters: gate_id, methods, path_pattern, from_date, to_date, agent_id, has_diff, sort_order, sort_field, offset, limit
	rows := pgxmock.NewRows([]string{
		"id", "gate_id", "agent_id", "method", "path", "headers", "body", "created_at",
	}).
		AddRow(
			pgtype.UUID{Bytes: req1ID.UUID(), Valid: true},
			pgtype.UUID{Bytes: gateID.UUID(), Valid: true},
			pgtype.Text{String: "", Valid: false},
			"GET",
			"/test1",
			[]byte(`{}`),
			[]byte(`{}`),
			pgtype.Timestamptz{Time: now, Valid: true},
		).
		AddRow(
			pgtype.UUID{Bytes: req2ID.UUID(), Valid: true},
			pgtype.UUID{Bytes: gateID.UUID(), Valid: true},
			pgtype.Text{String: "", Valid: false},
			"POST",
			"/test2",
			[]byte(`{}`),
			[]byte(`{}`),
			pgtype.Timestamptz{Time: now, Valid: true},
		)

	mock.ExpectQuery(`SELECT id, gate_id, agent_id, method, path, headers, body, created_at FROM requests WHERE`).
		WithArgs(
			pgtype.UUID{Bytes: gateID.UUID(), Valid: true},
			[]string{},                        // methods (empty slice, not nil)
			pgtype.Text{Valid: false},          // path_pattern
			pgtype.Timestamptz{Valid: false},   // from_date
			pgtype.Timestamptz{Valid: false},   // to_date
			pgtype.Text{Valid: false},          // agent_id
			pgtype.Bool{Valid: false},          // has_diff
			"desc",                            // sort_order
			"created_at",                      // sort_field
			int32(0),                          // offset
			int32(50),                         // limit
		).
		WillReturnRows(rows)

	result, err := repo.GetAllByGateID(context.Background(), gateID, filters, sort, params)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, int64(2), result.Total)
	assert.Equal(t, 50, result.Limit)
	assert.Equal(t, 0, result.Offset)
	assert.False(t, result.HasMore)
	assert.Equal(t, req1ID.String(), result.Items[0].ID.String())
	assert.Equal(t, req2ID.String(), result.Items[1].ID.String())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRequestRepository_GetAllByGateID_empty_result(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := db.New(mock)
	repo := postgres.NewRequestRepository(queries, mock)

	gateID := traffictesting.NewGateID()
	params, _ := pagination.NewParams(50, 0)

	filters := traffictesting.EmptyRequestFilters()
	sort := traffictesting.DefaultRequestSort()

	// Expect CountFilteredRequests with all filter parameters returning 0
	countRows := pgxmock.NewRows([]string{"count"}).AddRow(int64(0))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM requests WHERE`).
		WithArgs(
			pgtype.UUID{Bytes: gateID.UUID(), Valid: true},
			[]string{},                        // methods (empty slice, not nil)
			pgtype.Text{Valid: false},          // path_pattern
			pgtype.Timestamptz{Valid: false},   // from_date
			pgtype.Timestamptz{Valid: false},   // to_date
			pgtype.Text{Valid: false},          // agent_id
			pgtype.Bool{Valid: false},          // has_diff
		).
		WillReturnRows(countRows)

	// Expect GetFilteredRequests returning empty result
	rows := pgxmock.NewRows([]string{
		"id", "gate_id", "agent_id", "method", "path", "headers", "body", "created_at",
	})

	mock.ExpectQuery(`SELECT id, gate_id, agent_id, method, path, headers, body, created_at FROM requests WHERE`).
		WithArgs(
			pgtype.UUID{Bytes: gateID.UUID(), Valid: true},
			[]string{},                        // methods (empty slice, not nil)
			pgtype.Text{Valid: false},          // path_pattern
			pgtype.Timestamptz{Valid: false},   // from_date
			pgtype.Timestamptz{Valid: false},   // to_date
			pgtype.Text{Valid: false},          // agent_id
			pgtype.Bool{Valid: false},          // has_diff
			"desc",                            // sort_order
			"created_at",                      // sort_field
			int32(0),                          // offset
			int32(50),                         // limit
		).
		WillReturnRows(rows)

	result, err := repo.GetAllByGateID(context.Background(), gateID, filters, sort, params)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Items)
	assert.Equal(t, int64(0), result.Total)
	assert.False(t, result.HasMore)
	assert.NoError(t, mock.ExpectationsWereMet())
}
