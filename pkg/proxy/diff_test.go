package proxy_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/pedrobarco/mroki/pkg/diff"
	"github.com/pedrobarco/mroki/pkg/proxy"
	"github.com/stretchr/testify/assert"
)

// makeResp builds a ProxyResponse with the given status, Content-Type, and body.
// An empty contentType leaves the Content-Type header unset.
func makeResp(status int, contentType string, body []byte) proxy.ProxyResponse {
	h := http.Header{}
	if contentType != "" {
		h.Set("Content-Type", contentType)
	}
	return proxy.ProxyResponse{
		StatusCode: status,
		Response:   &http.Response{Header: h},
		Body:       body,
	}
}

func TestProxyResponseDiffer_Diff_identical_responses(t *testing.T) {
	differ := proxy.NewProxyResponseDiffer()

	live := proxy.ProxyResponse{
		StatusCode: 200,
		Response: &http.Response{
			Header: http.Header{"Content-Type": []string{"application/json"}},
		},
		Body: []byte(`{"status":"ok"}`),
	}
	shadow := proxy.ProxyResponse{
		StatusCode: 200,
		Response: &http.Response{
			Header: http.Header{"Content-Type": []string{"application/json"}},
		},
		Body: []byte(`{"status":"ok"}`),
	}

	ops, err := differ.Diff(live, shadow)

	assert.NoError(t, err)
	assert.Empty(t, ops)
}

func TestProxyResponseDiffer_Diff_different_status_codes(t *testing.T) {
	differ := proxy.NewProxyResponseDiffer()

	live := proxy.ProxyResponse{
		StatusCode: 200,
		Response: &http.Response{
			Header: http.Header{},
		},
		Body: []byte(`{"status":"ok"}`),
	}
	shadow := proxy.ProxyResponse{
		StatusCode: 500,
		Response: &http.Response{
			Header: http.Header{},
		},
		Body: []byte(`{"status":"error"}`),
	}

	ops, err := differ.Diff(live, shadow)

	assert.NoError(t, err)
	assert.NotEmpty(t, ops)

	paths := map[string]bool{}
	for _, op := range ops {
		paths[op.Path] = true
	}
	assert.True(t, paths["/statusCode"])
}

func TestProxyResponseDiffer_Diff_different_bodies(t *testing.T) {
	differ := proxy.NewProxyResponseDiffer()

	live := proxy.ProxyResponse{
		StatusCode: 200,
		Response: &http.Response{
			Header: http.Header{},
		},
		Body: []byte(`{"user":"alice"}`),
	}
	shadow := proxy.ProxyResponse{
		StatusCode: 200,
		Response: &http.Response{
			Header: http.Header{},
		},
		Body: []byte(`{"user":"bob"}`),
	}

	ops, err := differ.Diff(live, shadow)

	assert.NoError(t, err)
	assert.NotEmpty(t, ops)
}

func TestProxyResponseDiffer_Diff_different_headers(t *testing.T) {
	differ := proxy.NewProxyResponseDiffer()

	live := proxy.ProxyResponse{
		StatusCode: 200,
		Response: &http.Response{
			Header: http.Header{"X-Request-Id": []string{"abc"}},
		},
		Body: []byte(`{"ok":true}`),
	}
	shadow := proxy.ProxyResponse{
		StatusCode: 200,
		Response: &http.Response{
			Header: http.Header{"X-Request-Id": []string{"def"}},
		},
		Body: []byte(`{"ok":true}`),
	}

	ops, err := differ.Diff(live, shadow)

	assert.NoError(t, err)
	assert.NotEmpty(t, ops, "should detect header differences")
}

func TestProxyResponseDiffer_Diff_empty_bodies(t *testing.T) {
	differ := proxy.NewProxyResponseDiffer()

	live := proxy.ProxyResponse{
		StatusCode: 200,
		Response: &http.Response{
			Header: http.Header{},
		},
		Body: []byte(`{}`),
	}
	shadow := proxy.ProxyResponse{
		StatusCode: 200,
		Response: &http.Response{
			Header: http.Header{},
		},
		Body: []byte(`{}`),
	}

	ops, err := differ.Diff(live, shadow)

	assert.NoError(t, err)
	assert.Empty(t, ops)
}

func TestProxyResponseDiffer_Diff_invalid_json_body(t *testing.T) {
	differ := proxy.NewProxyResponseDiffer()

	live := proxy.ProxyResponse{
		StatusCode: 200,
		Response: &http.Response{
			Header: http.Header{},
		},
		Body: []byte(`not json`),
	}
	shadow := proxy.ProxyResponse{
		StatusCode: 200,
		Response: &http.Response{
			Header: http.Header{},
		},
		Body: []byte(`{"ok":true}`),
	}

	ops, err := differ.Diff(live, shadow)

	// A JSON-classified body (missing Content-Type defaults to JSON) that fails
	// to parse falls back to a raw string, so the diff still surfaces instead of
	// failing live traffic.
	assert.NoError(t, err)
	assert.NotEmpty(t, ops)
}

func TestProxyResponseDiffer_Diff_nested_objects(t *testing.T) {
	differ := proxy.NewProxyResponseDiffer()

	live := proxy.ProxyResponse{
		StatusCode: 200,
		Response: &http.Response{
			Header: http.Header{},
		},
		Body: []byte(`{"user":{"name":"alice","age":30}}`),
	}
	shadow := proxy.ProxyResponse{
		StatusCode: 200,
		Response: &http.Response{
			Header: http.Header{},
		},
		Body: []byte(`{"user":{"name":"bob","age":25}}`),
	}

	ops, err := differ.Diff(live, shadow)

	assert.NoError(t, err)
	assert.NotEmpty(t, ops)
	// Should detect changes in nested paths
	assert.True(t, len(ops) >= 2, "expected at least 2 changes (name + age)")
}

func TestProxyResponseDiffer_Diff_with_diff_options(t *testing.T) {
	differ := proxy.NewProxyResponseDiffer(
		diff.WithIgnoredFields("body.timestamp"),
	)

	live := proxy.ProxyResponse{
		StatusCode: 200,
		Response: &http.Response{
			Header: http.Header{},
		},
		Body: []byte(`{"user":"alice","timestamp":"2024-01-01"}`),
	}
	shadow := proxy.ProxyResponse{
		StatusCode: 200,
		Response: &http.Response{
			Header: http.Header{},
		},
		Body: []byte(`{"user":"alice","timestamp":"2024-01-02"}`),
	}

	ops, err := differ.Diff(live, shadow)

	assert.NoError(t, err)
	assert.Empty(t, ops, "timestamp should be ignored")
}

// TestProxyResponseDiffer_Diff_contentTypeEmbedding exercises the Content-Type
// driven body embedding strategies: JSON content types are compared as
// structured JSON, non-JSON text as raw strings, and binary content is skipped
// (only its metadata note participates in the diff).
func TestProxyResponseDiffer_Diff_contentTypeEmbedding(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		liveBody    []byte
		shadowBody  []byte
		wantOps     bool
	}{
		{
			name:        "json identical",
			contentType: "application/json",
			liveBody:    []byte(`{"a":1}`),
			shadowBody:  []byte(`{"a":1}`),
			wantOps:     false,
		},
		{
			name:        "json different",
			contentType: "application/json",
			liveBody:    []byte(`{"a":1}`),
			shadowBody:  []byte(`{"a":2}`),
			wantOps:     true,
		},
		{
			name:        "json with charset parameter is parsed as json",
			contentType: "application/json; charset=utf-8",
			liveBody:    []byte(`{"a":1}`),
			shadowBody:  []byte(`{"a":2}`),
			wantOps:     true,
		},
		{
			name:        "json suffix content type is parsed as json",
			contentType: "application/vnd.api+json",
			liveBody:    []byte(`{"a":1}`),
			shadowBody:  []byte(`{"a":2}`),
			wantOps:     true,
		},
		{
			name:        "invalid json under json content type falls back to raw string",
			contentType: "application/json",
			liveBody:    []byte(`not json`),
			shadowBody:  []byte(`{"a":1}`),
			wantOps:     true,
		},
		{
			name:        "text plain identical",
			contentType: "text/plain",
			liveBody:    []byte("hello world"),
			shadowBody:  []byte("hello world"),
			wantOps:     false,
		},
		{
			name:        "text plain different is compared as raw string",
			contentType: "text/plain; charset=utf-8",
			liveBody:    []byte("hello world"),
			shadowBody:  []byte("goodbye world"),
			wantOps:     true,
		},
		{
			name:        "non-json text does not error",
			contentType: "text/plain",
			liveBody:    []byte("plain <not> json {"),
			shadowBody:  []byte("plain <not> json {"),
			wantOps:     false,
		},
		{
			name:        "html different",
			contentType: "text/html",
			liveBody:    []byte("<p>alice</p>"),
			shadowBody:  []byte("<p>bob</p>"),
			wantOps:     true,
		},
		{
			name:        "xml different",
			contentType: "application/xml",
			liveBody:    []byte("<user>alice</user>"),
			shadowBody:  []byte("<user>bob</user>"),
			wantOps:     true,
		},
		{
			name:        "binary identical bytes produce no diff",
			contentType: "application/octet-stream",
			liveBody:    []byte{0x00, 0x01, 0x02},
			shadowBody:  []byte{0x00, 0x01, 0x02},
			wantOps:     false,
		},
		{
			name:        "binary same size different bytes surface via sha256",
			contentType: "application/octet-stream",
			liveBody:    []byte{0x00, 0x01, 0x02},
			shadowBody:  []byte{0xFF, 0xFE, 0xFD},
			wantOps:     true,
		},
		{
			name:        "binary different size surfaces via metadata note",
			contentType: "application/octet-stream",
			liveBody:    []byte{0x00, 0x01, 0x02},
			shadowBody:  []byte{0x00, 0x01, 0x02, 0x03, 0x04},
			wantOps:     true,
		},
		{
			name:        "binary non-json bytes never error",
			contentType: "image/png",
			liveBody:    []byte{0x89, 0x50, 0x4E, 0x47},
			shadowBody:  []byte{0x89, 0x50, 0x4E, 0x47, 0x0D},
			wantOps:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			differ := proxy.NewProxyResponseDiffer()

			ops, err := differ.Diff(
				makeResp(200, tt.contentType, tt.liveBody),
				makeResp(200, tt.contentType, tt.shadowBody),
			)

			assert.NoError(t, err)
			if tt.wantOps {
				assert.NotEmpty(t, ops)
			} else {
				assert.Empty(t, ops)
			}
		})
	}
}

// TestProxyResponseDiffer_Diff_binaryMetadataNote verifies that a skipped binary
// body is replaced with a metadata note, so size differences surface under the
// note path rather than embedding raw bytes.
func TestProxyResponseDiffer_Diff_binaryMetadataNote(t *testing.T) {
	differ := proxy.NewProxyResponseDiffer()

	live := makeResp(200, "application/octet-stream", []byte{0x00, 0x01, 0x02})
	shadow := makeResp(200, "application/octet-stream", []byte{0x00, 0x01, 0x02, 0x03, 0x04})

	ops, err := differ.Diff(live, shadow)

	assert.NoError(t, err)
	assert.NotEmpty(t, ops)

	found := false
	for _, op := range ops {
		if strings.Contains(op.Path, "_mroki") {
			found = true
		}
	}
	assert.True(t, found, "binary size difference should surface under the /body/_mroki metadata note")
}

// TestProxyResponseDiffer_Diff_mixedContentTypes ensures a JSON live response
// compared against a text shadow response embeds each side per its own
// Content-Type and reports a difference without erroring.
func TestProxyResponseDiffer_Diff_mixedContentTypes(t *testing.T) {
	differ := proxy.NewProxyResponseDiffer()

	live := makeResp(200, "application/json", []byte(`{"ok":true}`))
	shadow := makeResp(200, "text/plain", []byte("ok"))

	ops, err := differ.Diff(live, shadow)

	assert.NoError(t, err)
	assert.NotEmpty(t, ops)
}
