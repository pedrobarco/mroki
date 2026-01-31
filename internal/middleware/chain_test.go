package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChain_Then(t *testing.T) {
	var order []string

	mw1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw1-before")
			next.ServeHTTP(w, r)
			order = append(order, "mw1-after")
		})
	}

	mw2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw2-before")
			next.ServeHTTP(w, r)
			order = append(order, "mw2-after")
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
		w.WriteHeader(http.StatusOK)
	})

	chain := Chain{mw1, mw2}
	wrapped := chain.Then(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	// Verify middleware execution order (backward chain means mw1 wraps mw2)
	expected := []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d calls, got %d", len(expected), len(order))
	}

	for i, exp := range expected {
		if order[i] != exp {
			t.Errorf("at position %d: expected %s, got %s", i, exp, order[i])
		}
	}
}

func TestChain_ThenFunc(t *testing.T) {
	called := false

	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			next.ServeHTTP(w, r)
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	chain := Chain{mw}
	wrapped := chain.ThenFunc(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if !called {
		t.Error("middleware was not called")
	}

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}
