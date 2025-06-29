package commands

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

// コマンドの定義を格納するスライス
var Commands = []*discordgo.ApplicationCommand{
	{
		Name:        "ping",
		Description: "Pong! と返します",
	},
	// ここに新しいコマンドの定義を追加していく
}

// コマンド名とハンドラ関数をマッピング
var CommandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"ping": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		log.Println("ping command received")
		// Pong!というメッセージで応答する
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Pong! 🏓",
			},
		})
		if err != nil {
			log.Printf("Error responding to ping command: %v", err)
		}
	},
	// ここに新しいコマンドのハンドラを追加していく
}
