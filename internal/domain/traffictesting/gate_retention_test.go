package traffictesting_test

import (
	"testing"
	"time"

	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRetention_valid(t *testing.T) {
	r, err := traffictesting.ParseRetention("168h")

	require.NoError(t, err)
	assert.True(t, r.IsSet())
	assert.Equal(t, 168*time.Hour, r.Duration())
	assert.Equal(t, "168h0m0s", r.String())
}

func TestParseRetention_invalid_format(t *testing.T) {
	r, err := traffictesting.ParseRetention("not-a-duration")

	require.Error(t, err)
	assert.ErrorIs(t, err, traffictesting.ErrInvalidRetention)
	assert.False(t, r.IsSet())
}

func TestParseRetention_zero_is_rejected(t *testing.T) {
	_, err := traffictesting.ParseRetention("0s")

	require.Error(t, err)
	assert.ErrorIs(t, err, traffictesting.ErrInvalidRetention)
}

func TestParseRetention_negative_is_rejected(t *testing.T) {
	_, err := traffictesting.ParseRetention("-1h")

	require.Error(t, err)
	assert.ErrorIs(t, err, traffictesting.ErrInvalidRetention)
}

func TestNoRetention(t *testing.T) {
	r := traffictesting.NoRetention()

	assert.False(t, r.IsSet())
	assert.Equal(t, time.Duration(0), r.Duration())
	assert.Equal(t, "", r.String())
}

func TestRetention_Effective(t *testing.T) {
	global := 720 * time.Hour

	// Unset retention falls back to the global floor.
	assert.Equal(t, global, traffictesting.NoRetention().Effective(global))

	// A custom retention above the floor is used as-is.
	custom, err := traffictesting.ParseRetention("1000h")
	require.NoError(t, err)
	assert.Equal(t, 1000*time.Hour, custom.Effective(global))

	// A custom retention below the floor is clamped up to the global floor,
	// so a gate is never pruned below the global retention.
	below, err := traffictesting.ParseRetention("168h")
	require.NoError(t, err)
	assert.Equal(t, global, below.Effective(global))

	// A custom retention equal to the floor resolves to the floor.
	equal, err := traffictesting.ParseRetention("720h")
	require.NoError(t, err)
	assert.Equal(t, global, equal.Effective(global))
}
