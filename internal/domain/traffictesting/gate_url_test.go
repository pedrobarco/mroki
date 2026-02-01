package traffictesting_test

import (
	"net/url"
	"testing"

	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/stretchr/testify/assert"
)

func TestParseGateURL_valid_http_url(t *testing.T) {
	input := "http://example.com:8080/api/v1?query=value"

	gateURL, err := traffictesting.ParseGateURL(input)

	assert.NoError(t, err)
	assert.Equal(t, input, gateURL.String())
	assert.NotNil(t, gateURL.URL())
}

func TestParseGateURL_valid_https_url(t *testing.T) {
	input := "https://example.com"

	gateURL, err := traffictesting.ParseGateURL(input)

	assert.NoError(t, err)
	assert.Equal(t, input, gateURL.String())
}

func TestParseGateURL_invalid_scheme(t *testing.T) {
	input := "ftp://example.com"

	gateURL, err := traffictesting.ParseGateURL(input)

	assert.ErrorIs(t, err, traffictesting.ErrInvalidGateURL)
	assert.Empty(t, gateURL.String())
	assert.Contains(t, err.Error(), "scheme must be http or https")
}

func TestParseGateURL_malformed_url(t *testing.T) {
	input := "http://[invalid"

	gateURL, err := traffictesting.ParseGateURL(input)

	assert.ErrorIs(t, err, traffictesting.ErrInvalidGateURL)
	assert.Empty(t, gateURL.String())
}

func TestGateURLFromURL_preserves_value(t *testing.T) {
	original, _ := url.Parse("http://example.com")

	gateURL := traffictesting.GateURLFromURL(original)

	assert.Equal(t, original.String(), gateURL.String())
	assert.Equal(t, original, gateURL.URL())
}

func TestGateURL_String_zero_value(t *testing.T) {
	var gateURL traffictesting.GateURL

	assert.Equal(t, "", gateURL.String())
}
