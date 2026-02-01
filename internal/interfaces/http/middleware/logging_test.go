package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestLogging_success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	})

	middleware := Logging(logger)
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestLogging_custom_status(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	})

	middleware := Logging(logger)
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestWrappedWriter_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	wrapped := &wrappedWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	wrapped.WriteHeader(http.StatusCreated)

	if wrapped.statusCode != http.StatusCreated {
		t.Errorf("expected statusCode 201, got %d", wrapped.statusCode)
	}

	if w.Code != http.StatusCreated {
		t.Errorf("expected response status 201, got %d", w.Code)
	}
}
