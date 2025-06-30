package main

import (
	"os"
	"strings"

	"luna/commands"
	"luna/logger"

	"github.com/bwmarrin/discordgo"
)

func main() {
	logger.Init()
	token := os.Getenv("DISCORD_BOT_TOKEN")
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		logger.Fatal.Printf("Discordセッションの作成中にエラーが発生しました: %v", err)
	}

	dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMembers | discordgo.IntentsGuildVoiceStates | discordgo.IntentGuildModeration

	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			if h, ok := commands.CommandHandlers[i.ApplicationCommandData().Name]; ok {
				h(s, i)
			}
		case discordgo.InteractionMessageComponent:
			customID := i.MessageComponentData().CustomID
			switch customID {
			case "open_ticket_modal":
				commands.HandleOpenTicketModal(s, i)
			case "close_ticket_button":
				commands.HandleTicketClose(s, i)
			}
		case discordgo.InteractionModalSubmit:
			customID := i.ModalSubmitData().CustomID
			// モーダルのIDを':'で分割して、どの処理か判断する
			parts := strings.Split(customID, ":")
			if len(parts) < 1 {
				return
			}
			modalType := parts[0]

			switch modalType {
			case "ticket_creation_modal":
				commands.HandleTicketCreation(s, i)
			case "embed_creation_modal":
				commands.HandleEmbedCreation(s, i)
			// --- モデレーションの確認モーダル処理を追加 ---
			case "moderate_kick_confirm":
				commands.HandleExecuteKick(s, i, parts)
			case "moderate_ban_confirm":
				commands.HandleExecuteBan(s, i, parts)
			case "moderate_timeout_confirm":
				commands.HandleExecuteTimeout(s, i, parts)
			}
		}
	})

	// (ロギング用ハンドラなどは変更なし)
	// ...
}
