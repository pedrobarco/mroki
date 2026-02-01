package traffictesting_test

import (
	"testing"

	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/stretchr/testify/assert"
)

func TestParseAgentID_valid_hybrid_format(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{"simple hostname", "web-server-a1b2c3d4"},
		{"hostname with numbers", "api-prod-1-550e8400"},
		{"short hostname", "app-f47ac10b"},
		{"long hostname", "my-very-long-hostname-that-is-almost-fifty-chars-12345678"},
		{"uppercase hex accepted", "server-A1B2C3D4"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			id, err := traffictesting.ParseAgentID(tc.input)

			assert.NoError(t, err)
			assert.Equal(t, tc.input, id.String())
		})
	}
}

func TestParseAgentID_invalid_formats(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"pure uuid", "550e8400-e29b-41d4-a716-446655440000"},
		{"no dash", "webservera1b2c3d4"},
		{"too short uuid segment", "server-a1b2c3d"},
		{"too long uuid segment", "server-a1b2c3d4e"},
		{"non-hex chars", "server-g1h2i3j4"},
		{"special chars in hostname", "web@server-a1b2c3d4"},
		{"hostname too long", "this-is-a-very-very-very-very-very-very-very-long-hostname-that-exceeds-fifty-characters-a1b2c3d4"},
		{"space in hostname", "web server-a1b2c3d4"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			id, err := traffictesting.ParseAgentID(tc.input)

			assert.Error(t, err)
			assert.ErrorIs(t, err, traffictesting.ErrInvalidAgentID)
			assert.True(t, id.IsZero())
		})
	}
}

func TestNewAgentID_creates_unique_ids(t *testing.T) {
	id1 := traffictesting.NewAgentID()
	id2 := traffictesting.NewAgentID()

	assert.NotEqual(t, id1.String(), id2.String(), "should create unique IDs")
	assert.False(t, id1.IsZero())
	assert.False(t, id2.IsZero())

	// Should match the hybrid format
	_, err1 := traffictesting.ParseAgentID(id1.String())
	_, err2 := traffictesting.ParseAgentID(id2.String())
	assert.NoError(t, err1)
	assert.NoError(t, err2)
}

func TestNewAgentIDWithHostname(t *testing.T) {
	testCases := []struct {
		name     string
		hostname string
	}{
		{"simple", "web-server"},
		{"with numbers", "api-v2-prod"},
		{"needs sanitization", "web@server!123"},
		{"very long", "this-is-a-very-very-very-very-very-very-very-long-hostname-that-exceeds-max"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			id := traffictesting.NewAgentIDWithHostname(tc.hostname)

			assert.False(t, id.IsZero())

			// Should be valid hybrid format
			parsed, err := traffictesting.ParseAgentID(id.String())
			assert.NoError(t, err)
			assert.Equal(t, id.String(), parsed.String())
		})
	}
}

func TestAgentID_IsZero(t *testing.T) {
	var zero traffictesting.AgentID
	assert.True(t, zero.IsZero())

	nonZero := traffictesting.NewAgentID()
	assert.False(t, nonZero.IsZero())
}
