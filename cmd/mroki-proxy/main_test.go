package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pedrobarco/mroki/pkg/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutboundTarget(t *testing.T) {
	const (
		liveHost   = "live.example.com"
		shadowHost = "shadow.example.com"
	)

	tests := []struct {
		name       string
		host       string
		shadowMode bool
		want       string
	}{
		{name: "live by host", host: liveHost, want: "live"},
		{name: "shadow by host", host: shadowHost, want: "shadow"},
		{name: "unknown host", host: "other.example.com", want: "unknown"},
		{
			name:       "shadow by header on shared host",
			host:       liveHost,
			shadowMode: true,
			want:       "shadow",
		},
		{
			name:       "shadow header wins regardless of host",
			host:       "other.example.com",
			shadowMode: true,
			want:       "shadow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://"+tt.host+"/anything", nil)
			if tt.shadowMode {
				req.Header.Set(proxy.ShadowHeader, proxy.ShadowHeaderValue)
			}

			assert.Equal(t, tt.want, outboundTarget(req, liveHost, shadowHost))
		})
	}
}

// TestOutboundTarget_SharedHost guards the original bug: when live and shadow
// share a host the header is the only signal that distinguishes them, so a
// host-only resolver would mislabel the shadow call as "live".
func TestOutboundTarget_SharedHost(t *testing.T) {
	const sharedHost = "httpbin.org"

	live := httptest.NewRequest(http.MethodGet, "http://"+sharedHost+"/anything?service=live", nil)
	shadow := httptest.NewRequest(http.MethodGet, "http://"+sharedHost+"/anything?service=shadow", nil)
	shadow.Header.Set(proxy.ShadowHeader, proxy.ShadowHeaderValue)

	assert.Equal(t, "live", outboundTarget(live, sharedHost, sharedHost))
	assert.Equal(t, "shadow", outboundTarget(shadow, sharedHost, sharedHost))
}

func TestNewProxyMetrics_Disabled(t *testing.T) {
	platform, recorder, err := newProxyMetrics(false)

	require.NoError(t, err)
	assert.Nil(t, platform, "no platform should be built when metrics are disabled")
	assert.Nil(t, recorder)
	// A nil platform's Shutdown is a no-op rather than a panic.
	assert.NoError(t, platform.Shutdown(context.Background()))
}

func TestNewProxyMetrics_Enabled(t *testing.T) {
	platform, recorder, err := newProxyMetrics(true)

	require.NoError(t, err)
	require.NotNil(t, platform)
	require.NotNil(t, platform.Provider)
	require.NotNil(t, platform.MetricsHandler())
	require.NotNil(t, recorder)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	platform.MetricsHandler().ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	// Runtime collectors registered by newProxyMetrics should be exposed.
	assert.Contains(t, w.Body.String(), "go_goroutines")

	assert.NoError(t, platform.Shutdown(context.Background()))
}
