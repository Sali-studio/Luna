package interfaces

import (
	"context"
	"luna/storage"

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
	CreateMessageCache(messageID, content, authorID string) error
	GetMessageCache(messageID string) (*storage.CachedMessage, error)
	SaveQuizQuestion(guildID, topic, question string) error
	GetRecentQuizQuestions(guildID, topic string, limit int) ([]string, error)
	IncrementWordCount(guildID, userID, word string) error
	GetWordCount(guildID, userID, word string) (int, error)
	GetWordCountRanking(guildID, word string, limit int) ([]storage.WordCount, error)
	AddCountableWord(guildID, word string) error
	RemoveCountableWord(guildID, word string) error
	GetCountableWords(guildID string) ([]string, error)
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

// MusicPlayer は音楽再生機能のインターフェースを定義します。
type MusicPlayer interface {
	JoinVC(guildID, channelID string) error
	LeaveVC(guildID string)
	Play(guildID string, url, title, author string) error // Song構造体ではなく、必要なフィールドを直接渡す
	Stop(guildID string)
	Skip(guildID string)
	GetQueue(guildID string) []struct{ URL, Title, Author string } // Song構造体ではなく、匿名構造体で返す
	NowPlaying(guildID string) *struct{ URL, Title, Author string } // Song構造体ではなく、匿名構造体で返す
	GetGuildPlayer(guildID string) interface{} // player.GuildPlayerの代わりにinterface{}を返す
	GetAudioStreamURL(url string) (streamURL, title, author string, err error)
}
