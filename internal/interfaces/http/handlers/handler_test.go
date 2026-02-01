package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pedrobarco/mroki/pkg/dto"
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
	apiErr := dto.NewError(
		http.StatusBadRequest,
		dto.ErrorTypeInvalidRequestBody,
		"Invalid Request Body",
		"validation failed",
		testErr,
	)

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

	result := decodeErrorResponse(t, rec)

	if result.Status != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, result.Status)
	}
	if result.Type != dto.ErrorTypeInvalidRequestBody {
		t.Errorf("expected type %q, got %q", dto.ErrorTypeInvalidRequestBody, result.Type)
	}
	if result.Title != "Invalid Request Body" {
		t.Errorf("expected title %q, got %q", "Invalid Request Body", result.Title)
	}
	if !strings.Contains(result.Detail, "validation failed") {
		t.Errorf("expected detail to contain %q, got %q", "validation failed", result.Detail)
	}
	if result.Instance != "/test" {
		t.Errorf("expected instance %q, got %q", "/test", result.Instance)
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

	result := decodeErrorResponse(t, rec)

	if result.Status != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, result.Status)
	}
	if result.Type != dto.ErrorTypeInternalError {
		t.Errorf("expected type %q, got %q", dto.ErrorTypeInternalError, result.Type)
	}
	if result.Title != "Internal Server Error" {
		t.Errorf("expected title %q, got %q", "Internal Server Error", result.Title)
	}
	expectedMessage := "An unknown error occurred. Please try again later."
	if !strings.Contains(result.Detail, expectedMessage) {
		t.Errorf("expected detail to contain %q, got %q", expectedMessage, result.Detail)
	}
	// 5xx errors should NOT have instance field populated
	if result.Instance != "" {
		t.Errorf("expected empty instance for 5xx error, got %q", result.Instance)
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

// Test helper functions for RFC 7807 error assertions

// decodeErrorResponse decodes an RFC 7807 error response from a response body.
func decodeErrorResponse(t *testing.T, body *httptest.ResponseRecorder) dto.APIError {
	t.Helper()
	var apiErr dto.APIError
	if err := json.NewDecoder(body.Body).Decode(&apiErr); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	return apiErr
}
