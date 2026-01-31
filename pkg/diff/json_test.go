package diff_test

import (
	"testing"

	"github.com/pedrobarco/mroki/pkg/diff"
	"github.com/stretchr/testify/assert"
)

func TestJSON_identical_json_returns_empty_diff(t *testing.T) {
	a := `{"key": "value", "nested": {"foo": "bar"}}`
	b := `{"key": "value", "nested": {"foo": "bar"}}`

	result, err := diff.JSON(a, b)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestJSON_different_json_returns_diff(t *testing.T) {
	a := `{"key": "value1", "count": 1}`
	b := `{"key": "value2", "count": 2}`

	result, err := diff.JSON(a, b)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "value1")
	assert.Contains(t, result, "value2")
}

func TestJSON_invalid_first_input_returns_error(t *testing.T) {
	a := `{invalid json}`
	b := `{"key": "value"}`

	result, err := diff.JSON(a, b)

	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "failed to read JSON from first input")
}

func TestJSON_invalid_second_input_returns_error(t *testing.T) {
	a := `{"key": "value"}`
	b := `{invalid json}`

	result, err := diff.JSON(a, b)

	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "failed to read JSON from second input")
}
