package telemetry

import (
	"log/slog"
	"os"
)

func NewLogger(level string) *slog.Logger {
	var minLevel slog.Level

	switch level {
	case "debug":
		minLevel = slog.LevelDebug
	case "warn":
		minLevel = slog.LevelWarn
	case "error":
		minLevel = slog.LevelError
	default:
		minLevel = slog.LevelInfo
	}

	return slog.New(slog.NewJSONHandler(
		os.Stdout,
		&slog.HandlerOptions{Level: minLevel},
	)).With("service", "trip-service")
}
