package traffictesting

import (
	"testing"
)

func TestNewGateSortField(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValue string
		wantErr   bool
	}{
		{
			name:      "valid id lowercase",
			input:     "id",
			wantValue: "id",
			wantErr:   false,
		},
		{
			name:      "valid live_url lowercase",
			input:     "live_url",
			wantValue: "live_url",
			wantErr:   false,
		},
		{
			name:      "valid shadow_url lowercase",
			input:     "shadow_url",
			wantValue: "shadow_url",
			wantErr:   false,
		},
		{
			name:      "valid id uppercase",
			input:     "ID",
			wantValue: "id",
			wantErr:   false,
		},
		{
			name:      "valid live_url uppercase",
			input:     "LIVE_URL",
			wantValue: "live_url",
			wantErr:   false,
		},
		{
			name:      "valid shadow_url mixed case",
			input:     "Shadow_URL",
			wantValue: "shadow_url",
			wantErr:   false,
		},
		{
			name:      "empty string defaults to id",
			input:     "",
			wantValue: "id",
			wantErr:   false,
		},
		{
			name:      "whitespace defaults to id",
			input:     "   ",
			wantValue: "id",
			wantErr:   false,
		},
		{
			name:    "invalid field",
			input:   "invalid",
			wantErr: true,
		},
		{
			name:    "invalid field created_at",
			input:   "created_at",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewGateSortField(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewGateSortField() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("NewGateSortField() unexpected error: %v", err)
				return
			}

			if got.String() != tt.wantValue {
				t.Errorf("NewGateSortField() = %v, want %v", got.String(), tt.wantValue)
			}
		})
	}
}

func TestGateSortFieldFactories(t *testing.T) {
	t.Run("SortByGateID factory", func(t *testing.T) {
		field := SortByGateID()
		if !field.IsID() {
			t.Error("SortByGateID() should create id field")
		}
		if field.String() != "id" {
			t.Errorf("SortByGateID() value = %v, want 'id'", field.String())
		}
	})

	t.Run("SortByLiveURL factory", func(t *testing.T) {
		field := SortByLiveURL()
		if !field.IsLiveURL() {
			t.Error("SortByLiveURL() should create live_url field")
		}
		if field.String() != "live_url" {
			t.Errorf("SortByLiveURL() value = %v, want 'live_url'", field.String())
		}
	})

	t.Run("SortByShadowURL factory", func(t *testing.T) {
		field := SortByShadowURL()
		if !field.IsShadowURL() {
			t.Error("SortByShadowURL() should create shadow_url field")
		}
		if field.String() != "shadow_url" {
			t.Errorf("SortByShadowURL() value = %v, want 'shadow_url'", field.String())
		}
	})
}


func TestGateSortFieldPredicates(t *testing.T) {
	id := SortByGateID()
	liveURL := SortByLiveURL()
	shadowURL := SortByShadowURL()

	t.Run("IsID predicate", func(t *testing.T) {
		if !id.IsID() {
			t.Error("IsID() should return true for id field")
		}
		if liveURL.IsID() {
			t.Error("IsID() should return false for live_url field")
		}
		if shadowURL.IsID() {
			t.Error("IsID() should return false for shadow_url field")
		}
	})

	t.Run("IsLiveURL predicate", func(t *testing.T) {
		if !liveURL.IsLiveURL() {
			t.Error("IsLiveURL() should return true for live_url field")
		}
		if id.IsLiveURL() {
			t.Error("IsLiveURL() should return false for id field")
		}
		if shadowURL.IsLiveURL() {
			t.Error("IsLiveURL() should return false for shadow_url field")
		}
	})

	t.Run("IsShadowURL predicate", func(t *testing.T) {
		if !shadowURL.IsShadowURL() {
			t.Error("IsShadowURL() should return true for shadow_url field")
		}
		if id.IsShadowURL() {
			t.Error("IsShadowURL() should return false for id field")
		}
		if liveURL.IsShadowURL() {
			t.Error("IsShadowURL() should return false for live_url field")
		}
	})
}

func TestGateSortFieldEquals(t *testing.T) {
	id1 := SortByGateID()
	id2 := SortByGateID()
	liveURL := SortByLiveURL()

	t.Run("same values are equal", func(t *testing.T) {
		if !id1.Equals(id2) {
			t.Error("Two id fields should be equal")
		}
	})

	t.Run("different values are not equal", func(t *testing.T) {
		if id1.Equals(liveURL) {
			t.Error("id and live_url fields should not be equal")
		}
	})
}