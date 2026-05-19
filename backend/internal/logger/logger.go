package logger

import (
	"log"
	"log/slog"
	"os"
)

func NewStruturedLogger(logLevel string) *log.Logger {
	level := slog.LevelDebug

	switch logLevel {
	case slog.LevelDebug.String():
		level = slog.LevelDebug
	case slog.LevelWarn.String():
		level = slog.LevelWarn
	case slog.LevelInfo.String():
		level = slog.LevelInfo
	case slog.LevelError.String():
		level = slog.LevelError
	default:
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return slog.NewLogLogger(h, level)
}
