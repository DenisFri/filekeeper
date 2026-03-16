package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New creates a new slog.Logger based on the provided configuration.
// logLevel: debug, info, warn, error (default: info)
// logFormat: text, json (default: text)
func New(logLevel, logFormat string) *slog.Logger {
	level := parseLevel(logLevel)
	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	format := strings.ToLower(logFormat)
	if format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

// parseLevel converts a string log level to slog.Level.
// Returns slog.LevelInfo as the default for empty or invalid values.
func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
