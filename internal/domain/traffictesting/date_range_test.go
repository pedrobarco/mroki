package traffictesting

import (
	"testing"
	"time"
)

func TestNewDateRange(t *testing.T) {
	now := time.Now()
	past := now.Add(-24 * time.Hour)
	future := now.Add(24 * time.Hour)

	tests := []struct {
		name    string
		from    *time.Time
		to      *time.Time
		wantErr bool
	}{
		{
			name:    "both nil is valid",
			from:    nil,
			to:      nil,
			wantErr: false,
		},
		{
			name:    "only from is valid",
			from:    &past,
			to:      nil,
			wantErr: false,
		},
		{
			name:    "only to is valid",
			from:    nil,
			to:      &future,
			wantErr: false,
		},
		{
			name:    "valid range past to now",
			from:    &past,
			to:      &now,
			wantErr: false,
		},
		{
			name:    "valid range now to future",
			from:    &now,
			to:      &future,
			wantErr: false,
		},
		{
			name:    "valid range same time",
			from:    &now,
			to:      &now,
			wantErr: false,
		},
		{
			name:    "invalid range from after to",
			from:    &future,
			to:      &past,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewDateRange(tt.from, tt.to)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewDateRange() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("NewDateRange() unexpected error: %v", err)
				return
			}

			// Verify from
			if tt.from == nil && got.From() != nil {
				t.Error("From() should be nil")
			}
			if tt.from != nil && (got.From() == nil || !got.From().Equal(*tt.from)) {
				t.Errorf("From() = %v, want %v", got.From(), tt.from)
			}

			// Verify to
			if tt.to == nil && got.To() != nil {
				t.Error("To() should be nil")
			}
			if tt.to != nil && (got.To() == nil || !got.To().Equal(*tt.to)) {
				t.Errorf("To() = %v, want %v", got.To(), tt.to)
			}
		})
	}
}

func TestEmptyDateRange(t *testing.T) {
	t.Run("creates empty range", func(t *testing.T) {
		dr := EmptyDateRange()
		if !dr.IsEmpty() {
			t.Error("EmptyDateRange() should create empty range")
		}
		if dr.From() != nil {
			t.Error("From() should be nil")
		}
		if dr.To() != nil {
			t.Error("To() should be nil")
		}
	})
}

func TestDateRangeIsEmpty(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name string
		dr   DateRange
		want bool
	}{
		{
			name: "empty range",
			dr:   EmptyDateRange(),
			want: true,
		},
		{
			name: "only from set",
			dr:   DateRange{from: &now},
			want: false,
		},
		{
			name: "only to set",
			dr:   DateRange{to: &now},
			want: false,
		},
		{
			name: "both set",
			dr:   DateRange{from: &now, to: &now},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.dr.IsEmpty(); got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDateRangeHasFromHasTo(t *testing.T) {
	now := time.Now()

	t.Run("HasFrom", func(t *testing.T) {
		emptyRange := EmptyDateRange()
		if emptyRange.HasFrom() {
			t.Error("Empty range should not have From")
		}

		withFrom := DateRange{from: &now}
		if !withFrom.HasFrom() {
			t.Error("Range with From should have From")
		}
	})

	t.Run("HasTo", func(t *testing.T) {
		emptyRange := EmptyDateRange()
		if emptyRange.HasTo() {
			t.Error("Empty range should not have To")
		}

		withTo := DateRange{to: &now}
		if !withTo.HasTo() {
			t.Error("Range with To should have To")
		}
	})
}

func TestDateRangeString(t *testing.T) {
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	past := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		dr   DateRange
		want string
	}{
		{
			name: "empty range",
			dr:   EmptyDateRange(),
			want: "no date range",
		},
		{
			name: "only from",
			dr:   DateRange{from: &past},
			want: "from 2026-01-01T12:00:00Z",
		},
		{
			name: "only to",
			dr:   DateRange{to: &now},
			want: "until 2026-01-15T12:00:00Z",
		},
		{
			name: "full range",
			dr:   DateRange{from: &past, to: &now},
			want: "2026-01-01T12:00:00Z to 2026-01-15T12:00:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.dr.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDateRangeEquals(t *testing.T) {
	now := time.Now()
	past := now.Add(-24 * time.Hour)
	future := now.Add(24 * time.Hour)

	dr1 := DateRange{from: &past, to: &now}
	dr2 := DateRange{from: &past, to: &now}
	dr3 := DateRange{from: &now, to: &future}
	empty1 := EmptyDateRange()
	empty2 := EmptyDateRange()

	t.Run("same ranges are equal", func(t *testing.T) {
		if !dr1.Equals(dr2) {
			t.Error("Two ranges with same values should be equal")
		}
	})

	t.Run("different ranges are not equal", func(t *testing.T) {
		if dr1.Equals(dr3) {
			t.Error("Ranges with different values should not be equal")
		}
	})

	t.Run("empty ranges are equal", func(t *testing.T) {
		if !empty1.Equals(empty2) {
			t.Error("Two empty ranges should be equal")
		}
	})

	t.Run("empty and non-empty are not equal", func(t *testing.T) {
		if empty1.Equals(dr1) {
			t.Error("Empty and non-empty ranges should not be equal")
		}
	})
}
