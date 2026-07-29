package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Level is the logging verbosity threshold. Use ParseLevel to build one from a
// configuration string.
type Level int

const (
	// LevelDebug enables debug, info, warn, and error records.
	LevelDebug Level = iota
	// LevelInfo enables info, warn, and error records.
	LevelInfo
	// LevelWarn enables warn and error records.
	LevelWarn
	// LevelError enables error records only.
	LevelError
)

// Format is the logging output encoding. Use ParseFormat to build one from a
// configuration string.
type Format int

const (
	// FormatText selects human-readable key=value output.
	FormatText Format = iota
	// FormatJSON selects structured JSON output.
	FormatJSON
)

// New builds a *slog.Logger for the given level and format, installs it as the
// process-wide slog default, and returns it. It is unaware of environment
// variables: callers resolve the level and format (see ParseLevel/ParseFormat)
// before calling New.
func New(level Level, format Format) *slog.Logger {
	l := slog.New(newHandler(os.Stderr, level, format))
	slog.SetDefault(l)
	return l
}

// newHandler builds the slog.Handler for the given writer, level, and format.
func newHandler(w io.Writer, level Level, format Format) slog.Handler {
	opts := &slog.HandlerOptions{Level: level.slog()}
	if format == FormatJSON {
		return slog.NewJSONHandler(w, opts)
	}
	return slog.NewTextHandler(w, opts)
}

// slog maps a Level to its slog.Level equivalent.
func (l Level) slog() slog.Level {
	switch l {
	case LevelDebug:
		return slog.LevelDebug
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// ParseLevel maps a level string (debug, info, warn, error) to a Level. It is
// case-insensitive and returns an error for empty or unrecognised values.
func ParseLevel(level string) (Level, error) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return LevelDebug, nil
	case "info":
		return LevelInfo, nil
	case "warn":
		return LevelWarn, nil
	case "error":
		return LevelError, nil
	default:
		return 0, fmt.Errorf("invalid log level %q: must be one of debug, info, warn, error", level)
	}
}

// ParseFormat maps a format string (text, json) to a Format. It is
// case-insensitive and returns an error for empty or unrecognised values.
func ParseFormat(format string) (Format, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "text":
		return FormatText, nil
	case "json":
		return FormatJSON, nil
	default:
		return 0, fmt.Errorf("invalid log format %q: must be one of text, json", format)
	}
}
