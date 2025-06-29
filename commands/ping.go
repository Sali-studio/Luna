package commands

import (
	"fmt"
	"luna/logger" // 新しいloggerパッケージをインポート

	"github.com/bwmarrin/discordgo"
)

var Commands = []*discordgo.ApplicationCommand{
	{
		Name:        "ping",
		Description: "ボットのレイテンシを測定します",
	},
}

var CommandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"ping": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		logger.Info.Println("ping command received") // log.Println を logger.Info.Println に変更

		latency := s.HeartbeatLatency().Milliseconds()
		message := fmt.Sprintf("Pong! 🏓 (%dms)", latency)

		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: message,
			},
		})
		if err != nil {
			logger.Error.Printf("Error responding to ping command: %v", err) // log.Printf を logger.Error.Printf に変更
		}
	},
}
