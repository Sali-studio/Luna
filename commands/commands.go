package commands

import (
	"luna/logger"
	"luna/storage"

	"github.com/bwmarrin/discordgo"
)

var (
	Commands        = []*discordgo.ApplicationCommand{}
	CommandHandlers = make(map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate))
	Config          *storage.ConfigStore
)

func init() {
	var err error
	Config, err = storage.NewConfigStore("config.json")
	if err != nil {
		logger.Fatal.Fatalf("Failed to initialize config store: %v", err)
	}
}
func int64Ptr(i int64) *int64 {
	return &i
}
