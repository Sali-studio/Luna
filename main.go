package main

import (
	"os"

	"luna/bot"
	"luna/commands"
	"luna/config"
	"luna/handlers/events"
	"luna/handlers/web"
	"luna/logger"
	"luna/player"
	"luna/servers"
	"luna/storage"

	"github.com/robfig/cron/v3"
)

func main() {
	log := logger.New()

	var musicPlayer *player.Player // musicPlayerをここで宣言

	if err := config.LoadConfig(log); err != nil {
		log.Fatal("設定ファイルの読み込みに失敗しました", "error", err)
	}

	// 認証システムの初期化
	web.InitAuth(config.Cfg)

	// Google Cloudの認証情報を環境変数に設定
	if config.Cfg.Google.CredentialsPath != "" {
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", config.Cfg.Google.CredentialsPath)
	}

	// --- サーバー群の自動起動 ---
	serverManager := servers.NewManager(log)
	serverManager.AddServer(servers.NewGenericServer("Python AI Server", "python", []string{"python_server.py"}, ""))
	// serverManager.AddServer(servers.NewGenericServer("C# OCR Server", "dotnet", []string{"run"}, "./csharp_server"))

	serverManager.StartAll()
	defer serverManager.StopAll()

	// 依存関係のインスタンスを生成
	db, err := storage.NewDBStore("./luna.db")
	if err != nil {
		log.Fatal("データベースの初期化に失敗しました", "error", err)
	}
	scheduler := cron.New()

	// 音楽プレイヤーのインスタンスを先に生成 (Sessionは後で設定)
		musicPlayer = player.NewPlayer(nil, log, db)

	// Botに依存性を注入
	b, err := bot.New(log, db, scheduler, musicPlayer)
	if err != nil {
		log.Fatal("Botの初期化に失敗しました", "error", err)
	}

	// BotのSessionをPlayerに設定
	musicPlayer.Session = b.GetSession()

	// イベントハンドラーを登録
	events.NewChannelHandler(log, db).Register(b.GetSession())
	events.NewMemberHandler(log, db).Register(b.GetSession())
	events.NewMessageHandler(log, db).Register(b.GetSession())
	events.NewVoiceHandler(log, db).Register(b.GetSession())

	// コマンドハンドラーを登録
	commandHandlers, componentHandlers, registeredCommands := commands.RegisterCommands(b.GetDBStore(), b.GetScheduler(), b.GetPlayer(), b.GetSession(), b.GetStartTime())

	// Botを起動
	if err := b.Start(commandHandlers, componentHandlers, registeredCommands); err != nil {
		log.Fatal("Botの起動に失敗しました", "error", err)
	}
}
