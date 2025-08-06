package bot

import (
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"luna/config"
	"luna/handlers/events"
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
	player    interfaces.MusicPlayer
}

// New は新しいBotインスタンスを作成します。
func New(log interfaces.Logger, db interfaces.DataStore, scheduler interfaces.Scheduler, player interfaces.MusicPlayer) (*Bot, error) {
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
		player:    player,
	}, nil
}

// Start はBotを起動し、Discordに接続します。
func (b *Bot) Start(commandHandlers map[string]interfaces.CommandHandler, componentHandlers map[string]interfaces.CommandHandler, registeredCommands []*discordgo.ApplicationCommand) error {
	// ボット起動時に一度だけ実行されるハンドラ
	b.Session.AddHandlerOnce(func(s *discordgo.Session, r *discordgo.Ready) {
		b.log.Info("Bot is ready. Clearing old global commands...")
		// 古いグローバルコマンドをクリア
		if _, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, "", []*discordgo.ApplicationCommand{}); err != nil {
			b.log.Error("Failed to clear global commands", "error", err)
		}
		b.log.Info("Global commands cleared.")
	})

	// ギルドが利用可能になった（起動時や新規参加時）ときにコマンドを登録するハンドラ
	b.Session.AddHandler(func(s *discordgo.Session, g *discordgo.GuildCreate) {
		b.log.Info("Registering commands for guild", "guild_id", g.ID, "guild_name", g.Name)
		if _, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, g.ID, registeredCommands); err != nil {
			b.log.Error("Failed to register commands for guild", "guild_id", g.ID, "error", err)
		}
	})

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

	// イベントハンドラの登録
	events.NewMemberEventHandler(b.dbStore, b.log).RegisterHandlers(b.Session)

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

func (b *Bot) GetPlayer() interfaces.MusicPlayer {
	return b.player
}