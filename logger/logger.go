package logger

import (
	"io"
	"log/slog"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

var logger *slog.Logger

func Init() {
	logFile := &lumberjack.Logger{
		Filename:   "luna.log",
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
	}

	multiWriter := io.MultiWriter(os.Stdout, logFile)

	logLevel := new(slog.Level)
	if err := logLevel.UnmarshalText([]byte(os.Getenv("LUNA_LOG_LEVEL"))); err != nil {
		*logLevel = slog.LevelInfo // Default to INFO level
	}

	logger = slog.New(slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
		AddSource: true,
		Level:     *logLevel,
	}))
}

func Info(msg string, args ...any) {
	logger.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	logger.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	logger.Error(msg, args...)
}

func Fatal(msg string, args ...any) {
	logger.Error(msg, args...)
	// Ensure the log is written before exiting
	_ = logger.Handler().Enabled(nil, slog.LevelError)
	os.Exit(1)
}
