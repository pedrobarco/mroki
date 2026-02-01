package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAppHandler_ServeHTTP_Success(t *testing.T) {
	handler := AppHandler(func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"message":"success"}`))
		return err
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	expected := `{"message":"success"}`
	if rec.Body.String() != expected {
		t.Errorf("expected body %q, got %q", expected, rec.Body.String())
	}
}

func TestAppHandler_ServeHTTP_APIError(t *testing.T) {
	testErr := errors.New("test error")
	apiErr := NewError(http.StatusBadRequest, "validation failed", testErr)

	handler := AppHandler(func(w http.ResponseWriter, r *http.Request) error {
		return apiErr
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	var result APIError
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status code 400, got %d", result.StatusCode)
	}

	if result.Code != "Bad Request" {
		t.Errorf("expected code 'Bad Request', got %q", result.Code)
	}

	if result.Message != "validation failed" {
		t.Errorf("expected message 'validation failed', got %q", result.Message)
	}
}

func TestAppHandler_ServeHTTP_UnknownError(t *testing.T) {
	testErr := errors.New("unexpected error")

	handler := AppHandler(func(w http.ResponseWriter, r *http.Request) error {
		return testErr
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	var result APIError
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status code 500, got %d", result.StatusCode)
	}

	if result.Code != "Internal Server Error" {
		t.Errorf("expected code 'Internal Server Error', got %q", result.Code)
	}

	if result.Message != unknownErrorMessage {
		t.Errorf("expected message %q, got %q", unknownErrorMessage, result.Message)
	}
}

func TestAppHandler_ServeHTTP_NilError(t *testing.T) {
	handler := AppHandler(func(w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusOK)
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}
