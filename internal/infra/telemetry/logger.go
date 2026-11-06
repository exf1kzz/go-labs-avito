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

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: minLevel})
	return slog.New(&traceHandler{Handler: handler}).With("service", "trip-service")
}
