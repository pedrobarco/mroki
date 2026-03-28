package traffictesting

import (
	"testing"
)

func TestNewGateSort(t *testing.T) {
	t.Run("create from value objects", func(t *testing.T) {
		field := SortByLiveURL()
		order := Asc()

		sort := NewGateSort(field, order)

		if !sort.Field().Equals(field) {
			t.Error("Field() should return the provided field")
		}
		if !sort.Order().Equals(order) {
			t.Error("Order() should return the provided order")
		}
	})

	t.Run("create with id desc", func(t *testing.T) {
		field := SortByGateID()
		order := Desc()

		sort := NewGateSort(field, order)

		if !sort.Field().IsID() {
			t.Error("Should have id field")
		}
		if !sort.Order().IsDesc() {
			t.Error("Should have desc order")
		}
	})

	t.Run("create with shadow_url asc", func(t *testing.T) {
		field := SortByShadowURL()
		order := Asc()

		sort := NewGateSort(field, order)

		if !sort.Field().IsShadowURL() {
			t.Error("Should have shadow_url field")
		}
		if !sort.Order().IsAsc() {
			t.Error("Should have asc order")
		}
	})
}

func TestDefaultGateSort(t *testing.T) {
	t.Run("default is id asc", func(t *testing.T) {
		sort := DefaultGateSort()

		if !sort.Field().IsID() {
			t.Error("Default should sort by id")
		}
		if !sort.Order().IsAsc() {
			t.Error("Default should be ascending order")
		}
	})
}

func TestGateSortString(t *testing.T) {
	tests := []struct {
		name    string
		field   GateSortField
		order   SortOrder
		wantStr string
	}{
		{
			name:    "id asc",
			field:   SortByGateID(),
			order:   Asc(),
			wantStr: "sort by id asc",
		},
		{
			name:    "live_url desc",
			field:   SortByLiveURL(),
			order:   Desc(),
			wantStr: "sort by live_url desc",
		},
		{
			name:    "shadow_url asc",
			field:   SortByShadowURL(),
			order:   Asc(),
			wantStr: "sort by shadow_url asc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sort := NewGateSort(tt.field, tt.order)
			if got := sort.String(); got != tt.wantStr {
				t.Errorf("String() = %v, want %v", got, tt.wantStr)
			}
		})
	}
}

func TestGateSortEquals(t *testing.T) {
	sort1 := NewGateSort(SortByGateID(), Asc())
	sort2 := NewGateSort(SortByGateID(), Asc())
	sort3 := NewGateSort(SortByLiveURL(), Desc())
	sort4 := NewGateSort(SortByGateID(), Desc())

	t.Run("same field and order are equal", func(t *testing.T) {
		if !sort1.Equals(sort2) {
			t.Error("Two sorts with same field and order should be equal")
		}
	})

	t.Run("different field are not equal", func(t *testing.T) {
		if sort1.Equals(sort3) {
			t.Error("Sorts with different fields should not be equal")
		}
	})

	t.Run("different order are not equal", func(t *testing.T) {
		if sort1.Equals(sort4) {
			t.Error("Sorts with different orders should not be equal")
		}
	})
}

func TestGateSortImmutability(t *testing.T) {
	t.Run("getters return copies", func(t *testing.T) {
		originalField := SortByLiveURL()
		originalOrder := Asc()

		sort := NewGateSort(originalField, originalOrder)

		field := sort.Field()
		order := sort.Order()

		if !field.Equals(originalField) {
			t.Error("Field() should return equal field")
		}
		if !order.Equals(originalOrder) {
			t.Error("Order() should return equal order")
		}
	})
}
