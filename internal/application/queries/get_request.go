package queries

import (
	"context"
	"encoding/json"

	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/pedrobarco/mroki/pkg/diff"
)

// GetRequestQuery represents a query to retrieve a single request
type GetRequestQuery struct {
	ID     string
	GateID string
}

// GetRequestHandler handles the GetRequest query
type GetRequestHandler struct {
	repo traffictesting.RequestRepository
}

// NewGetRequestHandler creates a new GetRequestHandler
func NewGetRequestHandler(repo traffictesting.RequestRepository) *GetRequestHandler {
	return &GetRequestHandler{repo: repo}
}

// Handle executes the GetRequest query
func (h *GetRequestHandler) Handle(ctx context.Context, q GetRequestQuery) (*traffictesting.Request, error) {
	// Parse and validate IDs
	id, err := traffictesting.ParseRequestID(q.ID)
	if err != nil {
		return nil, err
	}

	gateID, err := traffictesting.ParseGateID(q.GateID)
	if err != nil {
		return nil, err
	}

	// Retrieve from repository
	req, err := h.repo.GetByID(ctx, id, gateID)
	if err != nil {
		return nil, err
	}

	// When sort_arrays is enabled, pre-sort response bodies so the
	// rendered data matches the diff patch indices.
	if req.Diff.Config.SortArrays {
		sortResponseBody(&req.LiveResponse)
		sortResponseBody(&req.ShadowResponse)
	}

	return req, nil
}

// sortResponseBody unmarshals the JSON body, sorts all arrays in-place
// using the same deterministic key as the diff engine, and re-marshals
// the result back into resp.Body. Non-JSON or empty bodies are left unchanged.
func sortResponseBody(resp *traffictesting.Response) {
	if len(resp.Body) == 0 {
		return
	}

	var tree any
	if err := json.Unmarshal(resp.Body, &tree); err != nil {
		return
	}

	diff.SortArraysInTree(tree)

	sorted, err := json.Marshal(tree)
	if err != nil {
		return
	}
	resp.Body = sorted
}
