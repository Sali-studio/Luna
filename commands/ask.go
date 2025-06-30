package commands

import (
	"fmt"
	"luna/gemini" // 作成したgeminiパッケージをインポート
	"luna/logger"
	"os"

	"github.com/bwmarrin/discordgo"
)

func init() {
	cmd := &discordgo.ApplicationCommand{
		Name:        "ask",
		Description: "AIに質問します。",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "question",
				Description: "AIへの質問内容",
				Required:    true,
			},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		logger.Info.Println("ask command received")

		// APIからの応答には時間がかかる可能性があるため、先に「考え中...」と応答する
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
		if err != nil {
			logger.Error.Printf("Failed to defer interaction: %v", err)
			return
		}

		// 環境変数からAPIキーを取得
		apiKey := os.Getenv("GEMINI_API_KEY")
		if apiKey == "" {
			content := "❌ エラー: `GEMINI_API_KEY`が設定されていません。"
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &content,
			})
			return
		}

		// 質問内容を取得
		question := i.ApplicationCommandData().Options[0].StringValue()
		
		// Geminiクライアントを呼び出して、応答を生成
		response, err := gemini.GenerateContent(apiKey, question)
		if err != nil {
			logger.Error.Printf("Failed to generate content from Gemini: %v", err)
			content := fmt.Sprintf("❌ AIからの応答の取得中にエラーが発生しました。\n`%v`", err)
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &content,
			})
			return
		}

		// AIからの応答をEmbedに整形して表示
		embed := &discordgo.MessageEmbed{
			Author: &discordgo.MessageEmbedAuthor{
				Name: i.Member.User.Username,
				IconURL: i.Member.User.AvatarURL(""),
			},
			Description: question, // ユーザーの質問
			Color: 0x4A90E2,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name: "🤖 Luna (Gemini)",
					Value: response,
				},
			},
		}
		
		// 最初に返信したメッセージを、AIの応答で編集する
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
		})
	}

	Commands = append(Commands, cmd)
	CommandHandlers[cmd.Name] = handler
}