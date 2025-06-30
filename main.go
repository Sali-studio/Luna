package main

import (
	"os"
	"os/signal"
	"syscall"

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

	// --- イベントハンドラ ---
	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		// スラッシュコマンド
		case discordgo.InteractionApplicationCommand:
			if h, ok := commands.CommandHandlers[i.ApplicationCommandData().Name]; ok {
				h(s, i)
			}
		// ボタンなどのコンポーネント
		case discordgo.InteractionMessageComponent:
			customID := i.MessageComponentData().CustomID
			switch customID {
			case "open_ticket_modal":
				commands.HandleOpenTicketModal(s, i)
			case "close_ticket_button":
				commands.HandleTicketClose(s, i)
			}
		// モーダル送信時の処理
		case discordgo.InteractionModalSubmit:
			if i.ModalSubmitData().CustomID == "ticket_creation_modal" {
				commands.HandleTicketCreation(s, i)
			}
		}
	})

	// --- ロギング用ハンドラ ---
	dg.AddHandler(commands.HandleGuildBanAdd)
	dg.AddHandler(commands.HandleGuildMemberRemove)
	dg.AddHandler(commands.HandleGuildMemberUpdate)
	dg.AddHandler(commands.HandleChannelCreate)
	dg.AddHandler(commands.HandleChannelDelete)
	dg.AddHandler(commands.HandleMessageDelete)
	dg.AddHandler(commands.HandleWebhooksUpdate)
	dg.AddHandler(commands.HandleGuildMemberAddLog)

	err = dg.Open()
	if err != nil {
		logger.Fatal.Printf("Discordへの接続中にエラーが発生しました: %v", err)
	}
	defer dg.Close()

	logger.Info.Println("Botが起動しました。スラッシュコマンドを登録します。")
	_, err = dg.ApplicationCommandBulkOverwrite(dg.State.User.ID, "", commands.Commands)
	if err != nil {
		logger.Fatal.Printf("コマンドの登録に失敗しました: %v", err)
	}

	logger.Info.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	logger.Info.Println("Botをシャットダウンします。")
}
