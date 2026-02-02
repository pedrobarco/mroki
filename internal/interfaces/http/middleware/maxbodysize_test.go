package middleware_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pedrobarco/mroki/internal/interfaces/http/middleware"
)

func TestMaxBodySize(t *testing.T) {
	maxSize := int64(10) // 10 bytes for testing

	handler := middleware.Chain{
		middleware.MaxBodySize(maxSize),
	}.Then(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to read body
		_, err := io.ReadAll(r.Body)
		if err != nil {
			// MaxBytesReader returns an error when limit exceeded
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name           string
		bodySize       int
		expectedStatus int
	}{
		{"under limit", 5, http.StatusOK},
		{"at limit", 10, http.StatusOK},
		{"over limit by 1 byte", 11, http.StatusRequestEntityTooLarge},
		{"way over limit", 100, http.StatusRequestEntityTooLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := bytes.Repeat([]byte("a"), tt.bodySize)
			req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestMaxBodySize_GET_requests(t *testing.T) {
	maxSize := int64(10)

	handler := middleware.Chain{
		middleware.MaxBodySize(maxSize),
	}.Then(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d for GET request, got %d", http.StatusOK, rec.Code)
	}
}

func TestMaxBodySize_no_body(t *testing.T) {
	maxSize := int64(10)

	handler := middleware.Chain{
		middleware.MaxBodySize(maxSize),
	}.Then(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("unexpected error reading empty body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d for empty body, got %d", http.StatusOK, rec.Code)
	}
}
