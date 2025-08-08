package bot

import (
	"fmt"
	"luna/config"
	"luna/handlers/events"
	"luna/interfaces"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Bot は、Discordセッション、ロガー、データストアなど、ボットの主要なコンポーネントを保持します。
type Bot struct {
	session   *discordgo.Session
	log       interfaces.Logger
	db        interfaces.DataStore
	scheduler interfaces.Scheduler
	startTime time.Time
}

// New は、新しいBotインスタンスを初期化して返します。
func New(log interfaces.Logger, db interfaces.DataStore, scheduler interfaces.Scheduler) (*Bot, error) {
	session, err := discordgo.New("Bot " + config.Cfg.Discord.Token)
	if err != nil {
		return nil, fmt.Errorf("discordgoセッションの作成に失敗しました: %w", err)
	}

	return &Bot{
		session:   session,
		log:       log,
		db:        db,
		scheduler: scheduler,
		startTime: time.Now(),
	}, nil
}

// Start は、ボットを起動し、Discordに接続してイベントのリスニングを開始します。
func (b *Bot) Start(commandHandlers map[string]interfaces.CommandHandler, componentHandlers map[string]interfaces.CommandHandler, registeredCommands []*discordgo.ApplicationCommand) error {
	b.session.Identify.Intents = discordgo.IntentsAll

	// イベントハンドラを登録
	eventHandler := events.NewHandler(b.log, b.db, commandHandlers, componentHandlers)
	b.session.AddHandler(eventHandler.OnReady(registeredCommands))
	b.session.AddHandler(eventHandler.OnInteractionCreate)
	b.session.AddHandler(events.OnMessageCreate(b.db, b.log))
	b.session.AddHandler(events.OnMessageDelete(b.log))
	b.session.AddHandler(events.OnGuildMemberAdd(b.db, b.log))
	b.session.AddHandler(events.OnGuildMemberRemove(b.log))
	b.session.AddHandler(events.OnVoiceStateUpdate(b.log))
	b.session.AddHandler(events.OnChannelCreate(b.log))
	b.session.AddHandler(events.OnChannelDelete(b.log))
	b.session.AddHandler(events.OnGuildRoleCreate(b.log))
	b.session.AddHandler(events.OnGuildRoleDelete(b.log))

	if err := b.session.Open(); err != nil {
		return fmt.Errorf("Discordへの接続に失敗しました: %w", err)
	}

	b.log.Info("Botが正常に起動しました。Ctrl+Cで終了します。")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	return b.Close()
}

// Close は、ボットのすべてのコンポーネントを正常にシャットダウンします。
func (b *Bot) Close() error {
	b.log.Info("Botをシャットダウンしています...")
	b.scheduler.Stop()
	b.db.Close()
	return b.session.Close()
}

// GetSession は、現在のDiscordセッションを返します。
func (b *Bot) GetSession() *discordgo.Session {
	return b.session
}

// GetDBStore は、現在のデータストアを返します。
func (b *Bot) GetDBStore() interfaces.DataStore {
	return b.db
}

// GetScheduler は、現在のスケジューラを返します。
func (b *Bot) GetScheduler() interfaces.Scheduler {
	return b.scheduler
}

// GetStartTime は、ボットの起動時刻を返します。
func (b *Bot) GetStartTime() time.Time {
	return b.startTime
}