package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pedrobarco/mroki/internal/interfaces/http/middleware"
	"github.com/stretchr/testify/assert"
)

func TestRequestID_generates_id(t *testing.T) {
	var ctxID string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxID = middleware.GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	mw := middleware.RequestID()
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	assert.NotEmpty(t, ctxID, "should generate a request ID")
	assert.Equal(t, ctxID, rec.Header().Get("X-Request-ID"), "response header should match context ID")
}

func TestRequestID_reuses_header(t *testing.T) {
	const existingID = "my-custom-request-id"

	var ctxID string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxID = middleware.GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	mw := middleware.RequestID()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", existingID)
	rec := httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	assert.Equal(t, existingID, ctxID, "should reuse existing request ID from header")
}

func TestRequestID_sets_response_header(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := middleware.RequestID()
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	responseID := rec.Header().Get("X-Request-ID")
	assert.NotEmpty(t, responseID, "response should have X-Request-ID header")
}

func TestGetRequestID_from_context(t *testing.T) {
	const expectedID = "test-request-id-123"

	var ctxID string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxID = middleware.GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	mw := middleware.RequestID()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", expectedID)
	rec := httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	assert.Equal(t, expectedID, ctxID, "GetRequestID should return the ID from context")
}

func TestGetRequestID_missing_context(t *testing.T) {
	id := middleware.GetRequestID(context.Background())
	assert.Empty(t, id, "GetRequestID should return empty string for bare context")
}
