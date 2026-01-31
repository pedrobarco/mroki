package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pedrobarco/mroki/internal/domain/diffing"
	"github.com/pedrobarco/mroki/internal/storage/postgres/db"
)

type requestRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

var _ diffing.RequestRepository = (*requestRepository)(nil)

func NewRequestRepository(queries *db.Queries, pool *pgxpool.Pool) *requestRepository {
	return &requestRepository{
		queries: queries,
		pool:    pool,
	}
}

func (r *requestRepository) Save(ctx context.Context, request *diffing.Request) error {
	headers, err := json.Marshal(request.Headers)
	if err != nil {
		return fmt.Errorf("failed to marshal headers: %w", err)
	}

	params := db.SaveRequestParams{
		ID:        pgtype.UUID{Bytes: request.ID.UUID(), Valid: true},
		GateID:    pgtype.UUID{Bytes: request.GateID.UUID(), Valid: true},
		Method:    pgtype.Text{String: request.Method, Valid: true},
		Path:      pgtype.Text{String: request.Path, Valid: true},
		Headers:   headers,
		Body:      request.Body,
		CreatedAt: pgtype.Timestamptz{Time: request.CreatedAt, Valid: true},
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	qtx := r.queries.WithTx(tx)
	defer func() {
		if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
			fmt.Printf("failed to rollback transaction: %v\n", err)
		}
	}()

	if err := qtx.SaveRequest(ctx, params); err != nil {
		return fmt.Errorf("failed to save request: %w", err)
	}

	for _, resp := range request.Responses {
		headers, err := json.Marshal(resp.Headers)
		if err != nil {
			return fmt.Errorf("failed to marshal headers: %w", err)
		}

		if err := qtx.SaveResponse(ctx, db.SaveResponseParams{
			ID:         pgtype.UUID{Bytes: resp.ID, Valid: true},
			Type:       pgtype.Text{String: string(resp.Type), Valid: true},
			RequestID:  pgtype.UUID{Bytes: request.ID.UUID(), Valid: true},
			StatusCode: pgtype.Int4{Int32: int32(resp.StatusCode), Valid: true},
			Headers:    headers,
			Body:       resp.Body,
			CreatedAt:  pgtype.Timestamptz{Time: resp.CreatedAt, Valid: true},
		}); err != nil {
			return fmt.Errorf("failed to save response: %w", err)
		}
	}

	if err := qtx.SaveDiff(ctx, db.SaveDiffParams{
		ID:             pgtype.UUID{Bytes: request.Diff.ID, Valid: true},
		RequestID:      pgtype.UUID{Bytes: request.ID.UUID(), Valid: true},
		FromResponseID: pgtype.UUID{Bytes: request.Diff.FromResponseID, Valid: true},
		ToResponseID:   pgtype.UUID{Bytes: request.Diff.ToResponseID, Valid: true},
		Content:        []byte(request.Diff.Content),
	}); err != nil {
		return fmt.Errorf("failed to save diff: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *requestRepository) GetAllByGateID(ctx context.Context, gateID diffing.GateID) ([]*diffing.Request, error) {
	rows, err := r.queries.GetAllRequestsByGateID(ctx, pgtype.UUID{Bytes: gateID.UUID(), Valid: true})
	if err != nil {
		return nil, err
	}

	var reqs []*diffing.Request
	for _, raw := range rows {
		req, err := r.toDomain(raw)
		if err != nil {
			return nil, fmt.Errorf("failed to convert gate: %w", err)
		}
		reqs = append(reqs, req)
	}
	return reqs, nil
}

func (r *requestRepository) GetByID(ctx context.Context, id diffing.RequestID, gateID diffing.GateID) (*diffing.Request, error) {
	rows, err := r.queries.GetRequestByID(ctx, db.GetRequestByIDParams{
		ID:     pgtype.UUID{Bytes: id.UUID(), Valid: true},
		GateID: pgtype.UUID{Bytes: gateID.UUID(), Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get request by ID: %w", err)
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("%w: %s for gate %s", diffing.ErrRequestNotFound, id, gateID)
	}

	req, err := r.toFullRequest(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}
	return req, nil
}

func (r *requestRepository) toDomain(raw db.Request) (*diffing.Request, error) {
	headers := make(http.Header)
	if err := json.Unmarshal(raw.Headers, &headers); err != nil {
		return nil, fmt.Errorf("failed to unmarshal headers: %w", err)
	}

	id, err := diffing.ParseRequestID(raw.ID.String())
	if err != nil {
		return nil, fmt.Errorf("invalid request ID in database: %w", err)
	}

	gateID, err := diffing.ParseGateID(raw.GateID.String())
	if err != nil {
		return nil, fmt.Errorf("invalid gate ID in database: %w", err)
	}

	return &diffing.Request{
		ID:        id,
		GateID:    gateID,
		Method:    raw.Method.String,
		Path:      raw.Path.String,
		Headers:   headers,
		Body:      raw.Body,
		CreatedAt: raw.CreatedAt.Time,
	}, nil
}

func (r *requestRepository) toFullRequest(rows []db.GetRequestByIDRow) (*diffing.Request, error) {
	reqHeaders := make(http.Header)
	if err := json.Unmarshal(rows[0].RequestHeaders, &reqHeaders); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request headers: %w", err)
	}

	id, err := diffing.ParseRequestID(rows[0].RequestID.String())
	if err != nil {
		return nil, fmt.Errorf("invalid request ID in database: %w", err)
	}

	gateID, err := diffing.ParseGateID(rows[0].RequestGateID.String())
	if err != nil {
		return nil, fmt.Errorf("invalid gate ID in database: %w", err)
	}

	req := &diffing.Request{
		ID:        id,
		GateID:    gateID,
		Method:    rows[0].RequestMethod.String,
		Path:      rows[0].RequestPath.String,
		Headers:   reqHeaders,
		Body:      rows[0].RequestBody,
		CreatedAt: rows[0].RequestCreatedAt.Time,
	}

	for _, row := range rows {
		respHeaders := make(http.Header)
		if err := json.Unmarshal(row.ResponseHeaders, &respHeaders); err != nil {
			return nil, fmt.Errorf("failed to unmarshal request headers: %w", err)
		}

		req.Responses = append(req.Responses, diffing.Response{
			ID:         row.ResponseID.Bytes,
			Type:       diffing.ResponseType(row.ResponseType.String),
			StatusCode: int(row.ResponseStatusCode.Int32),
			Headers:    respHeaders,
			Body:       row.ResponseBody,
			CreatedAt:  row.ResponseCreatedAt.Time,
		})
	}

	req.Diff = diffing.Diff{
		ID:             rows[0].DiffID.Bytes,
		FromResponseID: rows[0].DiffFromResponseID.Bytes,
		ToResponseID:   rows[0].DiffToResponseID.Bytes,
		Content:        string(rows[0].DiffContent),
	}

	return req, nil
}
