package bot

import (
	"os/signal"
	"strings"
	"syscall"
	"time"

	"luna/commands"
	"luna/config"
	"luna/handlers"
	"luna/logger"
	"luna/storage"

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
)

// Bot はDiscordボットのコアな状態とロジックを管理します。
type Bot struct {
	Session           *discordgo.Session
	dbStore           *storage.DBStore
	scheduler         *cron.Cron
	commandHandlers   map[string]commands.CommandHandler
	componentHandlers map[string]commands.CommandHandler
	startTime         time.Time
}

// New は新しいBotインスタンスを作成します。
func New() (*Bot, error) {
	token := config.Cfg.Discord.Token
	if token == "" || token == "YOUR_DISCORD_BOT_TOKEN_HERE" {
		logger.Fatal("DiscordのBotトークンが設定されていません。config.yamlを確認してください。")
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	dg.State = discordgo.NewState()
	dg.State.MaxMessageCount = 2000
	dg.Identify.Intents = discordgo.IntentsAll

	dbStore, err := storage.NewDBStore("./luna.db")
	if err != nil {
		return nil, err
	}

	return &Bot{
		Session:           dg,
		dbStore:           dbStore,
		scheduler:         cron.New(),
		commandHandlers:   make(map[string]commands.CommandHandler),
		componentHandlers: make(map[string]commands.CommandHandler),
		startTime:         time.Now(),
	}, nil
}

// Start はBotを起動し、Discordに接続します。
func (b *Bot) Start() error {
	eventHandler := handlers.NewEventHandler(b.dbStore)
	eventHandler.RegisterAllHandlers(b.Session)

	b.registerCommands()

	b.Session.AddHandler(b.interactionCreate)

	if err := b.Session.Open(); err != nil {
		return err
	}
	defer b.Session.Close()
	defer b.dbStore.Close()

	b.scheduler.Start()
	defer b.scheduler.Stop()

	if scheduleCmd, ok := b.commandHandlers["schedule"].(*commands.ScheduleCommand); ok {
		scheduleCmd.LoadAndRegisterSchedules(b.Session)
	}

	logger.Info("Discord Botが起動しました。コマンドを登録します...")
	registeredCommands := make([]*discordgo.ApplicationCommand, 0, len(b.commandHandlers))
	for _, handler := range b.commandHandlers {
		registeredCommands = append(registeredCommands, handler.GetCommandDef())
	}

	if _, err := b.Session.ApplicationCommandBulkOverwrite(b.Session.State.User.ID, "", registeredCommands); err != nil {
		logger.Fatal("コマンドの登録に失敗しました", "error", err)
	}

	logger.Info("コマンドの登録が完了しました。Ctrl+Cで終了します。")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	logger.Info("Botをシャットダウンします...")
	return nil
}

func (b *Bot) registerCommands() {
	appContext := &commands.AppContext{
		Store:     b.dbStore,
		Scheduler: b.scheduler,
		StartTime: b.startTime,
	}
	for _, cmd := range commands.RegisterAllCommands(appContext, b.commandHandlers) {
		b.registerCommand(cmd)
	}
}

func (b *Bot) registerCommand(cmd commands.CommandHandler) {
	def := cmd.GetCommandDef()
	b.commandHandlers[def.Name] = cmd
	for _, id := range cmd.GetComponentIDs() {
		b.componentHandlers[id] = cmd
	}
}

func (b *Bot) interactionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		if h, ok := b.commandHandlers[i.ApplicationCommandData().Name]; ok {
			h.Handle(s, i)
		}
	case discordgo.InteractionMessageComponent:
		for id, h := range b.componentHandlers {
			if strings.HasPrefix(i.MessageComponentData().CustomID, id) {
				h.HandleComponent(s, i)
				return
			}
		}
	case discordgo.InteractionModalSubmit:
		for id, h := range b.componentHandlers {
			if strings.HasPrefix(i.ModalSubmitData().CustomID, id) {
				h.HandleModal(s, i)
				return
			}
		}
	}
}
