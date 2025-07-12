package bot

import (
	"context"
	"luna/storage"

	"github.com/robfig/cron/v3"
)

// DataStore は、ボットが依存するデータベース操作のインターフェースを定義します。
// これにより、ボットと具体的なデータベース実装との結合を疎にし、
// テストやメンテナンスを容易にすることができます。
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
