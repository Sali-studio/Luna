package commands

import (
	"luna/logger"
	"luna/storage"

	"github.com/bwmarrin/discordgo"
)

var (
	Commands        = []*discordgo.ApplicationCommand{}
	CommandHandlers = make(map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate))
	// ★★★ すべての設定を管理するグローバルなストア ★★★
	Config *storage.ConfigStore
)

func init() {
	var err error
	// ボットのルートに "config.json" という名前で設定ファイルが作成される
	Config, err = storage.NewConfigStore("config.json")
	if err != nil {
		logger.Fatal.Fatalf("Failed to initialize config store: %v", err)
	}
}

func int64Ptr(i int64) *int64 {
	return &i
}
