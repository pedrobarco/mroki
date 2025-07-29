package diff

import (
	"fmt"

	"github.com/josephburnett/jd/v2"
)

// JSON is a DifferFunc that compares two JSON strings.
func JSON(a, b string) (string, error) {
	ja, err := jd.ReadJsonString(string(a))
	if err != nil {
		return "", fmt.Errorf("failed to read JSON from first input: %w", err)
	}

	jb, err := jd.ReadJsonString(string(b))
	if err != nil {
		return "", fmt.Errorf("failed to read JSON from second input: %w", err)
	}

	jdiff := ja.Diff(jb)
	return jdiff.Render(), nil
}
