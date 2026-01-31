package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAppHandler_ServeHTTP_success(t *testing.T) {
	handler := AppHandler(func(w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestAppHandler_ServeHTTP_api_error(t *testing.T) {
	handler := AppHandler(func(w http.ResponseWriter, r *http.Request) error {
		return NewError(http.StatusNotFound, "resource not found", errors.New("not found"))
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
	}
}

func TestAppHandler_ServeHTTP_unknown_error(t *testing.T) {
	handler := AppHandler(func(w http.ResponseWriter, r *http.Request) error {
		return errors.New("some random error")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestAPIError_Error(t *testing.T) {
	err := errors.New("underlying error")
	apiErr := NewError(http.StatusBadRequest, "bad request", err)

	if apiErr.Error() != "underlying error" {
		t.Errorf("expected 'underlying error', got '%s'", apiErr.Error())
	}
}

func TestInvalidResponseBody(t *testing.T) {
	err := errors.New("encoding failed")
	apiErr := InvalidResponseBody(err)

	if apiErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", apiErr.StatusCode)
	}

	if apiErr.Message != "failed to encode response body" {
		t.Errorf("expected message 'failed to encode response body', got '%s'", apiErr.Message)
	}
}
