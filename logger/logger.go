package logger

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	// 各レベルのロガーを定義
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
	Fatal   *log.Logger
)

// Init はロガーを初期化します。
func Init() {
	// "logs" ディレクトリがなければ作成
	logDir := "logs"
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		os.Mkdir(logDir, 0755)
	}

	// タイムスタンプに基づいたログファイル名を作成 (例: 2023-10-27.log)
	logFile, err := os.OpenFile(
		filepath.Join(logDir, time.Now().Format("2006-01-02")+".log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0666,
	)
	if err != nil {
		log.Fatalf("ログファイルの作成に失敗しました: %v", err)
	}

	// コンソールとファイルの両方に出力する
	multiWriter := io.MultiWriter(os.Stdout, logFile)

	// 各レベルのロガーを設定
	Info = log.New(multiWriter, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	Warning = log.New(multiWriter, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(multiWriter, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	Fatal = log.New(multiWriter, "FATAL: ", log.Ldate|log.Ltime|log.Lshortfile)
}
