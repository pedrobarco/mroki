package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pedrobarco/mroki/internal/interfaces/http/middleware"
	"github.com/stretchr/testify/assert"
)

func TestAPIKeyAuth_ValidKey(t *testing.T) {
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	errorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		t.Fatal("error handler should not be called")
	}

	mw := middleware.APIKeyAuth("test-key-123",
		middleware.WithAuthErrorHandler(errorHandler),
	)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer test-key-123")
	rec := httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, called, "handler should be called with valid key")
}

func TestAPIKeyAuth_MissingHeader(t *testing.T) {
	var capturedErr error

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	errorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		capturedErr = err
		w.WriteHeader(http.StatusUnauthorized)
	}

	mw := middleware.APIKeyAuth("test-key-123",
		middleware.WithAuthErrorHandler(errorHandler),
	)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.ErrorIs(t, capturedErr, middleware.ErrMissingAuthHeader)
}

func TestAPIKeyAuth_InvalidFormat_MissingBearer(t *testing.T) {
	var capturedErr error

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	errorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		capturedErr = err
		w.WriteHeader(http.StatusUnauthorized)
	}

	mw := middleware.APIKeyAuth("test-key-123",
		middleware.WithAuthErrorHandler(errorHandler),
	)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "test-key-123") // Missing "Bearer " prefix
	rec := httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.ErrorIs(t, capturedErr, middleware.ErrInvalidAuthFormat)
}

func TestAPIKeyAuth_InvalidFormat_EmptyToken(t *testing.T) {
	var capturedErr error

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	errorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		capturedErr = err
		w.WriteHeader(http.StatusUnauthorized)
	}

	mw := middleware.APIKeyAuth("test-key-123",
		middleware.WithAuthErrorHandler(errorHandler),
	)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer ") // Empty token after Bearer
	rec := httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.ErrorIs(t, capturedErr, middleware.ErrInvalidAuthFormat)
}

func TestAPIKeyAuth_InvalidKey(t *testing.T) {
	var capturedErr error

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	errorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		capturedErr = err
		w.WriteHeader(http.StatusUnauthorized)
	}

	mw := middleware.APIKeyAuth("test-key-123",
		middleware.WithAuthErrorHandler(errorHandler),
	)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer wrong-key")
	rec := httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.ErrorIs(t, capturedErr, middleware.ErrInvalidAPIKey)
}

func TestAPIKeyAuth_CustomExtractor(t *testing.T) {
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	customExtractor := func(authHeader string) (string, bool) {
		// Custom format: "ApiKey <token>"
		const prefix = "ApiKey "
		if !strings.HasPrefix(authHeader, prefix) {
			return "", false
		}
		token := strings.TrimPrefix(authHeader, prefix)
		if token == "" {
			return "", false
		}
		return token, true
	}

	errorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		t.Fatalf("error handler should not be called, got error: %v", err)
	}

	mw := middleware.APIKeyAuth("test-key-123",
		middleware.WithAuthErrorHandler(errorHandler),
		middleware.WithTokenExtractor(customExtractor),
	)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "ApiKey test-key-123")
	rec := httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, called, "handler should be called with valid custom format")
}

func TestAPIKeyAuth_DefaultErrorHandler(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	// No error handler provided - uses default
	mw := middleware.APIKeyAuth("test-key-123")

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Unauthorized")
}

func TestAPIKeyAuth_WithMultipleOptions(t *testing.T) {
	called := false
	var capturedErr error

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	customExtractor := func(authHeader string) (string, bool) {
		return authHeader, authHeader != ""
	}

	errorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		capturedErr = err
		w.WriteHeader(http.StatusUnauthorized)
	}

	mw := middleware.APIKeyAuth("test-key-123",
		middleware.WithAuthErrorHandler(errorHandler),
		middleware.WithTokenExtractor(customExtractor),
	)

	// Test with valid key
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "test-key-123")
	rec := httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, called)

	// Test with invalid key
	called = false
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "wrong-key")
	rec = httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.False(t, called)
	assert.ErrorIs(t, capturedErr, middleware.ErrInvalidAPIKey)
}
