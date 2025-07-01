package commands

import (
	"fmt"
	"luna/logger"

	"github.com/bwmarrin/discordgo"
)

type PingCommand struct{}

func (c *PingCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "ping",
		Description: "ボットのレイテンシを測定します",
	}
}

func (c *PingCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	latency := s.HeartbeatLatency().Milliseconds()
	message := fmt.Sprintf("Pong! 🏓 (%dms)", latency)
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
		},
	})
	if err != nil {
		logger.Error.Printf("pingコマンドへの応答中にエラー: %v", err)
	}
}

func (c *PingCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// このコマンドにはComponentはありません
}

func (c *PingCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// このコマンドにはModalはありません
}
