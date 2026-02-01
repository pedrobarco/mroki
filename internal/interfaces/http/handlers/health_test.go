package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// mockHealthChecker implements HealthChecker for testing
type mockHealthChecker struct {
	pingFunc func(ctx context.Context) error
}

func (m *mockHealthChecker) Ping(ctx context.Context) error {
	if m.pingFunc != nil {
		return m.pingFunc(ctx)
	}
	return nil
}

func TestLiveness_Success(t *testing.T) {
	handler := Liveness()

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "text/plain; charset=utf-8" {
		t.Errorf("expected Content-Type 'text/plain; charset=utf-8', got %q", contentType)
	}

	body := rec.Body.String()
	if body != "OK" {
		t.Errorf("expected body 'OK', got %q", body)
	}
}

func TestLiveness_AlwaysSucceeds(t *testing.T) {
	// Liveness should always return 200 OK, regardless of circumstances
	handler := Liveness()

	// Make multiple requests to ensure consistent behavior
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected status 200, got %d", i, rec.Code)
		}
	}
}

func TestReadiness_Success(t *testing.T) {
	mock := &mockHealthChecker{
		pingFunc: func(ctx context.Context) error {
			return nil
		},
	}

	handler := Readiness(mock)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "text/plain; charset=utf-8" {
		t.Errorf("expected Content-Type 'text/plain; charset=utf-8', got %q", contentType)
	}

	body := rec.Body.String()
	if body != "OK" {
		t.Errorf("expected body 'OK', got %q", body)
	}
}

func TestReadiness_DatabaseFailure(t *testing.T) {
	dbErr := errors.New("connection refused")
	mock := &mockHealthChecker{
		pingFunc: func(ctx context.Context) error {
			return dbErr
		},
	}

	handler := Readiness(mock)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "text/plain; charset=utf-8" {
		t.Errorf("expected Content-Type 'text/plain; charset=utf-8', got %q", contentType)
	}

	body := rec.Body.String()
	expectedPrefix := "database health check failed:"
	if !strings.HasPrefix(body, expectedPrefix) {
		t.Errorf("expected body to start with %q, got %q", expectedPrefix, body)
	}

	if !strings.Contains(body, "connection refused") {
		t.Errorf("expected body to contain error message, got %q", body)
	}
}

func TestReadiness_ContextTimeout(t *testing.T) {
	// Test that the handler respects the 1-second timeout
	mock := &mockHealthChecker{
		pingFunc: func(ctx context.Context) error {
			// Verify that the context has a timeout set
			_, hasDeadline := ctx.Deadline()
			if !hasDeadline {
				t.Error("expected context to have a deadline")
			}
			return nil
		},
	}

	handler := Readiness(mock)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestReadiness_ContextCancellation(t *testing.T) {
	// Test that handler handles context-related errors properly
	// The test request's context is not cancelled, so we simulate a DB error
	mock := &mockHealthChecker{
		pingFunc: func(ctx context.Context) error {
			// Simulate a context cancellation error
			return context.Canceled
		},
	}

	handler := Readiness(mock)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Context cancellation error should be treated as a database failure
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", rec.Code)
	}
}

func TestReadiness_DifferentErrorTypes(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "network error",
			err:  errors.New("dial tcp: connection refused"),
		},
		{
			name: "timeout error",
			err:  errors.New("timeout waiting for connection"),
		},
		{
			name: "authentication error",
			err:  errors.New("authentication failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockHealthChecker{
				pingFunc: func(ctx context.Context) error {
					return tt.err
				},
			}

			handler := Readiness(mock)

			req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusServiceUnavailable {
				t.Errorf("expected status 503, got %d", rec.Code)
			}

			body := rec.Body.String()
			if !strings.Contains(body, tt.err.Error()) {
				t.Errorf("expected body to contain %q, got %q", tt.err.Error(), body)
			}
		})
	}
}
