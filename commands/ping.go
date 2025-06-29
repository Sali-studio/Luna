package commands

import (
	"fmt" // 文字列フォーマットのため fmt パッケージをインポート
	"log"

	"github.com/bwmarrin/discordgo"
)

// コマンドの定義を格納するスライス
var Commands = []*discordgo.ApplicationCommand{
	{
		Name:        "ping",
		Description: "ボットのレイテンシを測定します", // 説明を分かりやすく変更
	},
	// ここに新しいコマンドの定義を追加していく
}

// コマンド名とハンドラ関数をマッピング
var CommandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"ping": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		log.Println("ping command received")

		// s.HeartbeatLatency() でレイテンシを取得し、ミリ秒単位の数値に変換
		latency := s.HeartbeatLatency().Milliseconds()

		// 応答メッセージを作成
		message := fmt.Sprintf("Pong! 🏓 (%dms)", latency)

		// 作成したメッセージで応答する
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: message,
			},
		})
		if err != nil {
			log.Printf("Error responding to ping command: %v", err)
		}
	},
	// ここに新しいコマンドのハンドラを追加していく
}
