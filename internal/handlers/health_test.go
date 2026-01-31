package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pedrobarco/mroki/internal/handlers"
	"github.com/stretchr/testify/assert"
)

// Mock HealthChecker
type mockHealthChecker struct {
	pingError error
}

func (m *mockHealthChecker) Ping(ctx context.Context) error {
	return m.pingError
}

func TestLiveness_returns_200_ok(t *testing.T) {
	handler := handlers.Liveness()

	req := httptest.NewRequest("GET", "/health/live", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "OK", rec.Body.String())
	assert.Equal(t, "text/plain; charset=utf-8", rec.Header().Get("Content-Type"))
}

func TestReadiness_database_healthy(t *testing.T) {
	db := &mockHealthChecker{pingError: nil}
	handler := handlers.Readiness(db)

	req := httptest.NewRequest("GET", "/health/ready", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "OK", rec.Body.String())
	assert.Equal(t, "text/plain; charset=utf-8", rec.Header().Get("Content-Type"))
}

func TestReadiness_database_unhealthy(t *testing.T) {
	db := &mockHealthChecker{pingError: errors.New("connection refused")}
	handler := handlers.Readiness(db)

	req := httptest.NewRequest("GET", "/health/ready", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Contains(t, rec.Body.String(), "database health check failed")
	assert.Contains(t, rec.Body.String(), "connection refused")
	assert.Equal(t, "text/plain; charset=utf-8", rec.Header().Get("Content-Type"))
}
