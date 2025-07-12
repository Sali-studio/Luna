package interfaces

import (
	"context"
	"luna/storage"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
)

// Logger は、アプリケーション全体で使用されるロガーのインターフェースを定義します。
type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Fatal(msg string, args ...any)
}

// DataStore は、ボットが依存するデータベース操作のインターフェースを定義します。
type DataStore interface {
	Close()
	PingDB() error
	GetConfig(guildID, configName string, configStruct interface{}) error
	SaveConfig(guildID, configName string, configStruct interface{}) error
	GetNextTicketCounter(guildID string) (int, error)
	CreateTicketRecord(channelID, guildID, userID string) error
	CloseTicketRecord(channelID string) error
	GetReactionRole(guildID, messageID, emojiID string) (storage.ReactionRole, error)
	SaveReactionRole(rr storage.ReactionRole) error
	SaveSchedule(schedule storage.Schedule) error
	GetAllSchedules() ([]storage.Schedule, error)
	CreateMessageCache(messageID, content, authorID string) error
	GetMessageCache(messageID string) (*storage.CachedMessage, error)
}

// Scheduler は、タスクのスケジューリング機能のインターフェースを定義します。
type Scheduler interface {
	Start()
	Stop() context.Context
	AddFunc(spec string, cmd func()) (cron.EntryID, error)
}

// CommandHandler は、すべてのボットコマンドが実装すべきインターフェースを定義します。
type CommandHandler interface {
	GetCommandDef() *discordgo.ApplicationCommand
	Handle(s *discordgo.Session, i *discordgo.InteractionCreate)
	HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate)
	HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)
	GetComponentIDs() []string
	GetCategory() string
}
