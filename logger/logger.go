package logger

import (
	"io"
	"log/slog"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger は、アプリケーション全体で使用されるロガーのインターフェースを定義します。
// これにより、ロガーの実装を注入することができ、テストが容易になります。
type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Fatal(msg string, args ...any)
}

// slogLogger は、標準ライブラリのslogをラップしたLoggerの実装です。
type slogLogger struct {
	log *slog.Logger
}

// New は、新しいLoggerインスタンスを生成して返します。
func New() Logger {
	logFile := &lumberjack.Logger{
		Filename:   "luna.log",
		MaxSize:    10, // megabytes
		MaxBackups: 5,
		MaxAge:     30, // days
		Compress:   true,
	}

	multiWriter := io.MultiWriter(os.Stdout, logFile)

	logLevel := new(slog.Level)
	if err := logLevel.UnmarshalText([]byte(os.Getenv("LUNA_LOG_LEVEL"))); err != nil {
		*logLevel = slog.LevelInfo // Default to INFO level
	}

	logger := slog.New(slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
		AddSource: true,
		Level:     *logLevel,
	}))

	return &slogLogger{log: logger}
}

func (s *slogLogger) Info(msg string, args ...any) {
	s.log.Info(msg, args...)
}

func (s *slogLogger) Warn(msg string, args ...any) {
	s.log.Warn(msg, args...)
}

func (s *slogLogger) Error(msg string, args ...any) {
	s.log.Error(msg, args...)
}

func (s *slogLogger) Fatal(msg string, args ...any) {
	s.log.Error(msg, args...) // FatalはErrorとしてログを出し、その後終了する
	os.Exit(1)
}
