package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pedrobarco/mroki/pkg/dto"
)

func TestGetConfig_Success(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/config", nil)
	rec := httptest.NewRecorder()

	appHandler := GetConfig(720 * time.Hour)
	if err := appHandler(rec, req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp dto.Response[dto.Config]
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Data.Retention != "720h0m0s" {
		t.Errorf("expected retention %q, got %q", "720h0m0s", resp.Data.Retention)
	}
}

func TestGetConfig_RendersConfiguredRetention(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/config", nil)
	rec := httptest.NewRecorder()

	appHandler := GetConfig(168 * time.Hour)
	if err := appHandler(rec, req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp dto.Response[dto.Config]
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Data.Retention != "168h0m0s" {
		t.Errorf("expected retention %q, got %q", "168h0m0s", resp.Data.Retention)
	}
}
