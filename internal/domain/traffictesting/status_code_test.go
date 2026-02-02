package traffictesting

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseStatusCode_Valid(t *testing.T) {
	testCases := []struct {
		name string
		code int
	}{
		{"informational lower bound", 100},
		{"informational", 101},
		{"success", 200},
		{"success created", 201},
		{"redirection", 301},
		{"client error", 400},
		{"not found", 404},
		{"server error", 500},
		{"server error upper bound", 599},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			statusCode, err := ParseStatusCode(tc.code)

			assert.NoError(t, err)
			assert.Equal(t, tc.code, statusCode.Int())
			assert.Equal(t, tc.code, int(statusCode))
		})
	}
}

func TestParseStatusCode_Invalid(t *testing.T) {
	testCases := []struct {
		name string
		code int
	}{
		{"negative", -1},
		{"zero", 0},
		{"below range", 99},
		{"above range", 600},
		{"way above range", 999},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			statusCode, err := ParseStatusCode(tc.code)

			assert.Error(t, err)
			assert.True(t, errors.Is(err, ErrInvalidStatusCode))
			assert.Equal(t, StatusCode(0), statusCode)
			assert.Contains(t, err.Error(), "must be 100-599")
		})
	}
}

func TestStatusCode_String(t *testing.T) {
	testCases := []struct {
		code     int
		expected string
	}{
		{200, "200"},
		{404, "404"},
		{500, "500"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			statusCode, _ := ParseStatusCode(tc.code)
			assert.Equal(t, tc.expected, statusCode.String())
		})
	}
}
