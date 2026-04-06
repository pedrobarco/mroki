package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pedrobarco/mroki/internal/application/queries"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/pedrobarco/mroki/pkg/dto"
)

// mockStatsRepository implements traffictesting.StatsRepository for testing
type mockStatsRepository struct {
	getGlobalStatsFunc func(ctx context.Context) (*traffictesting.GlobalStats, error)
}

func (m *mockStatsRepository) GetGlobalStats(ctx context.Context) (*traffictesting.GlobalStats, error) {
	if m.getGlobalStatsFunc != nil {
		return m.getGlobalStatsFunc(ctx)
	}
	return nil, nil
}

func TestGetGlobalStats_Success(t *testing.T) {
	repo := &mockStatsRepository{
		getGlobalStatsFunc: func(ctx context.Context) (*traffictesting.GlobalStats, error) {
			return &traffictesting.GlobalStats{
				TotalGates:       5,
				TotalRequests24h: 1234,
				TotalDiffRate:    12.5,
			}, nil
		},
	}
	handler := queries.NewGetGlobalStatsHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	rec := httptest.NewRecorder()

	appHandler := GetGlobalStats(handler)
	err := appHandler(rec, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response dto.Response[dto.GlobalStats]
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Data.TotalGates != 5 {
		t.Errorf("expected TotalGates 5, got %d", response.Data.TotalGates)
	}

	if response.Data.TotalRequests24h != 1234 {
		t.Errorf("expected TotalRequests24h 1234, got %d", response.Data.TotalRequests24h)
	}

	if response.Data.TotalDiffRate != 12.5 {
		t.Errorf("expected TotalDiffRate 12.5, got %f", response.Data.TotalDiffRate)
	}
}

func TestGetGlobalStats_RepositoryError(t *testing.T) {
	repo := &mockStatsRepository{
		getGlobalStatsFunc: func(ctx context.Context) (*traffictesting.GlobalStats, error) {
			return nil, errors.New("database error")
		},
	}
	handler := queries.NewGetGlobalStatsHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	rec := httptest.NewRecorder()

	appHandler := GetGlobalStats(handler)
	err := appHandler(rec, req)

	apiErr, ok := err.(*dto.APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.Status != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", apiErr.Status)
	}
}
