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

	dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildMembers | discordgo.IntentsGuildVoiceStates | discordgo.IntentGuildModeration

	// --- イベントハンドラ ---
	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			if h, ok := commands.CommandHandlers[i.ApplicationCommandData().Name]; ok {
				h(s, i)
			}
		case discordgo.InteractionMessageComponent:
			// ... (ボタン処理は変更なし)
		case discordgo.InteractionModalSubmit:
			// ... (モーダル処理は変更なし)
		}
	})

	// --- 各機能のイベントハンドラを登録 ---
	// ★★★ すべてのハンドラ名を大文字始まりに統一 ★★★
	dg.AddHandler(commands.HandleGuildBanAdd)
	dg.AddHandler(commands.HandleGuildMemberRemove)
	dg.AddHandler(commands.HandleGuildMemberUpdate)
	dg.AddHandler(commands.HandleChannelCreate)
	dg.AddHandler(commands.HandleChannelDelete)
	dg.AddHandler(commands.HandleMessageDelete)
	dg.AddHandler(commands.HandleWebhooksUpdate)
	dg.AddHandler(commands.HandleGuildMemberAddLog)
	dg.AddHandler(commands.HandleMessageReactionAdd)
	dg.AddHandler(commands.HandleMessageReactionRemove)
	dg.AddHandler(commands.HandleVoiceStateUpdate)
	dg.AddHandler(commands.HandleMessageCreate)

	err = dg.Open()
	if err != nil {
		logger.Fatal.Printf("Discordへの接続中にエラーが発生しました: %v", err)
	}
	defer dg.Close()

	commands.StartDashboardUpdater(dg)

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
