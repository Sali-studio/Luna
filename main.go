package main

import (
	"os"

	"luna/bot"
	"luna/commands"
	"luna/config"
	"luna/handlers"
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
	serverManager.AddServer(servers.NewGenericServer("Python Music Server", "python", []string{"./music_player/player.py"}, ""))
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
	eventHandler := handlers.NewEventHandler(b.GetDBStore(), log)
	eventHandler.RegisterAllHandlers(b.GetSession())

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

	session := b.GetSession()
	defer func() {
		log.Info("Removing commands...")
		// Overwrite with empty slice to remove all commands
		session.ApplicationCommandBulkOverwrite(session.State.User.ID, "", []*discordgo.ApplicationCommand{})
	}()

	if _, err = session.ApplicationCommandBulkOverwrite(session.State.User.ID, "", registeredCommands); err != nil {
		log.Fatal("コマンドの登録に失敗しました", "error", err)
	}

	if err := b.Start(commandHandlers, componentHandlers); err != nil {
		log.Fatal("Botの起動に失敗しました", "error", err)
	}
}