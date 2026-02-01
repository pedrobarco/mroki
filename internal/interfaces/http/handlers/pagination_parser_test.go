package handlers

import (
	"net/url"
	"testing"
)

func TestParsePaginationQueryParams_Defaults(t *testing.T) {
	query := url.Values{}

	limit, offset, err := parsePaginationQueryParams(query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if limit != 0 {
		t.Errorf("expected limit 0 (not provided), got %d", limit)
	}

	if offset != 0 {
		t.Errorf("expected offset 0 (not provided), got %d", offset)
	}
}

func TestParsePaginationQueryParams_CustomValues(t *testing.T) {
	query := url.Values{
		"limit":  {"10"},
		"offset": {"20"},
	}

	limit, offset, err := parsePaginationQueryParams(query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if limit != 10 {
		t.Errorf("expected limit 10, got %d", limit)
	}

	if offset != 20 {
		t.Errorf("expected offset 20, got %d", offset)
	}
}

func TestParsePaginationQueryParams_OnlyLimit(t *testing.T) {
	query := url.Values{
		"limit": {"100"},
	}

	limit, offset, err := parsePaginationQueryParams(query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if limit != 100 {
		t.Errorf("expected limit 100, got %d", limit)
	}

	if offset != 0 {
		t.Errorf("expected default offset 0, got %d", offset)
	}
}

func TestParsePaginationQueryParams_OnlyOffset(t *testing.T) {
	query := url.Values{
		"offset": {"50"},
	}

	limit, offset, err := parsePaginationQueryParams(query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if limit != 0 {
		t.Errorf("expected limit 0 (not provided), got %d", limit)
	}

	if offset != 50 {
		t.Errorf("expected offset 50, got %d", offset)
	}
}

func TestParsePaginationQueryParams_InvalidLimit(t *testing.T) {
	query := url.Values{
		"limit": {"invalid"},
	}

	_, _, err := parsePaginationQueryParams(query)
	if err == nil {
		t.Fatal("expected error for invalid limit, got nil")
	}

	expectedMsg := "invalid limit parameter"
	if err.Error()[:len(expectedMsg)] != expectedMsg {
		t.Errorf("expected error message to start with %q, got %q", expectedMsg, err.Error())
	}
}

func TestParsePaginationQueryParams_InvalidOffset(t *testing.T) {
	query := url.Values{
		"offset": {"not-a-number"},
	}

	_, _, err := parsePaginationQueryParams(query)
	if err == nil {
		t.Fatal("expected error for invalid offset, got nil")
	}

	expectedMsg := "invalid offset parameter"
	if err.Error()[:len(expectedMsg)] != expectedMsg {
		t.Errorf("expected error message to start with %q, got %q", expectedMsg, err.Error())
	}
}

func TestParsePaginationQueryParams_NegativeValues(t *testing.T) {
	// Test that negative values are parsed (validation happens in domain layer)
	query := url.Values{
		"limit":  {"-10"},
		"offset": {"-5"},
	}

	limit, offset, err := parsePaginationQueryParams(query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if limit != -10 {
		t.Errorf("expected limit -10, got %d", limit)
	}

	if offset != -5 {
		t.Errorf("expected offset -5, got %d", offset)
	}
}

func TestParsePaginationQueryParams_ZeroValues(t *testing.T) {
	query := url.Values{
		"limit":  {"0"},
		"offset": {"0"},
	}

	limit, offset, err := parsePaginationQueryParams(query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if limit != 0 {
		t.Errorf("expected limit 0, got %d", limit)
	}

	if offset != 0 {
		t.Errorf("expected offset 0, got %d", offset)
	}
}

func TestParsePaginationQueryParams_LargeValues(t *testing.T) {
	query := url.Values{
		"limit":  {"1000"},
		"offset": {"999999"},
	}

	limit, offset, err := parsePaginationQueryParams(query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if limit != 1000 {
		t.Errorf("expected limit 1000, got %d", limit)
	}

	if offset != 999999 {
		t.Errorf("expected offset 999999, got %d", offset)
	}
}
