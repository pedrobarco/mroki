package traffictesting_test

import (
	"net/http"
	"testing"

	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/stretchr/testify/assert"
)

func TestNewHeaders_nil_returns_empty(t *testing.T) {
	h := traffictesting.NewHeaders(nil)
	assert.NotNil(t, h.HTTPHeader())
	assert.Empty(t, h.HTTPHeader())
}

func TestNewHeaders_roundtrip(t *testing.T) {
	original := http.Header{"Content-Type": {"application/json"}}
	h := traffictesting.NewHeaders(original)
	assert.Equal(t, original, h.HTTPHeader())
}
