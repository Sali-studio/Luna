package commands

import (
	"luna/gemini" // geminiパッケージをインポート
	"luna/logger"

	"github.com/bwmarrin/discordgo"
)

type AskCommand struct {
	// 依存としてGeminiクライアントを受け取る
	Gemini *gemini.Client
}

func (c *AskCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "ask",
		Description: "AI(Gemini)に質問します",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "prompt",
				Description: "AIへの質問内容",
				Required:    true,
			},
		},
	}
}

func (c *AskCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Geminiクライアントが正しく設定されているか確認
	if c.Gemini == nil {
		logger.Error.Println("Geminiクライアントがaskコマンドに設定されていません。")
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ このコマンドは現在利用できません。管理者に連絡してください。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	prompt := i.ApplicationCommandData().Options[0].StringValue()

	// 「考え中...」と先に返信
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		logger.Error.Printf("askコマンドの応答(defer)に失敗: %v", err)
		return
	}

	// Geminiクライアントを使ってコンテンツを生成
	responseContent, err := c.Gemini.GenerateContent(prompt)
	if err != nil {
		logger.Error.Printf("Geminiからの応答取得に失敗: %v", err)
		content := "❌ AIへの接続または応答の取得中にエラーが発生しました。"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
		return
	}

	// 応答を編集して送信
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &responseContent,
	})
}

func (c *AskCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *AskCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
