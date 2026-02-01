package traffictesting

import (
	"testing"
)

func TestNewSortOrder(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValue string
		wantErr   bool
	}{
		{
			name:      "valid asc lowercase",
			input:     "asc",
			wantValue: "asc",
			wantErr:   false,
		},
		{
			name:      "valid desc lowercase",
			input:     "desc",
			wantValue: "desc",
			wantErr:   false,
		},
		{
			name:      "valid asc uppercase",
			input:     "ASC",
			wantValue: "asc",
			wantErr:   false,
		},
		{
			name:      "valid desc uppercase",
			input:     "DESC",
			wantValue: "desc",
			wantErr:   false,
		},
		{
			name:      "valid asc mixed case",
			input:     "AsC",
			wantValue: "asc",
			wantErr:   false,
		},
		{
			name:      "empty string defaults to desc",
			input:     "",
			wantValue: "desc",
			wantErr:   false,
		},
		{
			name:      "whitespace defaults to desc",
			input:     "   ",
			wantValue: "desc",
			wantErr:   false,
		},
		{
			name:      "invalid value ascending",
			input:     "ascending",
			wantValue: "",
			wantErr:   true,
		},
		{
			name:      "invalid value descending",
			input:     "descending",
			wantValue: "",
			wantErr:   true,
		},
		{
			name:      "invalid random value",
			input:     "invalid",
			wantValue: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewSortOrder(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewSortOrder() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("NewSortOrder() unexpected error: %v", err)
				return
			}

			if got.String() != tt.wantValue {
				t.Errorf("NewSortOrder() = %v, want %v", got.String(), tt.wantValue)
			}
		})
	}
}

func TestSortOrderFactories(t *testing.T) {
	t.Run("Asc factory", func(t *testing.T) {
		order := Asc()
		if !order.IsAsc() {
			t.Error("Asc() should create ascending order")
		}
		if order.String() != "asc" {
			t.Errorf("Asc() value = %v, want 'asc'", order.String())
		}
	})

	t.Run("Desc factory", func(t *testing.T) {
		order := Desc()
		if !order.IsDesc() {
			t.Error("Desc() should create descending order")
		}
		if order.String() != "desc" {
			t.Errorf("Desc() value = %v, want 'desc'", order.String())
		}
	})
}

func TestSortOrderPredicates(t *testing.T) {
	asc := Asc()
	desc := Desc()

	t.Run("IsAsc predicate", func(t *testing.T) {
		if !asc.IsAsc() {
			t.Error("IsAsc() should return true for ascending order")
		}
		if desc.IsAsc() {
			t.Error("IsAsc() should return false for descending order")
		}
	})

	t.Run("IsDesc predicate", func(t *testing.T) {
		if !desc.IsDesc() {
			t.Error("IsDesc() should return true for descending order")
		}
		if asc.IsDesc() {
			t.Error("IsDesc() should return false for ascending order")
		}
	})
}

func TestSortOrderEquals(t *testing.T) {
	asc1 := Asc()
	asc2 := Asc()
	desc1 := Desc()

	t.Run("same values are equal", func(t *testing.T) {
		if !asc1.Equals(asc2) {
			t.Error("Two ascending orders should be equal")
		}
	})

	t.Run("different values are not equal", func(t *testing.T) {
		if asc1.Equals(desc1) {
			t.Error("Ascending and descending orders should not be equal")
		}
	})
}
