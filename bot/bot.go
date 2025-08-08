package bot

import (
	"fmt"
	"luna/config"
	"luna/handlers"
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
	h := handlers.NewEventHandler(b.log, b.db, commandHandlers, componentHandlers)
	b.session.AddHandler(h.OnReady)
	b.session.AddHandler(h.OnInteractionCreate)
	b.session.AddHandler(h.OnMessageCreate)
	b.session.AddHandler(h.OnMessageDelete)
	b.session.AddHandler(h.OnGuildMemberAdd)
	b.session.AddHandler(h.OnGuildMemberRemove)
	b.session.AddHandler(h.OnVoiceStateUpdate)
	b.session.AddHandler(h.OnChannelCreate)
	b.session.AddHandler(h.OnChannelDelete)
	b.session.AddHandler(h.OnGuildRoleCreate)
	b.session.AddHandler(h.OnGuildRoleDelete)

	if err := b.session.Open(); err != nil {
		return fmt.Errorf("Discordへの接続に失敗しました: %w", err)
	}

	// グローバルコマンドの登録
	// _, err := b.session.ApplicationCommandBulkOverwrite(b.session.State.User.ID, "", registeredCommands)
	// if err != nil {
	// 	b.log.Fatal("コマンドの登録に失敗しました", "error", err)
	// }

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
