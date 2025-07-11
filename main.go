package main

import (
	"os"

	"luna/bot"
	"luna/config"
	"luna/logger"
	"luna/servers"
	"luna/storage"
)

func main() {
	logger.Init()
	if err := config.LoadConfig(); err != nil {
		logger.Fatal("設定ファイルの読み込みに失敗しました", "error", err)
	}

	// Google Cloudの認証情報を環境変数に設定
	if config.Cfg.Google.CredentialsPath != "" {
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", config.Cfg.Google.CredentialsPath)
	}

	// --- サーバー群の自動起動 ---
	serverManager := servers.NewManager()
	serverManager.AddServer(servers.NewGenericServer("Python AI Server", "python", []string{"python_server.py"}, ""))
	serverManager.AddServer(servers.NewGenericServer("Python Music Server", "python", []string{"./music_player/player.py"}, ""))
	// serverManager.AddServer(servers.NewGenericServer("C# OCR Server", "dotnet", []string{"run"}, "./csharp_server"))

	serverManager.StartAll()
	defer serverManager.StopAll()

	// データベースインスタンスの生成
	db, err := storage.NewDBStore("./luna.db")
	if err != nil {
		logger.Fatal("データベースの初期化に失敗しました", "error", err)
	}

	// Botにデータベースインスタンスを注入
	b, err := bot.New(db)
	if err != nil {
		logger.Fatal("Botの初期化に失敗しました", "error", err)
	}

	if err := b.Start(); err != nil {
		logger.Fatal("Botの起動に失敗しました", "error", err)
	}
}

