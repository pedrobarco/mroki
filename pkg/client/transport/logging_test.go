package transport

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoggingRoundTripper_LogsSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	rt := NewLoggingRoundTripper(http.DefaultTransport, logger)
	req, err := http.NewRequest(http.MethodGet, server.URL+"/test", nil)
	require.NoError(t, err)

	resp, err := rt.RoundTrip(req)
	require.NoError(t, err)
	defer func() { require.NoError(t, resp.Body.Close()) }()

	logOutput := buf.String()
	assert.Contains(t, logOutput, "outgoing HTTP request")
	assert.Contains(t, logOutput, "GET")
	assert.Contains(t, logOutput, "/test")
	assert.Contains(t, logOutput, "status=200")
}

func TestLoggingRoundTripper_LogsFailure(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	rt := NewLoggingRoundTripper(http.DefaultTransport, logger)
	req, err := http.NewRequest(http.MethodGet, "http://localhost:1/fail", nil)
	require.NoError(t, err)

	resp, err := rt.RoundTrip(req)
	assert.Error(t, err)
	assert.Nil(t, resp)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "outgoing HTTP request failed")
	assert.Contains(t, logOutput, "GET")
	assert.Contains(t, logOutput, "/fail")
}
