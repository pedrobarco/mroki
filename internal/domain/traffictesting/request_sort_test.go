package traffictesting

import (
	"testing"
)

func TestNewRequestSort(t *testing.T) {
	t.Run("create from value objects", func(t *testing.T) {
		field := SortByMethod()
		order := Asc()

		sort := NewRequestSort(field, order)

		if !sort.Field().Equals(field) {
			t.Error("Field() should return the provided field")
		}
		if !sort.Order().Equals(order) {
			t.Error("Order() should return the provided order")
		}
	})

	t.Run("create with created_at desc", func(t *testing.T) {
		field := SortByCreatedAt()
		order := Desc()

		sort := NewRequestSort(field, order)

		if !sort.Field().IsCreatedAt() {
			t.Error("Should have created_at field")
		}
		if !sort.Order().IsDesc() {
			t.Error("Should have desc order")
		}
	})

	t.Run("create with path asc", func(t *testing.T) {
		field := SortByPath()
		order := Asc()

		sort := NewRequestSort(field, order)

		if !sort.Field().IsPath() {
			t.Error("Should have path field")
		}
		if !sort.Order().IsAsc() {
			t.Error("Should have asc order")
		}
	})
}

func TestDefaultRequestSort(t *testing.T) {
	t.Run("default is created_at desc", func(t *testing.T) {
		sort := DefaultRequestSort()

		if !sort.Field().IsCreatedAt() {
			t.Error("Default should sort by created_at")
		}
		if !sort.Order().IsDesc() {
			t.Error("Default should be descending order")
		}
	})
}

func TestRequestSortString(t *testing.T) {
	tests := []struct {
		name    string
		field   RequestSortField
		order   SortOrder
		wantStr string
	}{
		{
			name:    "created_at desc",
			field:   SortByCreatedAt(),
			order:   Desc(),
			wantStr: "sort by created_at desc",
		},
		{
			name:    "method asc",
			field:   SortByMethod(),
			order:   Asc(),
			wantStr: "sort by method asc",
		},
		{
			name:    "path desc",
			field:   SortByPath(),
			order:   Desc(),
			wantStr: "sort by path desc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sort := NewRequestSort(tt.field, tt.order)
			if got := sort.String(); got != tt.wantStr {
				t.Errorf("String() = %v, want %v", got, tt.wantStr)
			}
		})
	}
}

func TestRequestSortEquals(t *testing.T) {
	sort1 := NewRequestSort(SortByCreatedAt(), Desc())
	sort2 := NewRequestSort(SortByCreatedAt(), Desc())
	sort3 := NewRequestSort(SortByMethod(), Asc())
	sort4 := NewRequestSort(SortByCreatedAt(), Asc())

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

func TestRequestSortImmutability(t *testing.T) {
	t.Run("getters return copies", func(t *testing.T) {
		originalField := SortByMethod()
		originalOrder := Asc()

		sort := NewRequestSort(originalField, originalOrder)

		// Get field and order
		field := sort.Field()
		order := sort.Order()

		// Verify they match originals
		if !field.Equals(originalField) {
			t.Error("Field() should return equal field")
		}
		if !order.Equals(originalOrder) {
			t.Error("Order() should return equal order")
		}
	})
}
