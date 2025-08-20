package logger

import (
	"log/slog"
	"os"
)

type Logger struct {
	*slog.Logger
}

func SetupLogger() *Logger {
	logger := slog.New(slog.NewJSONHandler(os.Stdout,
		&slog.HandlerOptions{Level: slog.LevelError}))
	return &Logger{logger}
}
