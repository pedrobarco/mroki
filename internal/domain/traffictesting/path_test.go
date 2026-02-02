package traffictesting

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsePath_Valid(t *testing.T) {
	testCases := []struct {
		name string
		path string
	}{
		{"root", "/"},
		{"simple path", "/users"},
		{"nested path", "/api/v1/users"},
		{"path with id", "/users/123"},
		{"path with query string", "/search?q=test"},
		{"path with fragment", "/page#section"},
		{"path with special chars", "/users/%20/test"},
		{"max length", "/" + strings.Repeat("a", 2047)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path, err := ParsePath(tc.path)

			assert.NoError(t, err)
			assert.Equal(t, tc.path, path.String())
			assert.Equal(t, tc.path, string(path))
		})
	}
}

func TestParsePath_Invalid(t *testing.T) {
	testCases := []struct {
		name          string
		path          string
		errorContains string
	}{
		{"empty", "", "cannot be empty"},
		{"no leading slash", "users", "must start with /"},
		{"relative path", "api/users", "must start with /"},
		{"too long", "/" + strings.Repeat("a", 2048), "exceeds 2048 characters"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path, err := ParsePath(tc.path)

			assert.Error(t, err)
			assert.True(t, errors.Is(err, ErrInvalidPath))
			assert.Equal(t, Path(""), path)
			assert.Contains(t, err.Error(), tc.errorContains)
		})
	}
}

func TestPath_String(t *testing.T) {
	path, _ := ParsePath("/users/123")
	assert.Equal(t, "/users/123", path.String())
}
