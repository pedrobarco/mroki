package traffictesting_test

import (
	"net/http"
	"testing"

	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/stretchr/testify/assert"
)

func TestHeaders_Scrub_replaces_matching_headers(t *testing.T) {
	h := traffictesting.NewHeaders(http.Header{
		"Authorization": {"Bearer secret-token"},
		"Content-Type":  {"application/json"},
	})

	scrubbed := h.Scrub([]string{"Authorization"})

	assert.Equal(t, []string{traffictesting.RedactedValue}, scrubbed.HTTPHeader()["Authorization"])
	assert.Equal(t, []string{"application/json"}, scrubbed.HTTPHeader()["Content-Type"])
}

func TestHeaders_Scrub_case_insensitive(t *testing.T) {
	h := traffictesting.NewHeaders(http.Header{
		"Authorization": {"Bearer token"},
	})

	// Lowercase input should still match canonical "Authorization"
	scrubbed := h.Scrub([]string{"authorization"})

	assert.Equal(t, []string{traffictesting.RedactedValue}, scrubbed.HTTPHeader()["Authorization"])
}

func TestHeaders_Scrub_does_not_mutate_original(t *testing.T) {
	original := http.Header{
		"Authorization": {"Bearer secret"},
		"Cookie":        {"session=abc"},
	}
	h := traffictesting.NewHeaders(original)

	h.Scrub([]string{"Authorization", "Cookie"})

	// Original must be untouched
	assert.Equal(t, "Bearer secret", original.Get("Authorization"))
	assert.Equal(t, "session=abc", original.Get("Cookie"))
}

func TestHeaders_Scrub_multiple_values(t *testing.T) {
	h := traffictesting.NewHeaders(http.Header{
		"Set-Cookie": {"a=1", "b=2", "c=3"},
	})

	scrubbed := h.Scrub([]string{"Set-Cookie"})

	// All values should be replaced with a single redacted value
	assert.Equal(t, []string{traffictesting.RedactedValue}, scrubbed.HTTPHeader()["Set-Cookie"])
}

func TestHeaders_Scrub_no_matching_headers(t *testing.T) {
	h := traffictesting.NewHeaders(http.Header{
		"Content-Type": {"text/plain"},
	})

	scrubbed := h.Scrub([]string{"Authorization"})

	assert.Equal(t, []string{"text/plain"}, scrubbed.HTTPHeader()["Content-Type"])
	assert.Empty(t, scrubbed.HTTPHeader()["Authorization"])
}

func TestHeaders_Scrub_empty_names(t *testing.T) {
	h := traffictesting.NewHeaders(http.Header{
		"Authorization": {"Bearer token"},
	})

	scrubbed := h.Scrub([]string{})

	assert.Equal(t, []string{"Bearer token"}, scrubbed.HTTPHeader()["Authorization"])
}

func TestHeaders_Scrub_nil_headers(t *testing.T) {
	h := traffictesting.NewHeaders(nil)

	scrubbed := h.Scrub([]string{"Authorization"})

	assert.Empty(t, scrubbed.HTTPHeader())
}
