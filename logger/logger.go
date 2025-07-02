package logger

import (
	"log/slog"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

// シングルトンとしてロガーを保持
var logger *slog.Logger

func Init() {
	// ログローテーションの設定
	logFile := &lumberjack.Logger{
		Filename:   "luna.log", // ログファイル名
		MaxSize:    10,         // 1ファイルあたりの最大サイズ (MB)
		MaxBackups: 5,          // 保持する古いログの最大数
		MaxAge:     30,         // 古いログを保持する最大日数
		Compress:   true,       // 古いログをgzipで圧縮
	}

	// ログの出力先を「標準出力（コンソール）」と「ファイル」の両方に設定
	multiWriter := os.Stdout
	if logFile != nil {
		multiWriter = os.MultiWriter(os.Stdout, logFile)
	}

	logger = slog.New(slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
		AddSource: true,            // ソースコードのファイル名と行番号を追加
		Level:     slog.LevelDebug, // DEBUGレベル以上のログをすべて記録
	}))
}

// Infoレベルのログを出力
// 例: logger.Info("Botが起動しました", "version", "1.2.3")
func Info(msg string, args ...any) {
	logger.Info(msg, args...)
}

// Warnレベルのログを出力
// 例: logger.Warn("APIキーが設定されていません", "env", "GEMINI_API_KEY")
func Warn(msg string, args ...any) {
	logger.Warn(msg, args...)
}

// Errorレベルのログを出力
// 例: logger.Error("コマンドの実行に失敗", "error", err, "command", "ping")
func Error(msg string, args ...any) {
	logger.Error(msg, args...)
}

// Fatalレベルのログを出力（出力後にプログラムを終了）
func Fatal(msg string, args ...any) {
	logger.Error(msg, args...)
	os.Exit(1)
}
