package main

import (
	"os"

	"luna/bot"
	"luna/commands"
	"luna/config"
	"luna/handlers/events"
	"luna/interfaces"
	"luna/logger"
	"luna/servers"
	"luna/storage"

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
)

func main() {
	log := logger.New()

	if err := config.LoadConfig(log); err != nil {
		log.Fatal("設定ファイルの読み込みに失敗しました", "error", err)
	}

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

	// Botに依存性を注入
	b, err := bot.New(log, db, scheduler)
	if err != nil {
		log.Fatal("Botの初期化に失敗しました", "error", err)
	}

	// イベントハンドラの登録
	events.NewMessageHandler(log, b.GetDBStore()).Register(b.GetSession())
	events.NewMemberHandler(log, b.GetDBStore()).Register(b.GetSession())
	events.NewVoiceHandler(log, b.GetDBStore()).Register(b.GetSession())

	// コマンドの登録
	commandHandlers := make(map[string]interfaces.CommandHandler)
	componentHandlers := make(map[string]interfaces.CommandHandler)
	appContext := &commands.AppContext{
		Log:       log,
		Store:     b.GetDBStore(),
		Scheduler: b.GetScheduler(),
		StartTime: b.GetStartTime(),
	}
	registeredCommands := make([]*discordgo.ApplicationCommand, 0)
	for _, cmd := range commands.RegisterAllCommands(appContext, commandHandlers) {
		def := cmd.GetCommandDef()
		commandHandlers[def.Name] = cmd
		for _, id := range cmd.GetComponentIDs() {
			componentHandlers[id] = cmd
		}
		registeredCommands = append(registeredCommands, def)
	}

	if err := b.Start(commandHandlers, componentHandlers, registeredCommands); err != nil {
		log.Fatal("Botの起動に失敗しました", "error", err)
	}
}
