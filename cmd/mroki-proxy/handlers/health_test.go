package handlers

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLiveness_always_ok(t *testing.T) {
	handler := Liveness()

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "text/plain; charset=utf-8", rec.Header().Get("Content-Type"))
		assert.Equal(t, "OK", rec.Body.String())
	}
}

func TestReadiness_ready(t *testing.T) {
	var ready atomic.Bool
	ready.Store(true)
	handler := Readiness(&ready)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "OK", rec.Body.String())
}

func TestReadiness_not_ready(t *testing.T) {
	var ready atomic.Bool // defaults to false

	handler := Readiness(&ready)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Equal(t, "NOT READY", rec.Body.String())
}

func TestReadiness_reflects_flag_changes(t *testing.T) {
	var ready atomic.Bool
	handler := Readiness(&ready)

	// Initially not ready.
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)

	// Flip to ready.
	ready.Store(true)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	assert.Equal(t, http.StatusOK, rec.Code)

	// Flip back (e.g. shutdown begins).
	ready.Store(false)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}
