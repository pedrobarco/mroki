package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
