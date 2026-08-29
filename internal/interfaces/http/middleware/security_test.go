package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pedrobarco/mroki/internal/interfaces/http/middleware"
	"github.com/stretchr/testify/assert"
)

func TestSecurityHeaders_always_on(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := middleware.SecurityHeaders(middleware.SecurityHeadersOptions{})
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	assert.Equal(t, "no-referrer", rec.Header().Get("Referrer-Policy"))
}

func TestSecurityHeaders_hsts_absent_when_disabled(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := middleware.SecurityHeaders(middleware.SecurityHeadersOptions{
		HSTSEnabled: false,
		HSTSMaxAge:  8760 * time.Hour,
	})
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	assert.Empty(t, rec.Header().Get("Strict-Transport-Security"),
		"HSTS must not be emitted when disabled")
}

func TestSecurityHeaders_hsts_present_when_enabled(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := middleware.SecurityHeaders(middleware.SecurityHeadersOptions{
		HSTSEnabled: true,
		HSTSMaxAge:  365 * 24 * time.Hour, // 8760h = 31536000s
	})
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	assert.Equal(t, "max-age=31536000; includeSubDomains",
		rec.Header().Get("Strict-Transport-Security"))
}

func TestSecurityHeaders_set_on_error_response(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})

	mw := middleware.SecurityHeaders(middleware.SecurityHeadersOptions{
		HSTSEnabled: true,
		HSTSMaxAge:  365 * 24 * time.Hour,
	})
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	assert.Equal(t, "no-referrer", rec.Header().Get("Referrer-Policy"))
	assert.Equal(t, "max-age=31536000; includeSubDomains",
		rec.Header().Get("Strict-Transport-Security"))
}
