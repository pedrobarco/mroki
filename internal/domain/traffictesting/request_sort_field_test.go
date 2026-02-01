package traffictesting

import (
	"testing"
)

func TestNewRequestSortField(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValue string
		wantErr   bool
	}{
		{
			name:      "valid created_at lowercase",
			input:     "created_at",
			wantValue: "created_at",
			wantErr:   false,
		},
		{
			name:      "valid method lowercase",
			input:     "method",
			wantValue: "method",
			wantErr:   false,
		},
		{
			name:      "valid path lowercase",
			input:     "path",
			wantValue: "path",
			wantErr:   false,
		},
		{
			name:      "valid created_at uppercase",
			input:     "CREATED_AT",
			wantValue: "created_at",
			wantErr:   false,
		},
		{
			name:      "valid method uppercase",
			input:     "METHOD",
			wantValue: "method",
			wantErr:   false,
		},
		{
			name:      "valid path mixed case",
			input:     "Path",
			wantValue: "path",
			wantErr:   false,
		},
		{
			name:      "empty string defaults to created_at",
			input:     "",
			wantValue: "created_at",
			wantErr:   false,
		},
		{
			name:      "whitespace defaults to created_at",
			input:     "   ",
			wantValue: "created_at",
			wantErr:   false,
		},
		{
			name:      "invalid field",
			input:     "invalid",
			wantValue: "",
			wantErr:   true,
		},
		{
			name:      "invalid field status",
			input:     "status",
			wantValue: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewRequestSortField(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewRequestSortField() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("NewRequestSortField() unexpected error: %v", err)
				return
			}

			if got.String() != tt.wantValue {
				t.Errorf("NewRequestSortField() = %v, want %v", got.String(), tt.wantValue)
			}
		})
	}
}

func TestRequestSortFieldFactories(t *testing.T) {
	t.Run("SortByCreatedAt factory", func(t *testing.T) {
		field := SortByCreatedAt()
		if !field.IsCreatedAt() {
			t.Error("SortByCreatedAt() should create created_at field")
		}
		if field.String() != "created_at" {
			t.Errorf("SortByCreatedAt() value = %v, want 'created_at'", field.String())
		}
	})

	t.Run("SortByMethod factory", func(t *testing.T) {
		field := SortByMethod()
		if !field.IsMethod() {
			t.Error("SortByMethod() should create method field")
		}
		if field.String() != "method" {
			t.Errorf("SortByMethod() value = %v, want 'method'", field.String())
		}
	})

	t.Run("SortByPath factory", func(t *testing.T) {
		field := SortByPath()
		if !field.IsPath() {
			t.Error("SortByPath() should create path field")
		}
		if field.String() != "path" {
			t.Errorf("SortByPath() value = %v, want 'path'", field.String())
		}
	})
}

func TestRequestSortFieldPredicates(t *testing.T) {
	createdAt := SortByCreatedAt()
	method := SortByMethod()
	path := SortByPath()

	t.Run("IsCreatedAt predicate", func(t *testing.T) {
		if !createdAt.IsCreatedAt() {
			t.Error("IsCreatedAt() should return true for created_at field")
		}
		if method.IsCreatedAt() {
			t.Error("IsCreatedAt() should return false for method field")
		}
		if path.IsCreatedAt() {
			t.Error("IsCreatedAt() should return false for path field")
		}
	})

	t.Run("IsMethod predicate", func(t *testing.T) {
		if !method.IsMethod() {
			t.Error("IsMethod() should return true for method field")
		}
		if createdAt.IsMethod() {
			t.Error("IsMethod() should return false for created_at field")
		}
		if path.IsMethod() {
			t.Error("IsMethod() should return false for path field")
		}
	})

	t.Run("IsPath predicate", func(t *testing.T) {
		if !path.IsPath() {
			t.Error("IsPath() should return true for path field")
		}
		if createdAt.IsPath() {
			t.Error("IsPath() should return false for created_at field")
		}
		if method.IsPath() {
			t.Error("IsPath() should return false for method field")
		}
	})
}

func TestRequestSortFieldEquals(t *testing.T) {
	createdAt1 := SortByCreatedAt()
	createdAt2 := SortByCreatedAt()
	method := SortByMethod()

	t.Run("same values are equal", func(t *testing.T) {
		if !createdAt1.Equals(createdAt2) {
			t.Error("Two created_at fields should be equal")
		}
	})

	t.Run("different values are not equal", func(t *testing.T) {
		if createdAt1.Equals(method) {
			t.Error("created_at and method fields should not be equal")
		}
	})
}
