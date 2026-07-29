package logger

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  slog.Level
	}{
		{name: "debug", input: "debug", want: slog.LevelDebug},
		{name: "info", input: "info", want: slog.LevelInfo},
		{name: "warn", input: "warn", want: slog.LevelWarn},
		{name: "warning alias", input: "warning", want: slog.LevelWarn},
		{name: "error", input: "error", want: slog.LevelError},
		{name: "case insensitive", input: "DEBUG", want: slog.LevelDebug},
		{name: "trims whitespace", input: "  info  ", want: slog.LevelInfo},
		{name: "empty defaults to info", input: "", want: slog.LevelInfo},
		{name: "unknown defaults to info", input: "verbose", want: slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ParseLevel(tt.input))
		})
	}
}

func TestConfigure_SetsDefaultAndHonoursLevel(t *testing.T) {
	// Configure installs the logger as the slog default; capture that default's
	// output by pointing a fresh logger at a buffer via the same level.
	l := Configure("warn", "text")
	require.NotNil(t, l)
	assert.Same(t, l, slog.Default())

	// Below-threshold records are dropped, at/above-threshold are emitted.
	assert.False(t, l.Enabled(t.Context(), slog.LevelInfo))
	assert.True(t, l.Enabled(t.Context(), slog.LevelWarn))
	assert.True(t, l.Enabled(t.Context(), slog.LevelError))
}

func TestConfigure_Format(t *testing.T) {
	tests := []struct {
		name       string
		format     string
		wantSubstr string
	}{
		{name: "json", format: "json", wantSubstr: `"msg":"hello"`},
		{name: "json case insensitive", format: "JSON", wantSubstr: `"msg":"hello"`},
		{name: "text", format: "text", wantSubstr: "msg=hello"},
		{name: "empty defaults to text", format: "", wantSubstr: "msg=hello"},
		{name: "unknown defaults to text", format: "xml", wantSubstr: "msg=hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			slog.New(newHandler(&buf, "info", tt.format)).Info("hello")
			assert.Contains(t, buf.String(), tt.wantSubstr)
		})
	}
}
