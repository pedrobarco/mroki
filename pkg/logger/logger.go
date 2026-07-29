package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// New returns a bootstrap logger and installs it as the slog default. It reads
// MROKI_APP_LOG_LEVEL and MROKI_APP_LOG_FORMAT directly from the environment so
// that logs emitted before configuration is fully loaded (e.g. config
// validation errors) already honour the requested level and format. Call
// Configure again once configuration has been loaded and validated.
func New() *slog.Logger {
	return Configure(os.Getenv("MROKI_APP_LOG_LEVEL"), os.Getenv("MROKI_APP_LOG_FORMAT"))
}

// Configure builds a *slog.Logger for the given level and format, installs it as
// the process-wide slog default, and returns it. An empty or unrecognised level
// defaults to info; an empty or unrecognised format defaults to text.
func Configure(level, format string) *slog.Logger {
	l := slog.New(newHandler(os.Stderr, level, format))
	slog.SetDefault(l)
	return l
}

// newHandler builds the slog.Handler for the given writer, level, and format.
// A format of "json" (case-insensitive) selects JSON output; anything else
// falls back to text.
func newHandler(w io.Writer, level, format string) slog.Handler {
	opts := &slog.HandlerOptions{Level: ParseLevel(level)}
	if strings.EqualFold(strings.TrimSpace(format), "json") {
		return slog.NewJSONHandler(w, opts)
	}
	return slog.NewTextHandler(w, opts)
}

// ParseLevel maps a level string (debug, info, warn, error) to a slog.Level. It
// is case-insensitive and defaults to slog.LevelInfo for empty or unrecognised
// values.
func ParseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
