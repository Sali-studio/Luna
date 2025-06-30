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

	dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildMembers | discordgo.IntentsGuildVoiceStates | discordgo.IntentGuildModeration

	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			if h, ok := commands.CommandHandlers[i.ApplicationCommandData().Name]; ok {
				h(s, i)
			}
		case discordgo.InteractionMessageComponent:
			customID := i.MessageComponentData().CustomID
			switch customID {
			// チケット作成フロー
			case "open_ticket_modal":
				commands.HandleOpenTicketModal(s, i)
			case "close_ticket_button":
				commands.HandleTicketClose(s, i)
			// ★★★ 設定ダッシュボードのボタン処理を追加 ★★★
			case "config_ticket_button":
				commands.HandleShowTicketConfigModal(s, i)
			case "config_log_button":
				commands.HandleShowLogConfigModal(s, i)
			case "config_temp_vc_button":
				commands.HandleShowTempVCConfigModal(s, i)
			}
		case discordgo.InteractionModalSubmit:
			customID := i.ModalSubmitData().CustomID
			parts := strings.Split(customID, ":")
			modalType := parts[0]

			switch modalType {
			// チケット・Embed作成フロー
			case "ticket_creation_modal":
				commands.HandleTicketCreation(s, i)
			case "embed_creation_modal":
				commands.HandleEmbedCreation(s, i)
			// モデレーション確認フロー
			case "moderate_kick_confirm":
				commands.HandleExecuteKick(s, i, parts)
			case "moderate_ban_confirm":
				commands.HandleExecuteBan(s, i, parts)
			case "moderate_timeout_confirm":
				commands.HandleExecuteTimeout(s, i, parts)
			// ★★★ 設定モーダルの保存処理を追加 ★★★
			case "config_ticket_modal":
				commands.HandleSaveTicketConfig(s, i)
			case "config_log_modal":
				commands.HandleSaveLogConfig(s, i)
			case "config_temp_vc_modal":
				commands.HandleSaveTempVCConfig(s, i)
			}
		}
	})

	// (他のイベントハンドラは変更なし)
	dg.AddHandler(commands.HandleGuildBanAdd)
	// ...

	err = dg.Open()
	// (以降のコードも変更なし)
	// ...
}
