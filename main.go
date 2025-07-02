package main

import (
	"luna/commands"
	"luna/gemini"
	"luna/handlers"
	"luna/logger"
	"luna/storage"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
)

var (
	commandHandlers   map[string]handlers.CommandHandler
	componentHandlers map[string]handlers.CommandHandler
	startTime         time.Time
)

func main() {
	err := godotenv.Load()
	if err != nil {
		logger.Info(".envファイルが見つかりません。環境変数から直接読み込みます。")
	}

	startTime = time.Now()
	logger.Init()
	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		logger.Fatal("環境変数 'DISCORD_BOT_TOKEN' が設定されていません。")
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		logger.Fatal("Discordセッションの作成中にエラー", "error", err)
	}

	dbStore, err := storage.NewDBStore("./luna.db")
	if err != nil {
		logger.Fatal("データベースの初期化に失敗", "error", err)
	}
	defer dbStore.Close()

	geminiClient, err := gemini.NewClient(os.Getenv("GEMINI_API_KEY"))
	if err != nil {
		logger.Warn("Geminiクライアントの初期化に失敗", "error", err)
	}

	scheduler := cron.New()

	commandHandlers = make(map[string]handlers.CommandHandler)
	componentHandlers = make(map[string]handlers.CommandHandler)

	// --- 全てのコマンドを登録 ---
	registerCommand(&commands.AskCommand{Gemini: geminiClient})
	registerCommand(&commands.AvatarCommand{})
	registerCommand(&commands.CalculatorCommand{})
	registerCommand(&commands.ConfigCommand{Store: dbStore})
	registerCommand(&commands.DashboardCommand{Store: dbStore, Scheduler: scheduler})
	registerCommand(&commands.EmbedCommand{})
	registerCommand(&commands.ModerateCommand{})
	registerCommand(&commands.PingCommand{StartTime: startTime, Store: dbStore})
	registerCommand(&commands.PokemonCalculatorCommand{})
	registerCommand(&commands.PollCommand{})
	registerCommand(&commands.PowerConverterCommand{})
	registerCommand(&commands.ReactionRoleCommand{Store: dbStore})
	registerCommand(&commands.ScheduleCommand{Scheduler: scheduler, Store: dbStore})
	registerCommand(&commands.TicketCommand{Store: dbStore, Gemini: geminiClient})
	registerCommand(&commands.TranslateCommand{Gemini: geminiClient})
	registerCommand(&commands.UserInfoCommand{})
	registerCommand(&commands.WeatherCommand{APIKey: os.Getenv("WEATHER_API_KEY")})
	// 最後にヘルプコマンドを登録
	registerCommand(&commands.HelpCommand{AllCommands: commandHandlers})

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

	logger.Info("Botが起動しました。コマンドを登録します...")
	registeredCommands := make([]*discordgo.ApplicationCommand, 0, len(commandHandlers))
	for _, handler := range commandHandlers {
		registeredCommands = append(registeredCommands, handler.GetCommandDef())
	}
	if _, err = dg.ApplicationCommandBulkOverwrite(dg.State.User.ID, "", registeredCommands); err != nil {
		logger.Fatal("コマンドの登録に失敗しました", "error", err)
	}
	logger.Info("コマンドの登録が完了しました。Ctrl+Cで終了します。")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	logger.Info("Botをシャットダウンします...")
}

func registerCommand(cmd handlers.CommandHandler) {
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
