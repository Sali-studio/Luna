package main

import (
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"luna/commands"
	"luna/handlers"
	"luna/logger"
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

	// 1. PythonのAIサーバーを起動
	log.Println("Starting Python AI server...")
	pyCmd := exec.Command("python", "python_server.py")
	pyCmd.Stdout = os.Stdout
	pyCmd.Stderr = os.Stderr
	if err := pyCmd.Start(); err != nil {
		log.Fatalf("Failed to start Python server: %v", err)
	}
	defer pyCmd.Process.Kill()
	log.Println("Python AI server started successfully.")

	// 2. C#のOCRサーバーを起動 --無効化--
	//log.Println("Starting C# OCR server...")
	//csCmd := exec.Command("dotnet", "run")
	//csCmd.Dir = "./csharp_server" // csharp_serverフォルダ内で実行
	//csCmd.Stdout = os.Stdout
	//csCmd.Stderr = os.Stderr
	//if err := csCmd.Start(); err != nil {
	//	log.Fatalf("Failed to start C# server: %v", err)
	//}
	//defer csCmd.Process.Kill()
	//log.Println("C# OCR server started successfully.")

	// 3. Juliaの計算サーバーを起動
	log.Println("Starting Julia calculation server...")
	jlCmd := exec.Command("julia", "julia_server.jl")
	jlCmd.Stdout = os.Stdout
	jlCmd.Stderr = os.Stderr
	if err := jlCmd.Start(); err != nil {
		log.Fatalf("Failed to start Julia server: %v", err)
	}
	defer jlCmd.Process.Kill()
	log.Println("Julia calculation server started successfully.")

	// 4. Pythonの音楽サーバーを起動
	log.Println("Starting Python music server...")
	musicCmd := exec.Command("python", "./music_player/player.py")
	musicCmd.Stdout = os.Stdout
	musicCmd.Stderr = os.Stderr
	if err := musicCmd.Start(); err != nil {
		log.Fatalf("Failed to start Python music server: %v", err)
	}
	defer musicCmd.Process.Kill()
	log.Println("Python music server started successfully.")

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
	//registerCommand(&commands.ConfigCommand{Store: dbStore})
	//registerCommand(&commands.DashboardCommand{Store: dbStore, Scheduler: scheduler})
	//registerCommand(&commands.ReactionRoleCommand{Store: dbStore})
	//registerCommand(&commands.ScheduleCommand{Scheduler: scheduler, Store: dbStore})
	//registerCommand(&commands.TicketCommand{Store: dbStore})
	registerCommand(&commands.PingCommand{StartTime: startTime, Store: dbStore})
	//registerCommand(&commands.AskCommand{})
	//registerCommand(&commands.AvatarCommand{})
	//registerCommand(&commands.CalculatorCommand{})
	//registerCommand(&commands.EmbedCommand{})
	//registerCommand(&commands.ModerateCommand{})
	//registerCommand(&commands.PokemonCalculatorCommand{})
	//registerCommand(&commands.PollCommand{})
	//registerCommand(&commands.PowerConverterCommand{})
	//registerCommand(&commands.TranslateCommand{})
	//registerCommand(&commands.UserInfoCommand{})
	//registerCommand(&commands.WeatherCommand{APIKey: os.Getenv("WEATHER_API_KEY")})
	//registerCommand(&commands.HelpCommand{AllCommands: commandHandlers})
	//registerCommand(&commands.ImagineCommand{})
	//registerCommand(&commands.MandelbrotCommand{})
	// registerCommand(&commands.ReadTextCommand{}) -- 無効化 --
	//registerCommand(&commands.MusicCommand{})

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

	guildID := "1385573037608271963"
	if _, err = dg.ApplicationCommandBulkOverwrite(dg.State.User.ID, guildID, registeredCommands); err != nil {
		logger.Fatal("コマンドの登録に失敗しました", "error", err, "guildID", guildID)
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
