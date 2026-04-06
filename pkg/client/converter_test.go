package client

import (
	"net/http"
	"testing"
	"time"

	"github.com/pedrobarco/mroki/pkg/proxy"
	"github.com/stretchr/testify/assert"
)

func TestConvertProxyToCapture(t *testing.T) {
	// Create test data
	proxyReq := proxy.ProxyRequest{
		Method: "POST",
		Path:   "/api/users",
		Headers: http.Header{
			"Content-Type": []string{"application/json"},
			"X-Request-ID": []string{"req-123"},
		},
		Body: []byte(`{"name":"John"}`),
	}

	liveResp := proxy.ProxyResponse{
		StatusCode: 200,
		Response: &http.Response{
			StatusCode: 200,
			Header: http.Header{
				"Content-Type":  []string{"application/json"},
				"X-Response-ID": []string{"resp-live"},
			},
		},
		Body:      []byte(`{"id":1,"name":"John"}`),
		LatencyMs: 142,
	}

	shadowResp := proxy.ProxyResponse{
		StatusCode: 201,
		Response: &http.Response{
			StatusCode: 201,
			Header: http.Header{
				"Content-Type":  []string{"application/json"},
				"X-Response-ID": []string{"resp-shadow"},
			},
		},
		Body:      []byte(`{"id":2,"name":"John"}`),
		LatencyMs: 187,
	}

	// Convert
	captured := ConvertProxyToCapture(proxyReq, liveResp, shadowResp)

	// Verify request fields
	assert.Equal(t, "POST", captured.Method)
	assert.Equal(t, "/api/users", captured.Path)
	assert.Equal(t, map[string][]string(proxyReq.Headers), captured.Headers)
	assert.Equal(t, "eyJuYW1lIjoiSm9obiJ9", captured.Body) // Base64 encoded {"name":"John"}
	assert.NotZero(t, captured.CreatedAt)

	// Verify live response
	liveCapture := captured.LiveResponse
	assert.Equal(t, 200, liveCapture.StatusCode)
	assert.Equal(t, map[string][]string(liveResp.Response.Header), liveCapture.Headers)
	assert.Equal(t, "eyJpZCI6MSwibmFtZSI6IkpvaG4ifQ==", liveCapture.Body) // Base64 encoded {"id":1,"name":"John"}
	assert.Equal(t, int64(142), liveCapture.LatencyMs)

	// Verify shadow response
	shadowCapture := captured.ShadowResponse
	assert.Equal(t, 201, shadowCapture.StatusCode)
	assert.Equal(t, map[string][]string(shadowResp.Response.Header), shadowCapture.Headers)
	assert.Equal(t, "eyJpZCI6MiwibmFtZSI6IkpvaG4ifQ==", shadowCapture.Body) // Base64 encoded {"id":2,"name":"John"}
	assert.Equal(t, int64(187), shadowCapture.LatencyMs)

	// Verify diff is nil (computed server-side)
	assert.Nil(t, captured.Diff)

	// Verify timestamps are consistent
	assert.Equal(t, captured.CreatedAt, liveCapture.CreatedAt)
	assert.Equal(t, captured.CreatedAt, shadowCapture.CreatedAt)
}

func TestConvertProxyToCapture_EmptyBody(t *testing.T) {
	// Test with empty request body
	proxyReq := proxy.ProxyRequest{
		Method:  "GET",
		Path:    "/api/users",
		Headers: http.Header{},
		Body:    []byte{},
	}

	liveResp := proxy.ProxyResponse{
		StatusCode: 204,
		Response: &http.Response{
			StatusCode: 204,
			Header:     http.Header{},
		},
		Body: []byte{},
	}

	shadowResp := proxy.ProxyResponse{
		StatusCode: 204,
		Response: &http.Response{
			StatusCode: 204,
			Header:     http.Header{},
		},
		Body: []byte{},
	}

	captured := ConvertProxyToCapture(proxyReq, liveResp, shadowResp)

	// Base64 encoding of empty byte array is empty string
	assert.Equal(t, "", captured.Body)
	assert.Equal(t, "", captured.LiveResponse.Body)
	assert.Equal(t, "", captured.ShadowResponse.Body)
}

func TestConvertProxyToCapture_MultipleHeaders(t *testing.T) {
	// Test with multiple header values
	proxyReq := proxy.ProxyRequest{
		Method: "GET",
		Path:   "/api/users",
		Headers: http.Header{
			"Accept":          []string{"application/json", "text/plain"},
			"Accept-Encoding": []string{"gzip", "deflate"},
		},
		Body: []byte{},
	}

	liveResp := proxy.ProxyResponse{
		StatusCode: 200,
		Response: &http.Response{
			StatusCode: 200,
			Header: http.Header{
				"Set-Cookie": []string{"session=abc", "user=xyz"},
			},
		},
		Body: []byte(`{}`),
	}

	shadowResp := proxy.ProxyResponse{
		StatusCode: 200,
		Response: &http.Response{
			StatusCode: 200,
			Header: http.Header{
				"Set-Cookie": []string{"session=def", "user=uvw"},
			},
		},
		Body: []byte(`{}`),
	}

	captured := ConvertProxyToCapture(proxyReq, liveResp, shadowResp)

	// Verify request headers preserved
	assert.Equal(t, []string{"application/json", "text/plain"}, captured.Headers["Accept"])
	assert.Equal(t, []string{"gzip", "deflate"}, captured.Headers["Accept-Encoding"])

	// Verify response headers preserved
	assert.Equal(t, []string{"session=abc", "user=xyz"}, captured.LiveResponse.Headers["Set-Cookie"])
	assert.Equal(t, []string{"session=def", "user=uvw"}, captured.ShadowResponse.Headers["Set-Cookie"])
}

func TestConvertProxyToCapture_TimestampConsistency(t *testing.T) {
	proxyReq := proxy.ProxyRequest{
		Method:  "GET",
		Path:    "/test",
		Headers: http.Header{},
		Body:    []byte{},
	}

	liveResp := proxy.ProxyResponse{
		StatusCode: 200,
		Response:   &http.Response{StatusCode: 200, Header: http.Header{}},
		Body:       []byte(`{}`),
	}

	shadowResp := proxy.ProxyResponse{
		StatusCode: 200,
		Response:   &http.Response{StatusCode: 200, Header: http.Header{}},
		Body:       []byte(`{}`),
	}

	before := time.Now()
	captured := ConvertProxyToCapture(proxyReq, liveResp, shadowResp)
	after := time.Now()

	// Verify timestamp is within reasonable range
	assert.True(t, captured.CreatedAt.After(before) || captured.CreatedAt.Equal(before))
	assert.True(t, captured.CreatedAt.Before(after) || captured.CreatedAt.Equal(after))

	// Verify all timestamps are the same
	assert.Equal(t, captured.CreatedAt, captured.LiveResponse.CreatedAt)
	assert.Equal(t, captured.CreatedAt, captured.ShadowResponse.CreatedAt)
}
