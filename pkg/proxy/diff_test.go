package proxy_test

import (
	"net/http"
	"testing"

	"github.com/pedrobarco/mroki/pkg/diff"
	"github.com/pedrobarco/mroki/pkg/proxy"
	"github.com/stretchr/testify/assert"
)

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

	_, err := differ.Diff(live, shadow)

	// Should return an error or handle gracefully
	// (behavior depends on diff.JSON implementation)
	assert.Error(t, err)
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