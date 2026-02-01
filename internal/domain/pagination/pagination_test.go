package pagination_test

import (
	"testing"

	"github.com/pedrobarco/mroki/internal/domain/pagination"
	"github.com/stretchr/testify/assert"
)

func TestNewParams(t *testing.T) {
	t.Run("valid params", func(t *testing.T) {
		params, err := pagination.NewParams(50, 100)
		assert.NoError(t, err)
		assert.NotNil(t, params)
		assert.Equal(t, 50, params.Limit())
		assert.Equal(t, 100, params.Offset())
	})

	t.Run("applies default limit when limit is 0", func(t *testing.T) {
		params, err := pagination.NewParams(0, 10)
		assert.NoError(t, err)
		assert.Equal(t, 50, params.Limit()) // defaultLimit = 50
		assert.Equal(t, 10, params.Offset())
	})

	t.Run("applies default limit when limit is negative", func(t *testing.T) {
		params, err := pagination.NewParams(-5, 10)
		assert.NoError(t, err)
		assert.Equal(t, 50, params.Limit()) // defaultLimit = 50
		assert.Equal(t, 10, params.Offset())
	})

	t.Run("enforces max limit", func(t *testing.T) {
		params, err := pagination.NewParams(200, 0)
		assert.NoError(t, err)
		assert.Equal(t, 100, params.Limit()) // maxLimit = 100
		assert.Equal(t, 0, params.Offset())
	})

	t.Run("handles negative offset", func(t *testing.T) {
		params, err := pagination.NewParams(50, -10)
		assert.NoError(t, err)
		assert.Equal(t, 50, params.Limit())
		assert.Equal(t, 0, params.Offset()) // defaults to 0
	})

	t.Run("uses defaults when both are invalid", func(t *testing.T) {
		params, err := pagination.NewParams(0, 0)
		assert.NoError(t, err)
		assert.Equal(t, 50, params.Limit())
		assert.Equal(t, 0, params.Offset())
	})
}

func TestParamsGetters(t *testing.T) {
	t.Run("getters return correct values", func(t *testing.T) {
		params, _ := pagination.NewParams(25, 50)
		assert.Equal(t, 25, params.Limit())
		assert.Equal(t, 50, params.Offset())
	})

	t.Run("params are immutable", func(t *testing.T) {
		params, _ := pagination.NewParams(25, 50)

		// Store original values
		originalLimit := params.Limit()
		originalOffset := params.Offset()

		// Call getters multiple times
		_ = params.Limit()
		_ = params.Offset()

		// Values should not change
		assert.Equal(t, originalLimit, params.Limit())
		assert.Equal(t, originalOffset, params.Offset())
	})
}

func TestParamsValidate(t *testing.T) {
	t.Run("valid params", func(t *testing.T) {
		params, _ := pagination.NewParams(50, 100)
		err := params.Validate()
		assert.NoError(t, err)
	})

	t.Run("minimum limit is valid", func(t *testing.T) {
		params, _ := pagination.NewParams(1, 0)
		err := params.Validate()
		assert.NoError(t, err) // minLimit is 1, so this is valid
	})

	t.Run("limit above maximum corrected by NewParams", func(t *testing.T) {
		params, _ := pagination.NewParams(150, 0)
		err := params.Validate()
		assert.NoError(t, err) // NewParams already enforced maxLimit
		assert.Equal(t, 100, params.Limit())
	})

	t.Run("negative offset corrected by NewParams", func(t *testing.T) {
		params, _ := pagination.NewParams(50, -10)
		err := params.Validate()
		assert.NoError(t, err) // NewParams already corrected offset
		assert.Equal(t, 0, params.Offset())
	})
}

func TestNewPagedResult(t *testing.T) {
	t.Run("has more pages", func(t *testing.T) {
		params, _ := pagination.NewParams(10, 0)
		items := []string{"a", "b", "c"}

		result := pagination.NewPagedResult(items, 100, params)

		assert.Equal(t, items, result.Items)
		assert.Equal(t, int64(100), result.Total)
		assert.Equal(t, 10, result.Limit)
		assert.Equal(t, 0, result.Offset)
		assert.True(t, result.HasMore) // 0+10 < 100
	})

	t.Run("no more pages - exactly at end", func(t *testing.T) {
		params, _ := pagination.NewParams(10, 90)
		items := []string{"a", "b"}

		result := pagination.NewPagedResult(items, 100, params)

		assert.Equal(t, int64(100), result.Total)
		assert.False(t, result.HasMore) // 90+10 = 100
	})

	t.Run("no more pages - past end", func(t *testing.T) {
		params, _ := pagination.NewParams(10, 95)
		items := []string{"a"}

		result := pagination.NewPagedResult(items, 100, params)

		assert.False(t, result.HasMore) // 95+10 > 100
	})

	t.Run("empty result set", func(t *testing.T) {
		params, _ := pagination.NewParams(10, 0)
		var items []string // nil slice

		result := pagination.NewPagedResult(items, 0, params)

		assert.NotNil(t, result.Items) // Should be empty slice, not nil
		assert.Len(t, result.Items, 0)
		assert.Equal(t, int64(0), result.Total)
		assert.False(t, result.HasMore)
	})

	t.Run("single page of results", func(t *testing.T) {
		params, _ := pagination.NewParams(50, 0)
		items := []string{"a", "b", "c"}

		result := pagination.NewPagedResult(items, 3, params)

		assert.Equal(t, int64(3), result.Total)
		assert.False(t, result.HasMore) // 0+50 >= 3
	})

	t.Run("preserves limit and offset from params", func(t *testing.T) {
		params, _ := pagination.NewParams(25, 50)
		items := []string{"a", "b"}

		result := pagination.NewPagedResult(items, 100, params)

		assert.Equal(t, 25, result.Limit)
		assert.Equal(t, 50, result.Offset)
	})
}

func TestPagedResultEmptyItems(t *testing.T) {
	t.Run("nil items converted to empty slice", func(t *testing.T) {
		params, _ := pagination.NewParams(10, 0)

		result := pagination.NewPagedResult[string](nil, 0, params)

		assert.NotNil(t, result.Items)
		assert.Equal(t, []string{}, result.Items)
		assert.Len(t, result.Items, 0)
	})

	t.Run("empty slice remains empty slice", func(t *testing.T) {
		params, _ := pagination.NewParams(10, 0)
		items := []string{}

		result := pagination.NewPagedResult(items, 0, params)

		assert.NotNil(t, result.Items)
		assert.Equal(t, []string{}, result.Items)
		assert.Len(t, result.Items, 0)
	})

	t.Run("typed nil slice converted to empty slice", func(t *testing.T) {
		params, _ := pagination.NewParams(10, 0)
		var items []int = nil

		result := pagination.NewPagedResult(items, 0, params)

		assert.NotNil(t, result.Items)
		assert.Equal(t, []int{}, result.Items)
		assert.Len(t, result.Items, 0)
	})
}
