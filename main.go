package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"luna/commands" // commandsパッケージをインポート

	"github.com/bwmarrin/discordgo"
)

func main() {
	// 環境変数からBotトークンを読み込む
	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		log.Fatal("環境変数 DISCORD_BOT_TOKEN が設定されていません。")
	}

	// Discordセッションを作成
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalf("Discordセッションの作成中にエラーが発生しました: %v", err)
	}

	// Interaction (スラッシュコマンドなど) が発生したときのハンドラを追加
	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		// commandsパッケージに定義されたコマンド名と一致するか確認
		if h, ok := commands.CommandHandlers[i.ApplicationCommandData().Name]; ok {
			// 一致したら、そのコマンドのハンドラを実行
			h(s, i)
		}
	})

	// DiscordへのWebSocket接続を開く
	err = dg.Open()
	if err != nil {
		log.Fatalf("Discordへの接続中にエラーが発生しました: %v", err)
	}
	defer dg.Close() // プログラム終了時に接続を閉じる

	log.Println("Botが起動しました。スラッシュコマンドを登録します。")

	// コマンドを一括で登録
	// 登録済みのコマンドは上書きされる
	registeredCommands, err := dg.ApplicationCommandBulkOverwrite(dg.State.User.ID, "", commands.Commands)
	if err != nil {
		log.Fatalf("コマンドの登録に失敗しました: %v", err)
	}

	log.Println("Bot is now running. Press CTRL-C to exit.")

	// Botが終了シグナル (Ctrl+Cなど) を受け取るまで待機
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	log.Println("Botをシャットダウンします。")

	// 登録したコマンドを削除
	for _, cmd := range registeredCommands {
		err := dg.ApplicationCommandDelete(dg.State.User.ID, "", cmd.ID)
		if err != nil {
			log.Printf("コマンドの削除に失敗しました: %v", err)
		}
	}

	log.Println("コマンドを削除しました。")
}
