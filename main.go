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

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
)

var (
	commandHandlers   map[string]handlers.CommandHandler
	componentHandlers map[string]handlers.CommandHandler
)

func main() {
	// 1. 初期化
	logger.Init()
	token := os.Getenv("DISCORD_BOT_TOKEN")
	geminiAPIKey := os.Getenv("GEMINI_API_KEY")
	weatherAPIKey := os.Getenv("WEATHER_API_KEY")
	translateAPIURL := os.Getenv("GOOGLE_TRANSLATE_API_URL")

	if token == "" {
		logger.Fatal.Println("環境変数 'DISCORD_BOT_TOKEN' が設定されていません。")
		return
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		logger.Fatal.Printf("Discordセッションの作成中にエラー: %v", err)
	}

	// 2. 依存関係のセットアップ
	configStore, err := storage.NewConfigStore("config.json")
	if err != nil {
		logger.Fatal.Fatalf("設定ストアの初期化に失敗: %v", err)
	}

	geminiClient, err := gemini.NewClient(geminiAPIKey)
	if err != nil {
		logger.Warning.Printf("Geminiクライアントの初期化に失敗: %v。askコマンドは無効になります。", err)
	}

	scheduler := cron.New()

	// 3. ハンドラの登録
	commandHandlers = make(map[string]handlers.CommandHandler)
	componentHandlers = make(map[string]handlers.CommandHandler)

	// --- 全てのコマンドを登録 ---
	registerCommand(&commands.AskCommand{Gemini: geminiClient})
	registerCommand(&commands.CalculatorCommand{})
	registerCommand(&commands.ConfigCommand{Store: configStore})
	registerCommand(&commands.DashboardCommand{Store: configStore, Scheduler: scheduler})
	registerCommand(&commands.EmbedCommand{})
	registerCommand(&commands.HelpCommand{})
	registerCommand(&commands.ModerateCommand{})
	registerCommand(&commands.PingCommand{})
	registerCommand(&commands.PokemonCalculatorCommand{})
	registerCommand(&commands.PollCommand{})
	registerCommand(&commands.ReactionRoleCommand{Store: configStore})
	registerCommand(&commands.ScheduleCommand{Scheduler: scheduler})
	registerCommand(&commands.TicketCommand{Store: configStore})
	registerCommand(&commands.TranslateCommand{APIURL: translateAPIURL})
	registerCommand(&commands.UserInfoCommand{})
	registerCommand(&commands.WeatherCommand{APIKey: weatherAPIKey})

	// 4. イベントハンドラの登録
	dg.AddHandler(interactionCreate)

	// ログ、リアクションロール、一時VCなどのイベントベースの機能を登録
	eventHandler := handlers.NewEventHandler(configStore)
	eventHandler.RegisterAllHandlers(dg)

	// 5. Botの起動
	err = dg.Open()
	if err != nil {
		logger.Fatal.Printf("Discordへの接続中にエラー: %v", err)
	}
	defer dg.Close()

	scheduler.Start()
	defer scheduler.Stop()

	logger.Info.Println("Botが起動しました。スラッシュコマンドを登録します...")

	registeredCommands := make([]*discordgo.ApplicationCommand, 0, len(commandHandlers))
	for _, handler := range commandHandlers {
		registeredCommands = append(registeredCommands, handler.GetCommandDef())
	}

	_, err = dg.ApplicationCommandBulkOverwrite(dg.State.User.ID, "", registeredCommands)
	if err != nil {
		logger.Fatal.Printf("コマンドの登録に失敗しました: %v", err)
	}

	logger.Info.Println("コマンドの登録が完了しました。Ctrl+Cで終了します。")

	// 6. 終了シグナルを待機
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	logger.Info.Println("Botをシャットダウンします...")
}

func registerCommand(cmd handlers.CommandHandler) {
	def := cmd.GetCommandDef()
	commandHandlers[def.Name] = cmd

	// ボタンやモーダルのCustomIDを、プレフィックス（前方一致）で判定できるように登録
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
		// CustomIDの前方一致でハンドラを検索
		for id, h := range componentHandlers {
			if strings.HasPrefix(i.MessageComponentData().CustomID, id) {
				h.HandleComponent(s, i)
				return
			}
		}
	case discordgo.InteractionModalSubmit:
		// CustomIDの前方一致でハンドラを検索
		for id, h := range componentHandlers {
			if strings.HasPrefix(i.ModalSubmitData().CustomID, id) {
				h.HandleModal(s, i)
				return
			}
		}
	}
}
