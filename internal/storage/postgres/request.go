package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pedrobarco/mroki/internal/domain/diffing"
	"github.com/pedrobarco/mroki/internal/storage/postgres/db"
)

type requestRepository struct {
	queries *db.Queries
	conn    *pgx.Conn
}

var _ diffing.RequestRepository = (*requestRepository)(nil)

func NewRequestRepository(queries *db.Queries, conn *pgx.Conn) *requestRepository {
	return &requestRepository{
		queries: queries,
		conn:    conn,
	}
}

func (r *requestRepository) Save(ctx context.Context, request *diffing.Request) error {
	headers, err := json.Marshal(request.Headers)
	if err != nil {
		return fmt.Errorf("failed to marshal headers: %w", err)
	}

	params := db.SaveRequestParams{
		ID:        pgtype.UUID{Bytes: request.ID, Valid: true},
		GateID:    pgtype.UUID{Bytes: request.GateID, Valid: true},
		Method:    pgtype.Text{String: request.Method, Valid: true},
		Path:      pgtype.Text{String: request.Path, Valid: true},
		Headers:   headers,
		Body:      request.Body,
		CreatedAt: pgtype.Timestamptz{Time: request.CreatedAt, Valid: true},
	}

	tx, err := r.conn.Begin(ctx)
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
			RequestID:  pgtype.UUID{Bytes: request.ID, Valid: true},
			StatusCode: pgtype.Int4{Int32: int32(resp.StatusCode), Valid: true},
			Headers:    headers,
			Body:       resp.Body,
			CreatedAt:  pgtype.Timestamptz{Time: resp.CreatedAt, Valid: true},
		}); err != nil {
			return fmt.Errorf("failed to save response: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *requestRepository) GetAllByGateID(ctx context.Context, gateID uuid.UUID) ([]*diffing.Request, error) {
	rows, err := r.queries.GetAllRequestsByGateID(ctx, pgtype.UUID{Bytes: gateID, Valid: true})
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

func (r *requestRepository) GetByID(ctx context.Context, id uuid.UUID, gateID uuid.UUID) (*diffing.Request, error) {
	raw, err := r.queries.GetRequestByID(ctx, db.GetRequestByIDParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		GateID: pgtype.UUID{Bytes: gateID, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get request by ID: %w", err)
	}

	req, err := r.toDomain(raw)
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

	return &diffing.Request{
		ID:        raw.ID.Bytes,
		GateID:    raw.GateID.Bytes,
		Method:    raw.Method.String,
		Path:      raw.Path.String,
		Headers:   headers,
		Body:      raw.Body,
		CreatedAt: raw.CreatedAt.Time,
	}, nil
}
