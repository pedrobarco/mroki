package traffictesting

import "fmt"

// StatusCode represents a valid HTTP status code (100-599)
type StatusCode int

// ParseStatusCode validates and creates a StatusCode
func ParseStatusCode(code int) (StatusCode, error) {
	if code < 100 || code > 599 {
		return 0, fmt.Errorf("%w: must be 100-599, got %d", ErrInvalidStatusCode, code)
	}
	return StatusCode(code), nil
}

// Int returns the status code as an integer
func (s StatusCode) Int() int {
	return int(s)
}

// String returns the status code as a string
func (s StatusCode) String() string {
	return fmt.Sprintf("%d", s)
}
