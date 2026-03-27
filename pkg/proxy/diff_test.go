package proxy_test

import (
	"net/http"
	"testing"

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
