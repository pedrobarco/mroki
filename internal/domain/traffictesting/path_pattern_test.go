package traffictesting

import (
	"strings"
	"testing"
)

func TestNewPathPattern(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValue string
		wantErr   bool
	}{
		{
			name:      "valid simple path",
			input:     "/api/users",
			wantValue: "/api/users",
			wantErr:   false,
		},
		{
			name:      "valid path with wildcard",
			input:     "/api/users/*",
			wantValue: "/api/users/*",
			wantErr:   false,
		},
		{
			name:      "valid path with multiple wildcards",
			input:     "/api/*/resources/*",
			wantValue: "/api/*/resources/*",
			wantErr:   false,
		},
		{
			name:      "empty string is valid",
			input:     "",
			wantValue: "",
			wantErr:   false,
		},
		{
			name:      "whitespace trimmed",
			input:     "  /api/users  ",
			wantValue: "/api/users",
			wantErr:   false,
		},
		{
			name:      "whitespace only becomes empty",
			input:     "   ",
			wantValue: "",
			wantErr:   false,
		},
		{
			name:      "SQL injection semicolon rejected",
			input:     "/api/users; DROP TABLE",
			wantValue: "",
			wantErr:   true,
		},
		{
			name:      "SQL injection comment rejected",
			input:     "/api/users--comment",
			wantValue: "",
			wantErr:   true,
		},
		{
			name:      "SQL SELECT keyword rejected",
			input:     "/api/SELECT",
			wantValue: "",
			wantErr:   true,
		},
		{
			name:      "SQL DROP keyword rejected",
			input:     "/api/DROP",
			wantValue: "",
			wantErr:   true,
		},
		{
			name:      "SQL single quote rejected",
			input:     "/api/users'",
			wantValue: "",
			wantErr:   true,
		},
		{
			name:      "SQL double quote rejected",
			input:     "/api/users\"",
			wantValue: "",
			wantErr:   true,
		},
		{
			name:      "exceeds max length",
			input:     "/" + strings.Repeat("a", 500),
			wantValue: "",
			wantErr:   true,
		},
		{
			name:      "max length exactly",
			input:     strings.Repeat("a", 500),
			wantValue: strings.Repeat("a", 500),
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewPathPattern(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewPathPattern() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("NewPathPattern() unexpected error: %v", err)
				return
			}

			if got.String() != tt.wantValue {
				t.Errorf("NewPathPattern() = %v, want %v", got.String(), tt.wantValue)
			}
		})
	}
}

func TestEmptyPathPattern(t *testing.T) {
	t.Run("creates empty pattern", func(t *testing.T) {
		pattern := EmptyPathPattern()
		if !pattern.IsEmpty() {
			t.Error("EmptyPathPattern() should create empty pattern")
		}
		if pattern.String() != "" {
			t.Errorf("EmptyPathPattern() value = %v, want empty string", pattern.String())
		}
	})
}

func TestPathPatternIsEmpty(t *testing.T) {
	tests := []struct {
		name    string
		pattern PathPattern
		want    bool
	}{
		{
			name:    "empty pattern",
			pattern: EmptyPathPattern(),
			want:    true,
		},
		{
			name:    "non-empty pattern",
			pattern: PathPattern{value: "/api/users"},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pattern.IsEmpty(); got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPathPatternEquals(t *testing.T) {
	pattern1, _ := NewPathPattern("/api/users")
	pattern2, _ := NewPathPattern("/api/users")
	pattern3, _ := NewPathPattern("/api/posts")
	empty1 := EmptyPathPattern()
	empty2 := EmptyPathPattern()

	t.Run("same values are equal", func(t *testing.T) {
		if !pattern1.Equals(pattern2) {
			t.Error("Two patterns with same value should be equal")
		}
	})

	t.Run("different values are not equal", func(t *testing.T) {
		if pattern1.Equals(pattern3) {
			t.Error("Patterns with different values should not be equal")
		}
	})

	t.Run("empty patterns are equal", func(t *testing.T) {
		if !empty1.Equals(empty2) {
			t.Error("Two empty patterns should be equal")
		}
	})

	t.Run("empty and non-empty are not equal", func(t *testing.T) {
		if empty1.Equals(pattern1) {
			t.Error("Empty and non-empty patterns should not be equal")
		}
	})
}
