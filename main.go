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
	if token == "" {
		logger.Fatal.Println("環境変数 DISCORD_BOT_TOKEN が設定されていません。")
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		logger.Fatal.Printf("Discordセッションの作成中にエラーが発生しました: %v", err)
	}

	// main.go の中の dg.AddHandler(...) の部分
	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			if h, ok := commands.CommandHandlers[i.ApplicationCommandData().Name]; ok {
				h(s, i)
			}
		case discordgo.InteractionMessageComponent:
			// 押されたボタンのIDによって処理を振り分ける
			customID := i.MessageComponentData().CustomID
			switch customID {
			case "create_ticket_button":
				commands.HandleTicketCreation(s, i)
			case "close_ticket_button":
				commands.HandleTicketClose(s, i)
			}
		}
	})

	err = dg.Open()
	if err != nil {
		logger.Fatal.Printf("Discordへの接続中にエラーが発生しました: %v", err)
	}
	defer dg.Close()

	logger.Info.Println("Botが起動しました。スラッシュコマンドを登録します。")

	registeredCommands, err := dg.ApplicationCommandBulkOverwrite(dg.State.User.ID, "", commands.Commands)
	if err != nil {
		logger.Fatal.Printf("コマンドの登録に失敗しました: %v", err)
	}

	logger.Info.Println("Bot is now running. Press CTRL-C to exit.")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	logger.Info.Println("Botをシャットダウンします。")

	for _, cmd := range registeredCommands {
		err := dg.ApplicationCommandDelete(dg.State.User.ID, "", cmd.ID)
		if err != nil {
			logger.Error.Printf("コマンドの削除に失敗しました: %v", err)
		}
	}

	logger.Info.Println("コマンドを削除しました。")
}
