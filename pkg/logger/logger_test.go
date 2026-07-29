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
		name    string
		input   string
		want    Level
		wantErr bool
	}{
		{name: "debug", input: "debug", want: LevelDebug},
		{name: "info", input: "info", want: LevelInfo},
		{name: "warn", input: "warn", want: LevelWarn},
		{name: "error", input: "error", want: LevelError},
		{name: "warning is an error", input: "warning", wantErr: true},
		{name: "case insensitive", input: "DEBUG", want: LevelDebug},
		{name: "trims whitespace", input: "  info  ", want: LevelInfo},
		{name: "empty is an error", input: "", wantErr: true},
		{name: "unknown is an error", input: "verbose", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLevel(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Format
		wantErr bool
	}{
		{name: "text", input: "text", want: FormatText},
		{name: "json", input: "json", want: FormatJSON},
		{name: "case insensitive", input: "JSON", want: FormatJSON},
		{name: "trims whitespace", input: "  text  ", want: FormatText},
		{name: "empty is an error", input: "", wantErr: true},
		{name: "unknown is an error", input: "xml", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFormat(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNew_SetsDefaultAndHonoursLevel(t *testing.T) {
	// New installs the logger as the slog default and applies the level
	// threshold.
	l := New(LevelWarn, FormatText)
	require.NotNil(t, l)
	assert.Same(t, l, slog.Default())

	// Below-threshold records are dropped, at/above-threshold are emitted.
	assert.False(t, l.Enabled(t.Context(), slog.LevelInfo))
	assert.True(t, l.Enabled(t.Context(), slog.LevelWarn))
	assert.True(t, l.Enabled(t.Context(), slog.LevelError))
}

func TestNewHandler_Format(t *testing.T) {
	tests := []struct {
		name       string
		format     Format
		wantSubstr string
	}{
		{name: "json", format: FormatJSON, wantSubstr: `"msg":"hello"`},
		{name: "text", format: FormatText, wantSubstr: "msg=hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			slog.New(newHandler(&buf, LevelInfo, tt.format)).Info("hello")
			assert.Contains(t, buf.String(), tt.wantSubstr)
		})
	}
}
