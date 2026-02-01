package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pedrobarco/mroki/internal/domain/pagination"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/pedrobarco/mroki/internal/infrastructure/persistence/postgres/db"
)

// TxBeginner is an interface for beginning transactions
type TxBeginner interface {
	Begin(context.Context) (pgx.Tx, error)
}

type requestRepository struct {
	queries *db.Queries
	pool    TxBeginner
}

var _ traffictesting.RequestRepository = (*requestRepository)(nil)

func NewRequestRepository(queries *db.Queries, pool TxBeginner) *requestRepository {
	return &requestRepository{
		queries: queries,
		pool:    pool,
	}
}

func (r *requestRepository) Save(ctx context.Context, request *traffictesting.Request) error {
	headers, err := json.Marshal(request.Headers)
	if err != nil {
		return fmt.Errorf("failed to marshal headers: %w", err)
	}

	params := db.SaveRequestParams{
		ID:        pgtype.UUID{Bytes: request.ID.UUID(), Valid: true},
		GateID:    pgtype.UUID{Bytes: request.GateID.UUID(), Valid: true},
		AgentID:   pgtype.Text{String: request.AgentID.String(), Valid: request.AgentID.String() != ""},
		Method:    request.Method,
		Path:      request.Path,
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
			Type:       string(resp.Type),
			RequestID:  pgtype.UUID{Bytes: request.ID.UUID(), Valid: true},
			StatusCode: int32(resp.StatusCode),
			Headers:    headers,
			Body:       resp.Body,
			CreatedAt:  pgtype.Timestamptz{Time: resp.CreatedAt, Valid: true},
		}); err != nil {
			return fmt.Errorf("failed to save response: %w", err)
		}
	}

	// Only save diff if it exists (has from/to response IDs)
	// Per domain rules: Request + 2 Responses always creates Diff (even if content empty)
	if !request.Diff.IsZero() {
		if err := qtx.SaveDiff(ctx, db.SaveDiffParams{
			RequestID:      pgtype.UUID{Bytes: request.ID.UUID(), Valid: true},
			FromResponseID: pgtype.UUID{Bytes: request.Diff.FromResponseID, Valid: true},
			ToResponseID:   pgtype.UUID{Bytes: request.Diff.ToResponseID, Valid: true},
			Content:        request.Diff.Content,
		}); err != nil {
			return fmt.Errorf("failed to save diff: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *requestRepository) GetAllByGateID(
	ctx context.Context,
	gateID traffictesting.GateID,
	filters traffictesting.RequestFilters,
	sort traffictesting.RequestSort,
	params *pagination.Params,
) (*pagination.PagedResult[*traffictesting.Request], error) {
	// Build sqlc parameters from domain value objects
	countParams := buildCountFilteredRequestsParams(gateID, filters)

	// Get total count with filters using sqlc-generated method
	total, err := r.queries.CountFilteredRequests(ctx, countParams)
	if err != nil {
		return nil, fmt.Errorf("failed to count requests: %w", err)
	}

	// Build parameters for paginated query
	queryParams := buildGetFilteredRequestsParams(gateID, filters, sort, params)

	// Execute query using sqlc-generated method
	rows, err := r.queries.GetFilteredRequests(ctx, queryParams)
	if err != nil {
		return nil, fmt.Errorf("failed to get requests: %w", err)
	}

	// Convert sqlc models to domain entities
	reqs := make([]*traffictesting.Request, 0, len(rows))
	for _, raw := range rows {
		req, err := r.toDomain(raw)
		if err != nil {
			return nil, fmt.Errorf("failed to convert request: %w", err)
		}
		reqs = append(reqs, req)
	}

	// Use factory to create PagedResult
	return pagination.NewPagedResult(reqs, total, params), nil
}

func (r *requestRepository) GetByID(ctx context.Context, id traffictesting.RequestID, gateID traffictesting.GateID) (*traffictesting.Request, error) {
	rows, err := r.queries.GetRequestByID(ctx, db.GetRequestByIDParams{
		ID:     pgtype.UUID{Bytes: id.UUID(), Valid: true},
		GateID: pgtype.UUID{Bytes: gateID.UUID(), Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get request by ID: %w", err)
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("%w: %s for gate %s", traffictesting.ErrRequestNotFound, id, gateID)
	}

	req, err := r.toFullRequest(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}
	return req, nil
}

func (r *requestRepository) toDomain(raw db.Request) (*traffictesting.Request, error) {
	headers := make(http.Header)
	if err := json.Unmarshal(raw.Headers, &headers); err != nil {
		return nil, fmt.Errorf("failed to unmarshal headers: %w", err)
	}

	id, err := traffictesting.ParseRequestID(raw.ID.String())
	if err != nil {
		return nil, fmt.Errorf("invalid request ID in database: %w", err)
	}

	gateID, err := traffictesting.ParseGateID(raw.GateID.String())
	if err != nil {
		return nil, fmt.Errorf("invalid gate ID in database: %w", err)
	}

	var agentID traffictesting.AgentID
	if raw.AgentID.Valid && raw.AgentID.String != "" {
		agentID, err = traffictesting.ParseAgentID(raw.AgentID.String)
		if err != nil {
			return nil, fmt.Errorf("invalid agent ID in database: %w", err)
		}
	}

	return &traffictesting.Request{
		ID:        id,
		GateID:    gateID,
		AgentID:   agentID,
		Method:    raw.Method,
		Path:      raw.Path,
		Headers:   headers,
		Body:      raw.Body,
		CreatedAt: raw.CreatedAt.Time,
	}, nil
}

func (r *requestRepository) toFullRequest(rows []db.GetRequestByIDRow) (*traffictesting.Request, error) {
	reqHeaders := make(http.Header)
	if err := json.Unmarshal(rows[0].RequestHeaders, &reqHeaders); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request headers: %w", err)
	}

	id, err := traffictesting.ParseRequestID(rows[0].RequestID.String())
	if err != nil {
		return nil, fmt.Errorf("invalid request ID in database: %w", err)
	}

	gateID, err := traffictesting.ParseGateID(rows[0].RequestGateID.String())
	if err != nil {
		return nil, fmt.Errorf("invalid gate ID in database: %w", err)
	}

	var agentID traffictesting.AgentID
	if rows[0].RequestAgentID.Valid && rows[0].RequestAgentID.String != "" {
		agentID, err = traffictesting.ParseAgentID(rows[0].RequestAgentID.String)
		if err != nil {
			return nil, fmt.Errorf("invalid agent ID in database: %w", err)
		}
	}

	req := &traffictesting.Request{
		ID:        id,
		GateID:    gateID,
		AgentID:   agentID,
		Method:    rows[0].RequestMethod,
		Path:      rows[0].RequestPath,
		Headers:   reqHeaders,
		Body:      rows[0].RequestBody,
		CreatedAt: rows[0].RequestCreatedAt.Time,
	}

	for _, row := range rows {
		respHeaders := make(http.Header)
		if err := json.Unmarshal(row.ResponseHeaders, &respHeaders); err != nil {
			return nil, fmt.Errorf("failed to unmarshal request headers: %w", err)
		}

		req.Responses = append(req.Responses, traffictesting.Response{
			ID:         row.ResponseID.Bytes,
			Type:       traffictesting.ResponseType(row.ResponseType.String),
			StatusCode: int(row.ResponseStatusCode.Int32),
			Headers:    respHeaders,
			Body:       row.ResponseBody,
			CreatedAt:  row.ResponseCreatedAt.Time,
		})
	}

	// Handle optional diff (LEFT JOIN can return NULL)
	if rows[0].DiffFromResponseID.Valid {
		req.Diff = traffictesting.Diff{
			FromResponseID: rows[0].DiffFromResponseID.Bytes,
			ToResponseID:   rows[0].DiffToResponseID.Bytes,
			Content:        rows[0].DiffContent.String,
		}
	}
	// If no diff exists, req.Diff remains zero value

	return req, nil
}
