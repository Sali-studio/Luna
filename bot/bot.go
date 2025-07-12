package bot

import (
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"luna/config"
	"luna/handlers"
	"luna/interfaces"

	"github.com/bwmarrin/discordgo"
)

// Bot はDiscordボットのコアな状態とロジックを管理します。
type Bot struct {
	Session   *discordgo.Session
	log       interfaces.Logger
	dbStore   interfaces.DataStore
	scheduler interfaces.Scheduler
	startTime time.Time
}

// New は新しいBotインスタンスを作成します。
func New(log interfaces.Logger, db interfaces.DataStore, scheduler interfaces.Scheduler) (*Bot, error) {
	token := config.Cfg.Discord.Token
	if token == "" || token == "YOUR_DISCORD_BOT_TOKEN_HERE" {
		log.Fatal("DiscordのBotトークンが設定されていません。config.yamlを確認してください。")
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	dg.State = discordgo.NewState()
	dg.State.MaxMessageCount = 2000
	dg.Identify.Intents = discordgo.IntentsAll

	return &Bot{
		Session:   dg,
		log:       log,
		dbStore:   db,
		scheduler: scheduler,
		startTime: time.Now(),
	}, nil
}

// Start はBotを起動し、Discordに接続します。
func (b *Bot) Start(commandHandlers map[string]interfaces.CommandHandler, componentHandlers map[string]interfaces.CommandHandler) error {
	eventHandler := handlers.NewEventHandler(b.dbStore, b.log)
	eventHandler.RegisterAllHandlers(b.Session)

	b.Session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
	})

	if err := b.Session.Open(); err != nil {
		return err
	}
	defer b.Session.Close()
	defer b.dbStore.Close()

	b.scheduler.Start()
	defer b.scheduler.Stop()

	b.log.Info("Botが起動しました。Ctrl+Cで終了します。")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	b.log.Info("Botをシャットダウンします...")
	return nil
}

func (b *Bot) GetDBStore() interfaces.DataStore {
	return b.dbStore
}

func (b *Bot) GetScheduler() interfaces.Scheduler {
	return b.scheduler
}

func (b *Bot) GetStartTime() time.Time {
	return b.startTime
}

func (b *Bot) GetSession() *discordgo.Session {
	return b.Session
}
