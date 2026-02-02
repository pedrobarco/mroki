package traffictesting

import (
	"fmt"
	"strings"
)

// Path represents a valid HTTP request path
type Path string

// ParsePath validates and creates a Path
func ParsePath(path string) (Path, error) {
	if path == "" {
		return "", fmt.Errorf("%w: cannot be empty", ErrInvalidPath)
	}
	if !strings.HasPrefix(path, "/") {
		return "", fmt.Errorf("%w: must start with /", ErrInvalidPath)
	}
	if len(path) > 2048 {
		return "", fmt.Errorf("%w: exceeds 2048 characters", ErrInvalidPath)
	}
	return Path(path), nil
}

// String returns the path as a string
func (p Path) String() string {
	return string(p)
}
