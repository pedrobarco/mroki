package handlers

import (
	"net/http"
	"testing"

	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/pedrobarco/mroki/pkg/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateStandaloneCallback_scrubs_sensitive_headers(t *testing.T) {
	cfg := ProxyConfig{
		ScrubNames: []string{"Authorization", "Cookie", "Set-Cookie", "X-Api-Key"},
	}

	callback := createStandaloneCallback(cfg)

	req := proxy.ProxyRequest{
		Method: "GET",
		Path:   "/api/test",
		Headers: http.Header{
			"Authorization": {"Bearer super-secret"},
			"Content-Type":  {"application/json"},
			"X-Request-ID":  {"req-123"},
		},
		Body: []byte(`{}`),
	}

	live := proxy.ProxyResponse{
		StatusCode: 200,
		Response: &http.Response{
			StatusCode: 200,
			Header: http.Header{
				"Set-Cookie":   {"session=abc123"},
				"Content-Type": {"application/json"},
			},
		},
		Body: []byte(`{"status":"ok"}`),
	}

	shadow := proxy.ProxyResponse{
		StatusCode: 200,
		Response: &http.Response{
			StatusCode: 200,
			Header: http.Header{
				"Cookie":       {"session=abc123"},
				"Content-Type": {"application/json"},
			},
		},
		Body: []byte(`{"status":"ok"}`),
	}

	err := callback(req, live, shadow)
	require.NoError(t, err)

	// Verify the response headers were scrubbed (ProxyResponse is passed by value,
	// but the Response pointer's Header map is mutated via reassignment in the callback).
	assert.Equal(t, []string{traffictesting.RedactedValue}, live.Response.Header["Set-Cookie"])
	assert.Equal(t, []string{"application/json"}, live.Response.Header["Content-Type"])

	assert.Equal(t, []string{traffictesting.RedactedValue}, shadow.Response.Header["Cookie"])
	assert.Equal(t, []string{"application/json"}, shadow.Response.Header["Content-Type"])
}

func TestCreateStandaloneCallback_no_scrub_names_preserves_headers(t *testing.T) {
	cfg := ProxyConfig{
		ScrubNames: nil,
	}

	callback := createStandaloneCallback(cfg)

	req := proxy.ProxyRequest{
		Method: "GET",
		Path:   "/api/test",
		Headers: http.Header{
			"Authorization": {"Bearer token"},
			"X-Request-ID":  {"req-456"},
		},
		Body: []byte(`{}`),
	}

	live := proxy.ProxyResponse{
		StatusCode: 200,
		Response: &http.Response{
			StatusCode: 200,
			Header: http.Header{
				"Set-Cookie": {"session=xyz"},
			},
		},
		Body: []byte(`{"status":"ok"}`),
	}

	shadow := proxy.ProxyResponse{
		StatusCode: 200,
		Response: &http.Response{
			StatusCode: 200,
			Header: http.Header{
				"Set-Cookie": {"session=xyz"},
			},
		},
		Body: []byte(`{"status":"ok"}`),
	}

	err := callback(req, live, shadow)
	require.NoError(t, err)

	// Headers should remain untouched when no scrub names configured
	assert.Equal(t, []string{"session=xyz"}, live.Response.Header["Set-Cookie"])
	assert.Equal(t, []string{"session=xyz"}, shadow.Response.Header["Set-Cookie"])
}
