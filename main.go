package main

import (
	"luna/bot"
	"luna/commands"
	"luna/config"
	"luna/handlers/web"
	"luna/logger"
	"luna/player"
	"luna/servers"
	"luna/storage"
	"math/rand"
	"os"

	"github.com/robfig/cron/v3"
)

func main() {
	log := logger.New()

	var musicPlayer *player.Player

	if err := config.LoadConfig(log); err != nil {
		log.Fatal("設定ファイルの読み込みに失敗しました", "error", err)
	}

	// 認証システムの初期化
	web.InitAuth(config.Cfg)

	// Google Cloudの認証情報を環境変数に設定
	if config.Cfg.Google.CredentialsPath != "" {
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", config.Cfg.Google.CredentialsPath)
	}

	// サーバーの自動起動
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
	// 初回起動時に企業データをDBに登録
	if err := db.SeedInitialCompanies(); err != nil {
		log.Fatal("企業データの初期化に失敗しました", "error", err)
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

	// コマンドハンドラーを登録
	commandHandlers, componentHandlers, registeredCommands, stockCmd := commands.RegisterCommands(log, b.GetDBStore(), b.GetScheduler(), b.GetPlayer(), b.GetSession(), b.GetStartTime())

	// 5分ごとに株価を更新
	scheduler.AddFunc("@every 5m", stockCmd.UpdateStockPrices)
	scheduler.AddFunc("@hourly", func() {
		if rand.Float32() < 0.25 { // 25% chance to trigger an event every hour
			// Need a guild ID to announce the event. This is a limitation.
			// We'll need to find a way to get a valid guild ID or handle announcements differently.
			// For now, the event will trigger but not be announced in a channel.
			stockCmd.TriggerRandomEvent(b.GetSession(), "") // Passing empty guildID for now
		}
	})
	scheduler.Start()

	// Botを起動
	if err := b.Start(commandHandlers, componentHandlers, registeredCommands); err != nil {
		log.Fatal("Botの起動に失敗しました", "error", err)
	}
}
