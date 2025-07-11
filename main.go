package main

import (
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"luna/commands"
	"luna/handlers"
	"luna/logger"
	"luna/servers"
	"luna/storage"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
)

var (
	commandHandlers   map[string]commands.CommandHandler
	componentHandlers map[string]commands.CommandHandler
	startTime         time.Time
)

func main() {
	logger.Init()
	godotenv.Load()

	// --- サーバー群の自動起動 ---
	serverManager := servers.NewManager()
	serverManager.AddServer(servers.NewGenericServer("Python AI Server", "python", []string{"python_server.py"}, ""))
	serverManager.AddServer(servers.NewGenericServer("Python Music Server", "python", []string{"./music_player/player.py"}, ""))
	// serverManager.AddServer(servers.NewGenericServer("C# OCR Server", "dotnet", []string{"run"}, "./csharp_server"))

	serverManager.StartAll()
	defer serverManager.StopAll()

	startTime = time.Now()
	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		logger.Fatal("環境変数 'DISCORD_BOT_TOKEN' が設定されていません。")
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		logger.Fatal("Discordセッションの作成中にエラー", "error", err)
	}

	dg.State = discordgo.NewState()
	dg.State.MaxMessageCount = 2000

	dg.Identify.Intents = discordgo.IntentsAll

	dbStore, err := storage.NewDBStore("./luna.db")
	if err != nil {
		logger.Fatal("データベースの初期化に失敗", "error", err)
	}
	defer dbStore.Close()

	scheduler := cron.New()
	commandHandlers = make(map[string]commands.CommandHandler)
	componentHandlers = make(map[string]commands.CommandHandler)

	// コマンドの登録
	appContext := &commands.AppContext{
		Store:     dbStore,
		Scheduler: scheduler,
		StartTime: startTime,
	}
	for _, cmd := range commands.RegisterAllCommands(appContext, commandHandlers) {
		registerCommand(cmd)
	}

	eventHandler := handlers.NewEventHandler(dbStore)
	eventHandler.RegisterAllHandlers(dg)

	dg.AddHandler(interactionCreate)

	if err = dg.Open(); err != nil {
		logger.Fatal("Discordへの接続中にエラー", "error", err)
	}
	defer dg.Close()

	scheduler.Start()
	defer scheduler.Stop()

	if scheduleCmd, ok := commandHandlers["schedule"].(*commands.ScheduleCommand); ok {
		scheduleCmd.LoadAndRegisterSchedules(dg)
	}

	logger.Info("Discord Botが起動しました。コマンドを登録します...")
	registeredCommands := make([]*discordgo.ApplicationCommand, 0, len(commandHandlers))
	for _, handler := range commandHandlers {
		registeredCommands = append(registeredCommands, handler.GetCommandDef())
	}

	// グローバルコマンドとして登録（サーバーIDを "" にする）
	if _, err = dg.ApplicationCommandBulkOverwrite(dg.State.User.ID, "", registeredCommands); err != nil {
		logger.Fatal("コマンドの登録に失敗しました", "error", err)
	}

	logger.Info("コマンドの登録が完了しました。Ctrl+Cで終了します。")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	logger.Info("Botをシャットダウンします...")
}

func registerCommand(cmd commands.CommandHandler) {
	def := cmd.GetCommandDef()
	commandHandlers[def.Name] = cmd
	for _, id := range cmd.GetComponentIDs() {
		componentHandlers[id] = cmd
	}
}

func interactionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h.Handle(s, i)
		}
	case discordgo.InteractionMessageComponent:
		for id, h := range componentHandlers {
			if strings.HasPrefix(i.MessageComponentData().CustomID, id) {
				h.HandleComponent(s, i)
				return
			}
		}
	case discordgo.InteractionModalSubmit:
		for id, h := range componentHandlers {
			if strings.HasPrefix(i.ModalSubmitData().CustomID, id) {
				h.HandleModal(s, i)
				return
			}
		}
	}
}
