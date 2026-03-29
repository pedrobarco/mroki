package ent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/pedrobarco/mroki/ent"
	"github.com/pedrobarco/mroki/ent/predicate"
	"github.com/pedrobarco/mroki/ent/request"
	"github.com/pedrobarco/mroki/internal/domain/pagination"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
)

type requestRepository struct {
	client *ent.Client
}

var _ traffictesting.RequestRepository = (*requestRepository)(nil)

func NewRequestRepository(client *ent.Client) *requestRepository {
	return &requestRepository{client: client}
}

func (r *requestRepository) Save(ctx context.Context, req *traffictesting.Request) error {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := recover(); err != nil {
			_ = tx.Rollback()
			panic(err)
		}
	}()

	if err := r.saveRequest(ctx, tx, req); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := r.saveResponses(ctx, tx, req); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := r.saveDiff(ctx, tx, req); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (r *requestRepository) saveRequest(ctx context.Context, tx *ent.Tx, req *traffictesting.Request) error {
	builder := tx.Request.Create().
		SetID(req.ID.UUID()).
		SetGateID(req.GateID.UUID()).
		SetMethod(req.Method.String()).
		SetPath(req.Path.String()).
		SetHeaders(req.Headers.HTTPHeader()).
		SetBody(req.Body).
		SetCreatedAt(req.CreatedAt)

	if _, err := builder.Save(ctx); err != nil {
		return fmt.Errorf("failed to save request: %w", err)
	}
	return nil
}

func (r *requestRepository) saveResponses(ctx context.Context, tx *ent.Tx, req *traffictesting.Request) error {
	for _, resp := range req.Responses {
		if _, err := tx.Response.Create().
			SetID(resp.ID).
			SetRequestID(req.ID.UUID()).
			SetType(string(resp.Type)).
			SetStatusCode(int32(resp.StatusCode.Int())).
			SetHeaders(resp.Headers.HTTPHeader()).
			SetBody(resp.Body).
			SetCreatedAt(resp.CreatedAt).
			Save(ctx); err != nil {
			return fmt.Errorf("failed to save response: %w", err)
		}
	}
	return nil
}

func (r *requestRepository) saveDiff(ctx context.Context, tx *ent.Tx, req *traffictesting.Request) error {
	if req.Diff.IsZero() {
		return nil
	}

	if _, err := tx.Diff.Create().
		SetRequestID(req.ID.UUID()).
		SetFromResponseID(req.Diff.FromResponseID).
		SetToResponseID(req.Diff.ToResponseID).
		SetContent(req.Diff.Content).
		SetCreatedAt(req.Diff.CreatedAt).
		Save(ctx); err != nil {
		return fmt.Errorf("failed to save diff: %w", err)
	}
	return nil
}


func (r *requestRepository) GetByID(ctx context.Context, id traffictesting.RequestID, gateID traffictesting.GateID) (*traffictesting.Request, error) {
	raw, err := r.client.Request.Query().
		Where(
			request.ID(id.UUID()),
			request.GateID(gateID.UUID()),
		).
		WithResponses().
		WithDiff().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("%w: %s for gate %s", traffictesting.ErrRequestNotFound, id, gateID)
		}
		return nil, fmt.Errorf("failed to get request by ID: %w", err)
	}

	req, err := mapRequestToDomain(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}

	// Map eager-loaded responses
	for _, respRaw := range raw.Edges.Responses {
		resp, err := mapResponseToDomain(respRaw)
		if err != nil {
			return nil, fmt.Errorf("failed to convert response: %w", err)
		}
		req.Responses = append(req.Responses, resp)
	}

	// Map eager-loaded diff
	req.Diff = mapDiffToDomain(raw.Edges.Diff)

	return req, nil
}

func (r *requestRepository) GetAllByGateID(
	ctx context.Context,
	gateID traffictesting.GateID,
	filters traffictesting.RequestFilters,
	sort traffictesting.RequestSort,
	params *pagination.Params,
) (*pagination.PagedResult[*traffictesting.Request], error) {
	// Build predicates — shared between count and list
	preds := r.buildPredicates(gateID, filters)
	where := request.And(preds...)

	total, err := r.client.Request.Query().Where(where).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count requests: %w", err)
	}

	rows, err := r.client.Request.Query().
		Where(where).
		Order(r.buildOrderBy(sort)).
		Limit(params.Limit()).
		Offset(params.Offset()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get requests: %w", err)
	}

	reqs := make([]*traffictesting.Request, 0, len(rows))
	for _, raw := range rows {
		req, err := mapRequestToDomain(raw)
		if err != nil {
			return nil, fmt.Errorf("failed to convert request: %w", err)
		}
		reqs = append(reqs, req)
	}

	return pagination.NewPagedResult(reqs, int64(total), params), nil
}

func (r *requestRepository) DeleteOlderThan(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)

	count, err := r.client.Request.Delete().
		Where(request.CreatedAtLT(cutoff)).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired requests: %w", err)
	}

	slog.Info("deleted expired requests", "count", count)
	return int64(count), nil
}

// buildPredicates composes Ent predicates from domain filters.
func (r *requestRepository) buildPredicates(gateID traffictesting.GateID, filters traffictesting.RequestFilters) []predicate.Request {
	preds := []predicate.Request{
		request.GateID(gateID.UUID()),
	}

	if filters.HasMethodFilter() {
		methods := filters.Methods()
		strs := make([]string, len(methods))
		for i, m := range methods {
			strs[i] = m.String()
		}
		preds = append(preds, request.MethodIn(strs...))
	}

	if filters.HasPathFilter() {
		pattern := filters.PathPattern().String()
		pattern = strings.ReplaceAll(pattern, "*", "%")
		preds = append(preds, request.PathContains(strings.ReplaceAll(pattern, "%", "")))
	}

	dateRange := filters.DateRange()
	if dateRange.HasFrom() {
		preds = append(preds, request.CreatedAtGTE(*dateRange.From()))
	}
	if dateRange.HasTo() {
		preds = append(preds, request.CreatedAtLTE(*dateRange.To()))
	}

	if filters.HasDiffFilter() {
		if *filters.HasDiff() {
			preds = append(preds, request.HasDiff())
		} else {
			preds = append(preds, request.Not(request.HasDiff()))
		}
	}

	return preds
}

// buildOrderBy maps domain sort to an Ent order option.
func (r *requestRepository) buildOrderBy(sort traffictesting.RequestSort) request.OrderOption {
	var opts []sql.OrderTermOption
	if sort.Order().IsAsc() {
		opts = append(opts, sql.OrderAsc())
	} else {
		opts = append(opts, sql.OrderDesc())
	}

	field := sort.Field()
	switch {
	case field.IsMethod():
		return request.ByMethod(opts...)
	case field.IsPath():
		return request.ByPath(opts...)
	default:
		return request.ByCreatedAt(opts...)
	}
}