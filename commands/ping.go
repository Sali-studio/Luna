package commands

import (
	"fmt"
	"luna/logger"

	"github.com/bwmarrin/discordgo"
)

// ★★★ このファイル全体をinit()で囲むように変更 ★★★
func init() {
	cmd := &discordgo.ApplicationCommand{
		Name:        "ping",
		Description: "ボットのレイテンシを測定します",
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		logger.Info.Println("ping command received")

		latency := s.HeartbeatLatency().Milliseconds()
		message := fmt.Sprintf("Pong! 🏓 (%dms)", latency)

		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: message,
			},
		})
		if err != nil {
			logger.Error.Printf("Error responding to ping command: %v", err)
		}
	}

	Commands = append(Commands, cmd)
	CommandHandlers[cmd.Name] = handler
}
