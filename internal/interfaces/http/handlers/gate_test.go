package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pedrobarco/mroki/internal/application/commands"
	"github.com/pedrobarco/mroki/internal/application/queries"
	"github.com/pedrobarco/mroki/internal/domain/pagination"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/pedrobarco/mroki/pkg/dto"
)

// mockGateRepository implements traffictesting.GateRepository for testing
type mockGateRepository struct {
	saveFunc    func(ctx context.Context, gate *traffictesting.Gate) error
	getByIDFunc func(ctx context.Context, id traffictesting.GateID) (*traffictesting.Gate, error)
	getAllFunc  func(ctx context.Context, filters traffictesting.GateFilters, sort traffictesting.GateSort, params *pagination.Params) (*pagination.PagedResult[*traffictesting.Gate], error)
	deleteFunc  func(ctx context.Context, id traffictesting.GateID) error
}

func (m *mockGateRepository) Save(ctx context.Context, gate *traffictesting.Gate) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, gate)
	}
	return nil
}

func (m *mockGateRepository) GetByID(ctx context.Context, id traffictesting.GateID) (*traffictesting.Gate, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockGateRepository) Update(ctx context.Context, gate *traffictesting.Gate) error {
	return nil
}

func (m *mockGateRepository) Delete(ctx context.Context, id traffictesting.GateID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *mockGateRepository) GetAll(ctx context.Context, filters traffictesting.GateFilters, sort traffictesting.GateSort, params *pagination.Params) (*pagination.PagedResult[*traffictesting.Gate], error) {
	if m.getAllFunc != nil {
		return m.getAllFunc(ctx, filters, sort, params)
	}
	return nil, nil
}

// mockGateStatsRepository implements traffictesting.StatsRepository for gate handler testing
type mockGateStatsRepository struct {
	getStatsByGateIDsFn func(context.Context, []traffictesting.GateID) (map[traffictesting.GateID]traffictesting.GateStats, error)
}

func (m *mockGateStatsRepository) GetGlobalStats(ctx context.Context) (*traffictesting.GlobalStats, error) {
	return nil, errors.New("not implemented")
}

func (m *mockGateStatsRepository) GetStatsByGateIDs(ctx context.Context, ids []traffictesting.GateID) (map[traffictesting.GateID]traffictesting.GateStats, error) {
	if m.getStatsByGateIDsFn != nil {
		return m.getStatsByGateIDsFn(ctx, ids)
	}
	return map[traffictesting.GateID]traffictesting.GateStats{}, nil
}

func TestCreateGate_Success(t *testing.T) {
	repo := &mockGateRepository{
		saveFunc: func(ctx context.Context, gate *traffictesting.Gate) error {
			return nil
		},
	}
	handler := commands.NewCreateGateHandler(repo)

	body := `{"name":"test-gate","live_url":"http://live.example.com","shadow_url":"http://shadow.example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/gates", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	appHandler := CreateGate(handler)
	err := appHandler(rec, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rec.Code)
	}

	var response dto.Response[dto.Gate]
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Data.LiveURL != "http://live.example.com" {
		t.Errorf("expected live URL %s, got %s", "http://live.example.com", response.Data.LiveURL)
	}

	if response.Data.ShadowURL != "http://shadow.example.com" {
		t.Errorf("expected shadow URL %s, got %s", "http://shadow.example.com", response.Data.ShadowURL)
	}
}

func TestCreateGate_InvalidJSON(t *testing.T) {
	repo := &mockGateRepository{}
	handler := commands.NewCreateGateHandler(repo)

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/gates", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	appHandler := CreateGate(handler)
	err := appHandler(rec, req)

	apiErr, ok := err.(*dto.APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.Status != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", apiErr.Status)
	}
}

func TestCreateGate_MissingLiveURL(t *testing.T) {
	repo := &mockGateRepository{}
	handler := commands.NewCreateGateHandler(repo)

	body := `{"name":"test-gate","shadow_url":"http://shadow.example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/gates", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	appHandler := CreateGate(handler)
	err := appHandler(rec, req)

	apiErr, ok := err.(*dto.APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.Status != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", apiErr.Status)
	}

	if apiErr.Detail != "live_url is required" { //nolint:goconst
		t.Errorf("unexpected error detail: %s", apiErr.Detail)
	}
}

func TestCreateGate_MissingShadowURL(t *testing.T) {
	repo := &mockGateRepository{}
	handler := commands.NewCreateGateHandler(repo)

	body := `{"name":"test-gate","live_url":"http://live.example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/gates", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	appHandler := CreateGate(handler)
	err := appHandler(rec, req)

	apiErr, ok := err.(*dto.APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.Status != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", apiErr.Status)
	}

	if apiErr.Detail != "shadow_url is required" {
		t.Errorf("unexpected error detail: %s", apiErr.Detail)
	}
}

func TestCreateGate_InvalidURL(t *testing.T) {
	repo := &mockGateRepository{}
	handler := commands.NewCreateGateHandler(repo)

	body := `{"name":"test-gate","live_url":"not-a-url","shadow_url":"http://shadow.example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/gates", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	appHandler := CreateGate(handler)
	err := appHandler(rec, req)

	apiErr, ok := err.(*dto.APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.Status != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", apiErr.Status)
	}
}

func TestCreateGate_RepositoryError(t *testing.T) {
	repo := &mockGateRepository{
		saveFunc: func(ctx context.Context, gate *traffictesting.Gate) error {
			return errors.New("database error")
		},
	}
	handler := commands.NewCreateGateHandler(repo)

	body := `{"name":"test-gate","live_url":"http://live.example.com","shadow_url":"http://shadow.example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/gates", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	appHandler := CreateGate(handler)
	err := appHandler(rec, req)

	apiErr, ok := err.(*dto.APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.Status != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", apiErr.Status)
	}
}

func TestGetGateByID_Success(t *testing.T) {
	gateID := traffictesting.NewGateID()
	name, _ := traffictesting.ParseGateName("get-gate")
	liveURL, _ := traffictesting.ParseGateURL("http://live.example.com")
	shadowURL, _ := traffictesting.ParseGateURL("http://shadow.example.com")
	expectedGate, _ := traffictesting.NewGate(name, liveURL, shadowURL, traffictesting.WithGateID(gateID))

	repo := &mockGateRepository{
		getByIDFunc: func(ctx context.Context, id traffictesting.GateID) (*traffictesting.Gate, error) {
			if id != gateID {
				t.Errorf("unexpected gate ID: %s", id)
			}
			return expectedGate, nil
		},
	}
	handler := queries.NewGetGateHandler(repo, &mockGateStatsRepository{})

	req := httptest.NewRequest(http.MethodGet, "/gates/"+gateID.String(), nil)
	req.SetPathValue("gate_id", gateID.String())
	rec := httptest.NewRecorder()

	appHandler := GetGateByID(handler)
	err := appHandler(rec, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response dto.Response[dto.Gate]
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Data.ID != gateID.String() {
		t.Errorf("expected ID %s, got %s", gateID.String(), response.Data.ID)
	}

	// Verify stats field is present (zero values since domain object has no stats set)
	if response.Data.Stats.RequestCount24h != 0 {
		t.Errorf("expected RequestCount24h 0, got %d", response.Data.Stats.RequestCount24h)
	}
	if response.Data.Stats.DiffCount24h != 0 {
		t.Errorf("expected DiffCount24h 0, got %d", response.Data.Stats.DiffCount24h)
	}
	if response.Data.Stats.DiffRate != 0 {
		t.Errorf("expected DiffRate 0, got %f", response.Data.Stats.DiffRate)
	}
	if response.Data.Stats.LastActive != nil {
		t.Errorf("expected LastActive nil, got %v", response.Data.Stats.LastActive)
	}
}

func TestGetGateByID_StatsPopulated(t *testing.T) {
	gateID := traffictesting.NewGateID()
	name, _ := traffictesting.ParseGateName("stats-gate")
	liveURL, _ := traffictesting.ParseGateURL("http://live.example.com")
	shadowURL, _ := traffictesting.ParseGateURL("http://shadow.example.com")
	expectedGate, _ := traffictesting.NewGate(name, liveURL, shadowURL, traffictesting.WithGateID(gateID))

	lastActive := time.Now()

	repo := &mockGateRepository{
		getByIDFunc: func(ctx context.Context, id traffictesting.GateID) (*traffictesting.Gate, error) {
			return expectedGate, nil
		},
	}
	statsRepo := &mockGateStatsRepository{
		getStatsByGateIDsFn: func(ctx context.Context, ids []traffictesting.GateID) (map[traffictesting.GateID]traffictesting.GateStats, error) {
			return map[traffictesting.GateID]traffictesting.GateStats{
				gateID: {RequestCount24h: 100, DiffCount24h: 25, DiffRate: 25.0, LastActive: &lastActive},
			}, nil
		},
	}
	handler := queries.NewGetGateHandler(repo, statsRepo)

	req := httptest.NewRequest(http.MethodGet, "/gates/"+gateID.String(), nil)
	req.SetPathValue("gate_id", gateID.String())
	rec := httptest.NewRecorder()

	appHandler := GetGateByID(handler)
	err := appHandler(rec, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var response dto.Response[dto.Gate]
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Data.Stats.RequestCount24h != 100 {
		t.Errorf("expected RequestCount24h 100, got %d", response.Data.Stats.RequestCount24h)
	}
	if response.Data.Stats.DiffCount24h != 25 {
		t.Errorf("expected DiffCount24h 25, got %d", response.Data.Stats.DiffCount24h)
	}
	if response.Data.Stats.DiffRate != 25.0 {
		t.Errorf("expected DiffRate 25.0, got %f", response.Data.Stats.DiffRate)
	}
	if response.Data.Stats.LastActive == nil {
		t.Fatal("expected LastActive to be non-nil")
	}
}

func TestGetGateByID_MissingPathParam(t *testing.T) {
	repo := &mockGateRepository{}
	handler := queries.NewGetGateHandler(repo, &mockGateStatsRepository{})

	req := httptest.NewRequest(http.MethodGet, "/gates/", nil)
	rec := httptest.NewRecorder()

	appHandler := GetGateByID(handler)
	err := appHandler(rec, req)

	apiErr, ok := err.(*dto.APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.Status != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", apiErr.Status)
	}
}

func TestGetGateByID_InvalidID(t *testing.T) {
	repo := &mockGateRepository{
		getByIDFunc: func(ctx context.Context, id traffictesting.GateID) (*traffictesting.Gate, error) {
			return nil, traffictesting.ErrInvalidGateID
		},
	}
	handler := queries.NewGetGateHandler(repo, &mockGateStatsRepository{})

	req := httptest.NewRequest(http.MethodGet, "/gates/invalid-id", nil)
	req.SetPathValue("gate_id", "invalid-id")
	rec := httptest.NewRecorder()

	appHandler := GetGateByID(handler)
	err := appHandler(rec, req)

	apiErr, ok := err.(*dto.APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.Status != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", apiErr.Status)
	}
}

func TestGetGateByID_NotFound(t *testing.T) {
	repo := &mockGateRepository{
		getByIDFunc: func(ctx context.Context, id traffictesting.GateID) (*traffictesting.Gate, error) {
			return nil, traffictesting.ErrGateNotFound
		},
	}
	handler := queries.NewGetGateHandler(repo, &mockGateStatsRepository{})

	gateID := traffictesting.NewGateID()
	req := httptest.NewRequest(http.MethodGet, "/gates/"+gateID.String(), nil)
	req.SetPathValue("gate_id", gateID.String())
	rec := httptest.NewRecorder()

	appHandler := GetGateByID(handler)
	err := appHandler(rec, req)

	apiErr, ok := err.(*dto.APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.Status != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", apiErr.Status)
	}
}

func TestGetAllGates_Success(t *testing.T) {
	name1, _ := traffictesting.ParseGateName("gate-1")
	name2, _ := traffictesting.ParseGateName("gate-2")
	liveURL, _ := traffictesting.ParseGateURL("http://live.example.com")
	shadowURL, _ := traffictesting.ParseGateURL("http://shadow.example.com")
	gate1, _ := traffictesting.NewGate(name1, liveURL, shadowURL)
	gate2, _ := traffictesting.NewGate(name2, liveURL, shadowURL)
	gates := []*traffictesting.Gate{gate1, gate2}

	params, _ := pagination.NewParams(10, 0)
	result := pagination.NewPagedResult(gates, 2, params)

	repo := &mockGateRepository{
		getAllFunc: func(ctx context.Context, filters traffictesting.GateFilters, sort traffictesting.GateSort, p *pagination.Params) (*pagination.PagedResult[*traffictesting.Gate], error) {
			return result, nil
		},
	}
	handler := queries.NewListGatesHandler(repo, &mockGateStatsRepository{})

	req := httptest.NewRequest(http.MethodGet, "/gates?limit=10&offset=0", nil)
	rec := httptest.NewRecorder()

	appHandler := GetAllGates(handler)
	err := appHandler(rec, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response dto.PaginatedResponse[[]dto.Gate]
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.Data) != 2 {
		t.Errorf("expected 2 gates, got %d", len(response.Data))
	}

	if response.Pagination.Total != 2 {
		t.Errorf("expected total 2, got %d", response.Pagination.Total)
	}
}

func TestGetAllGates_EmptyResults(t *testing.T) {
	params, _ := pagination.NewParams(50, 0)
	result := pagination.NewPagedResult([]*traffictesting.Gate{}, 0, params)

	repo := &mockGateRepository{
		getAllFunc: func(ctx context.Context, filters traffictesting.GateFilters, sort traffictesting.GateSort, p *pagination.Params) (*pagination.PagedResult[*traffictesting.Gate], error) {
			return result, nil
		},
	}
	handler := queries.NewListGatesHandler(repo, &mockGateStatsRepository{})

	req := httptest.NewRequest(http.MethodGet, "/gates", nil)
	rec := httptest.NewRecorder()

	appHandler := GetAllGates(handler)
	err := appHandler(rec, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var response dto.PaginatedResponse[[]dto.Gate]
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.Data) != 0 {
		t.Errorf("expected 0 gates, got %d", len(response.Data))
	}

	if response.Pagination.Total != 0 {
		t.Errorf("expected total 0, got %d", response.Pagination.Total)
	}
}

func TestGetAllGates_InvalidPagination(t *testing.T) {
	repo := &mockGateRepository{}
	handler := queries.NewListGatesHandler(repo, &mockGateStatsRepository{})

	req := httptest.NewRequest(http.MethodGet, "/gates?limit=invalid", nil)
	rec := httptest.NewRecorder()

	appHandler := GetAllGates(handler)
	err := appHandler(rec, req)

	apiErr, ok := err.(*dto.APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.Status != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", apiErr.Status)
	}
}

func TestGetAllGates_PaginationValidationError(t *testing.T) {
	repo := &mockGateRepository{
		getAllFunc: func(ctx context.Context, filters traffictesting.GateFilters, sort traffictesting.GateSort, p *pagination.Params) (*pagination.PagedResult[*traffictesting.Gate], error) {
			return nil, traffictesting.ErrInvalidPagination
		},
	}
	handler := queries.NewListGatesHandler(repo, &mockGateStatsRepository{})

	req := httptest.NewRequest(http.MethodGet, "/gates?limit=-1", nil)
	rec := httptest.NewRecorder()

	appHandler := GetAllGates(handler)
	err := appHandler(rec, req)

	apiErr, ok := err.(*dto.APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.Status != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", apiErr.Status)
	}
}


func TestGetAllGates_WithSortParams(t *testing.T) {
	name, _ := traffictesting.ParseGateName("sort-gate")
	liveURL, _ := traffictesting.ParseGateURL("http://live.example.com")
	shadowURL, _ := traffictesting.ParseGateURL("http://shadow.example.com")
	gate1, _ := traffictesting.NewGate(name, liveURL, shadowURL)
	gates := []*traffictesting.Gate{gate1}

	params, _ := pagination.NewParams(50, 0)
	result := pagination.NewPagedResult(gates, 1, params)

	repo := &mockGateRepository{
		getAllFunc: func(ctx context.Context, filters traffictesting.GateFilters, sort traffictesting.GateSort, p *pagination.Params) (*pagination.PagedResult[*traffictesting.Gate], error) {
			return result, nil
		},
	}
	handler := queries.NewListGatesHandler(repo, &mockGateStatsRepository{})

	req := httptest.NewRequest(http.MethodGet, "/gates?sort=live_url&order=asc", nil)
	rec := httptest.NewRecorder()

	appHandler := GetAllGates(handler)
	err := appHandler(rec, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response dto.PaginatedResponse[[]dto.Gate]
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.Data) != 1 {
		t.Errorf("expected 1 gate, got %d", len(response.Data))
	}
}

func TestGetAllGates_WithFilterParams(t *testing.T) {
	name, _ := traffictesting.ParseGateName("filter-gate")
	liveURL, _ := traffictesting.ParseGateURL("http://live.example.com")
	shadowURL, _ := traffictesting.ParseGateURL("http://shadow.example.com")
	gate1, _ := traffictesting.NewGate(name, liveURL, shadowURL)
	gates := []*traffictesting.Gate{gate1}

	params, _ := pagination.NewParams(50, 0)
	result := pagination.NewPagedResult(gates, 1, params)

	repo := &mockGateRepository{
		getAllFunc: func(ctx context.Context, filters traffictesting.GateFilters, sort traffictesting.GateSort, p *pagination.Params) (*pagination.PagedResult[*traffictesting.Gate], error) {
			return result, nil
		},
	}
	handler := queries.NewListGatesHandler(repo, &mockGateStatsRepository{})

	req := httptest.NewRequest(http.MethodGet, "/gates?live_url=example&shadow_url=shadow", nil)
	rec := httptest.NewRecorder()

	appHandler := GetAllGates(handler)
	err := appHandler(rec, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestGetAllGates_InvalidSortField(t *testing.T) {
	repo := &mockGateRepository{}
	handler := queries.NewListGatesHandler(repo, &mockGateStatsRepository{})

	req := httptest.NewRequest(http.MethodGet, "/gates?sort=invalid_field", nil)
	rec := httptest.NewRecorder()

	appHandler := GetAllGates(handler)
	err := appHandler(rec, req)

	apiErr, ok := err.(*dto.APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.Status != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", apiErr.Status)
	}
}

func TestGetAllGates_InvalidSortOrder(t *testing.T) {
	repo := &mockGateRepository{}
	handler := queries.NewListGatesHandler(repo, &mockGateStatsRepository{})

	req := httptest.NewRequest(http.MethodGet, "/gates?order=invalid", nil)
	rec := httptest.NewRecorder()

	appHandler := GetAllGates(handler)
	err := appHandler(rec, req)

	apiErr, ok := err.(*dto.APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.Status != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", apiErr.Status)
	}
}

func TestGetAllGates_AllParamsCombined(t *testing.T) {
	name, _ := traffictesting.ParseGateName("combined-gate")
	liveURL, _ := traffictesting.ParseGateURL("http://live.example.com")
	shadowURL, _ := traffictesting.ParseGateURL("http://shadow.example.com")
	gate1, _ := traffictesting.NewGate(name, liveURL, shadowURL)

	params, _ := pagination.NewParams(10, 0)
	result := pagination.NewPagedResult([]*traffictesting.Gate{gate1}, 1, params)

	repo := &mockGateRepository{
		getAllFunc: func(ctx context.Context, filters traffictesting.GateFilters, sort traffictesting.GateSort, p *pagination.Params) (*pagination.PagedResult[*traffictesting.Gate], error) {
			return result, nil
		},
	}
	handler := queries.NewListGatesHandler(repo, &mockGateStatsRepository{})

	req := httptest.NewRequest(http.MethodGet, "/gates?limit=10&offset=0&live_url=example&shadow_url=shadow&sort=shadow_url&order=desc", nil)
	rec := httptest.NewRecorder()

	appHandler := GetAllGates(handler)
	err := appHandler(rec, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response dto.PaginatedResponse[[]dto.Gate]
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.Data) != 1 {
		t.Errorf("expected 1 gate, got %d", len(response.Data))
	}
}

func TestDeleteGate_Success(t *testing.T) {
	gateID := traffictesting.NewGateID()
	repo := &mockGateRepository{
		deleteFunc: func(ctx context.Context, id traffictesting.GateID) error {
			if id != gateID {
				t.Errorf("unexpected gate ID: %s", id)
			}
			return nil
		},
	}
	handler := commands.NewDeleteGateHandler(repo)

	req := httptest.NewRequest(http.MethodDelete, "/gates/"+gateID.String(), nil)
	req.SetPathValue("gate_id", gateID.String())
	rec := httptest.NewRecorder()

	appHandler := DeleteGate(handler)
	err := appHandler(rec, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", rec.Code)
	}
}

func TestDeleteGate_InvalidID(t *testing.T) {
	repo := &mockGateRepository{}
	handler := commands.NewDeleteGateHandler(repo)

	req := httptest.NewRequest(http.MethodDelete, "/gates/invalid-id", nil)
	req.SetPathValue("gate_id", "invalid-id")
	rec := httptest.NewRecorder()

	appHandler := DeleteGate(handler)
	err := appHandler(rec, req)

	apiErr, ok := err.(*dto.APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.Status != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", apiErr.Status)
	}
}

func TestDeleteGate_NotFound(t *testing.T) {
	repo := &mockGateRepository{
		deleteFunc: func(ctx context.Context, id traffictesting.GateID) error {
			return traffictesting.ErrGateNotFound
		},
	}
	handler := commands.NewDeleteGateHandler(repo)

	gateID := traffictesting.NewGateID()
	req := httptest.NewRequest(http.MethodDelete, "/gates/"+gateID.String(), nil)
	req.SetPathValue("gate_id", gateID.String())
	rec := httptest.NewRecorder()

	appHandler := DeleteGate(handler)
	err := appHandler(rec, req)

	apiErr, ok := err.(*dto.APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.Status != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", apiErr.Status)
	}
}

func TestDeleteGate_InternalError(t *testing.T) {
	repo := &mockGateRepository{
		deleteFunc: func(ctx context.Context, id traffictesting.GateID) error {
			return errors.New("database error")
		},
	}
	handler := commands.NewDeleteGateHandler(repo)

	gateID := traffictesting.NewGateID()
	req := httptest.NewRequest(http.MethodDelete, "/gates/"+gateID.String(), nil)
	req.SetPathValue("gate_id", gateID.String())
	rec := httptest.NewRecorder()

	appHandler := DeleteGate(handler)
	err := appHandler(rec, req)

	apiErr, ok := err.(*dto.APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.Status != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", apiErr.Status)
	}
}

func TestDeleteGate_MissingPathParam(t *testing.T) {
	repo := &mockGateRepository{}
	handler := commands.NewDeleteGateHandler(repo)

	req := httptest.NewRequest(http.MethodDelete, "/gates/", nil)
	rec := httptest.NewRecorder()

	appHandler := DeleteGate(handler)
	err := appHandler(rec, req)

	apiErr, ok := err.(*dto.APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.Status != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", apiErr.Status)
	}
}