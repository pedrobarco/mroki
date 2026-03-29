package traffictesting

import (
	"testing"
)

func TestNewGateFilters(t *testing.T) {
	t.Run("create with all filters", func(t *testing.T) {
		filters := NewGateFilters("my-gate", "live.example.com", "shadow.example.com")

		if filters.Name() != "my-gate" {
			t.Errorf("Name() = %v, want 'my-gate'", filters.Name())
		}
		if filters.LiveURL() != "live.example.com" {
			t.Errorf("LiveURL() = %v, want 'live.example.com'", filters.LiveURL())
		}
		if filters.ShadowURL() != "shadow.example.com" {
			t.Errorf("ShadowURL() = %v, want 'shadow.example.com'", filters.ShadowURL())
		}
	})

	t.Run("values are trimmed", func(t *testing.T) {
		filters := NewGateFilters("  my-gate  ", "  live.example.com  ", "  shadow.example.com  ")

		if filters.Name() != "my-gate" {
			t.Errorf("Name() = %v, want 'my-gate'", filters.Name())
		}
		if filters.LiveURL() != "live.example.com" {
			t.Errorf("LiveURL() = %v, want 'live.example.com'", filters.LiveURL())
		}
		if filters.ShadowURL() != "shadow.example.com" {
			t.Errorf("ShadowURL() = %v, want 'shadow.example.com'", filters.ShadowURL())
		}
	})

	t.Run("empty strings produce empty filters", func(t *testing.T) {
		filters := NewGateFilters("", "", "")

		if !filters.IsEmpty() {
			t.Error("Filters with empty strings should be empty")
		}
	})

	t.Run("whitespace-only strings produce empty filters", func(t *testing.T) {
		filters := NewGateFilters("   ", "   ", "   ")

		if !filters.IsEmpty() {
			t.Error("Filters with whitespace-only strings should be empty")
		}
	})
}

func TestEmptyGateFilters(t *testing.T) {
	t.Run("creates empty filters", func(t *testing.T) {
		filters := EmptyGateFilters()

		if !filters.IsEmpty() {
			t.Error("EmptyGateFilters() should create empty filters")
		}
		if filters.LiveURL() != "" {
			t.Error("LiveURL() should be empty")
		}
		if filters.ShadowURL() != "" {
			t.Error("ShadowURL() should be empty")
		}
	})
}

func TestGateFiltersIsEmpty(t *testing.T) {
	tests := []struct {
		name    string
		filters GateFilters
		want    bool
	}{
		{
			name:    "empty filters",
			filters: EmptyGateFilters(),
			want:    true,
		},
		{
			name:    "with name only",
			filters: NewGateFilters("my-gate", "", ""),
			want:    false,
		},
		{
			name:    "with live_url only",
			filters: NewGateFilters("", "live.example.com", ""),
			want:    false,
		},
		{
			name:    "with shadow_url only",
			filters: NewGateFilters("", "", "shadow.example.com"),
			want:    false,
		},
		{
			name:    "with all filters",
			filters: NewGateFilters("my-gate", "live.example.com", "shadow.example.com"),
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.filters.IsEmpty(); got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGateFiltersHasMethods(t *testing.T) {
	t.Run("HasLiveURLFilter", func(t *testing.T) {
		empty := EmptyGateFilters()
		if empty.HasLiveURLFilter() {
			t.Error("Empty filters should not have live URL filter")
		}

		withLiveURL := NewGateFilters("", "live.example.com", "")
		if !withLiveURL.HasLiveURLFilter() {
			t.Error("Filters with live URL should have live URL filter")
		}
	})

	t.Run("HasShadowURLFilter", func(t *testing.T) {
		empty := EmptyGateFilters()
		if empty.HasShadowURLFilter() {
			t.Error("Empty filters should not have shadow URL filter")
		}

		withShadowURL := NewGateFilters("", "", "shadow.example.com")
		if !withShadowURL.HasShadowURLFilter() {
			t.Error("Filters with shadow URL should have shadow URL filter")
		}
	})
}

func TestGateFiltersString(t *testing.T) {
	tests := []struct {
		name    string
		filters GateFilters
		want    string
	}{
		{
			name:    "empty filters",
			filters: EmptyGateFilters(),
			want:    "no filters",
		},
		{
			name:    "with live_url",
			filters: NewGateFilters("", "live.example.com", ""),
			want:    "live_url='live.example.com'",
		},
		{
			name:    "with shadow_url",
			filters: NewGateFilters("", "", "shadow.example.com"),
			want:    "shadow_url='shadow.example.com'",
		},
		{
			name:    "with all",
			filters: NewGateFilters("my-gate", "live.example.com", "shadow.example.com"),
			want:    "name='my-gate', live_url='live.example.com', shadow_url='shadow.example.com'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.filters.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}
