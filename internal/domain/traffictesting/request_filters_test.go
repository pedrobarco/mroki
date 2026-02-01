package traffictesting

import (
	"testing"
	"time"
)

func TestNewRequestFilters(t *testing.T) {
	now := time.Now()
	past := now.Add(-24 * time.Hour)

	t.Run("create from value objects", func(t *testing.T) {
		methods := []HTTPMethod{GET(), POST()}
		pathPattern, _ := NewPathPattern("/api/users")
		dateRange, _ := NewDateRange(&past, &now)
		agentID := "agent-123"
		hasDiff := true

		filters := NewRequestFilters(methods, pathPattern, dateRange, agentID, &hasDiff)

		if len(filters.Methods()) != 2 {
			t.Errorf("Methods() length = %v, want 2", len(filters.Methods()))
		}
		if !filters.PathPattern().Equals(pathPattern) {
			t.Error("PathPattern() should return the provided pattern")
		}
		if !filters.DateRange().Equals(dateRange) {
			t.Error("DateRange() should return the provided range")
		}
		if filters.AgentID() != agentID {
			t.Errorf("AgentID() = %v, want %v", filters.AgentID(), agentID)
		}
		if filters.HasDiff() == nil || *filters.HasDiff() != hasDiff {
			t.Errorf("HasDiff() = %v, want %v", filters.HasDiff(), &hasDiff)
		}
	})

	t.Run("nil methods becomes empty slice", func(t *testing.T) {
		filters := NewRequestFilters(nil, EmptyPathPattern(), EmptyDateRange(), "", nil)

		methods := filters.Methods()
		if methods == nil {
			t.Error("Methods() should return empty slice, not nil")
		}
		if len(methods) != 0 {
			t.Error("Methods() should be empty")
		}
	})

	t.Run("agent ID is trimmed", func(t *testing.T) {
		filters := NewRequestFilters([]HTTPMethod{}, EmptyPathPattern(), EmptyDateRange(), "  agent-123  ", nil)

		if filters.AgentID() != "agent-123" {
			t.Errorf("AgentID() = %v, want 'agent-123'", filters.AgentID())
		}
	})
}

func TestEmptyRequestFilters(t *testing.T) {
	t.Run("creates empty filters", func(t *testing.T) {
		filters := EmptyRequestFilters()

		if !filters.IsEmpty() {
			t.Error("EmptyRequestFilters() should create empty filters")
		}
		if len(filters.Methods()) != 0 {
			t.Error("Methods() should be empty")
		}
		if !filters.PathPattern().IsEmpty() {
			t.Error("PathPattern() should be empty")
		}
		if !filters.DateRange().IsEmpty() {
			t.Error("DateRange() should be empty")
		}
		if filters.AgentID() != "" {
			t.Error("AgentID() should be empty")
		}
		if filters.HasDiff() != nil {
			t.Error("HasDiff() should be nil")
		}
	})
}

func TestRequestFiltersIsEmpty(t *testing.T) {
	tests := []struct {
		name    string
		filters RequestFilters
		want    bool
	}{
		{
			name:    "empty filters",
			filters: EmptyRequestFilters(),
			want:    true,
		},
		{
			name: "with methods",
			filters: NewRequestFilters(
				[]HTTPMethod{GET()},
				EmptyPathPattern(),
				EmptyDateRange(),
				"",
				nil,
			),
			want: false,
		},
		{
			name: "with path pattern",
			filters: NewRequestFilters(
				[]HTTPMethod{},
				PathPattern{value: "/api/users"},
				EmptyDateRange(),
				"",
				nil,
			),
			want: false,
		},
		{
			name: "with agent ID",
			filters: NewRequestFilters(
				[]HTTPMethod{},
				EmptyPathPattern(),
				EmptyDateRange(),
				"agent-123",
				nil,
			),
			want: false,
		},
		{
			name: "with has diff",
			filters: func() RequestFilters {
				hasDiff := true
				return NewRequestFilters(
					[]HTTPMethod{},
					EmptyPathPattern(),
					EmptyDateRange(),
					"",
					&hasDiff,
				)
			}(),
			want: false,
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

func TestRequestFiltersHasMethods(t *testing.T) {
	t.Run("HasMethodFilter", func(t *testing.T) {
		empty := EmptyRequestFilters()
		if empty.HasMethodFilter() {
			t.Error("Empty filters should not have method filter")
		}

		withMethods := NewRequestFilters(
			[]HTTPMethod{GET()},
			EmptyPathPattern(),
			EmptyDateRange(),
			"",
			nil,
		)
		if !withMethods.HasMethodFilter() {
			t.Error("Filters with methods should have method filter")
		}
	})

	t.Run("HasPathFilter", func(t *testing.T) {
		empty := EmptyRequestFilters()
		if empty.HasPathFilter() {
			t.Error("Empty filters should not have path filter")
		}

		withPath := NewRequestFilters(
			[]HTTPMethod{},
			PathPattern{value: "/api/users"},
			EmptyDateRange(),
			"",
			nil,
		)
		if !withPath.HasPathFilter() {
			t.Error("Filters with path should have path filter")
		}
	})

	t.Run("HasDateRangeFilter", func(t *testing.T) {
		empty := EmptyRequestFilters()
		if empty.HasDateRangeFilter() {
			t.Error("Empty filters should not have date range filter")
		}

		now := time.Now()
		withDateRange := NewRequestFilters(
			[]HTTPMethod{},
			EmptyPathPattern(),
			DateRange{from: &now},
			"",
			nil,
		)
		if !withDateRange.HasDateRangeFilter() {
			t.Error("Filters with date range should have date range filter")
		}
	})

	t.Run("HasAgentFilter", func(t *testing.T) {
		empty := EmptyRequestFilters()
		if empty.HasAgentFilter() {
			t.Error("Empty filters should not have agent filter")
		}

		withAgent := NewRequestFilters(
			[]HTTPMethod{},
			EmptyPathPattern(),
			EmptyDateRange(),
			"agent-123",
			nil,
		)
		if !withAgent.HasAgentFilter() {
			t.Error("Filters with agent ID should have agent filter")
		}
	})

	t.Run("HasDiffFilter", func(t *testing.T) {
		empty := EmptyRequestFilters()
		if empty.HasDiffFilter() {
			t.Error("Empty filters should not have diff filter")
		}

		hasDiff := true
		withDiff := NewRequestFilters(
			[]HTTPMethod{},
			EmptyPathPattern(),
			EmptyDateRange(),
			"",
			&hasDiff,
		)
		if !withDiff.HasDiffFilter() {
			t.Error("Filters with hasDiff should have diff filter")
		}
	})
}

func TestRequestFiltersString(t *testing.T) {
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	past := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		filters RequestFilters
		want    string
	}{
		{
			name:    "empty filters",
			filters: EmptyRequestFilters(),
			want:    "no filters",
		},
		{
			name: "with methods",
			filters: NewRequestFilters(
				[]HTTPMethod{GET(), POST()},
				EmptyPathPattern(),
				EmptyDateRange(),
				"",
				nil,
			),
			want: "methods=[GET,POST]",
		},
		{
			name: "with path pattern",
			filters: NewRequestFilters(
				[]HTTPMethod{},
				PathPattern{value: "/api/users"},
				EmptyDateRange(),
				"",
				nil,
			),
			want: "path='/api/users'",
		},
		{
			name: "with date range",
			filters: NewRequestFilters(
				[]HTTPMethod{},
				EmptyPathPattern(),
				DateRange{from: &past, to: &now},
				"",
				nil,
			),
			want: "2026-01-01T12:00:00Z to 2026-01-15T12:00:00Z",
		},
		{
			name: "with agent ID",
			filters: NewRequestFilters(
				[]HTTPMethod{},
				EmptyPathPattern(),
				EmptyDateRange(),
				"agent-123",
				nil,
			),
			want: "agent='agent-123'",
		},
		{
			name: "with has diff",
			filters: func() RequestFilters {
				hasDiff := true
				return NewRequestFilters(
					[]HTTPMethod{},
					EmptyPathPattern(),
					EmptyDateRange(),
					"",
					&hasDiff,
				)
			}(),
			want: "hasDiff=true",
		},
		{
			name: "with all filters",
			filters: func() RequestFilters {
				hasDiff := false
				return NewRequestFilters(
					[]HTTPMethod{GET(), POST()},
					PathPattern{value: "/api/*"},
					DateRange{from: &past, to: &now},
					"agent-123",
					&hasDiff,
				)
			}(),
			want: "methods=[GET,POST], path='/api/*', 2026-01-01T12:00:00Z to 2026-01-15T12:00:00Z, agent='agent-123', hasDiff=false",
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

func TestRequestFiltersImmutability(t *testing.T) {
	t.Run("Methods returns copy", func(t *testing.T) {
		original := []HTTPMethod{GET(), POST()}
		filters := NewRequestFilters(original, EmptyPathPattern(), EmptyDateRange(), "", nil)

		// Get methods and modify
		methods := filters.Methods()
		methods[0] = PUT()

		// Original should be unchanged
		if filters.Methods()[0].Equals(PUT()) {
			t.Error("Modifying returned methods should not affect internal state")
		}
	})
}
