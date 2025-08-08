package commands

import (
	"context"
	"fmt"
	"luna/ai"
	"luna/interfaces"

	"github.com/bwmarrin/discordgo"
)

// Pythonサーバーに送るテキスト生成リクエストの構造体
// type TextRequest struct {
// 	Prompt string `json:"prompt"`
// }

// Pythonサーバーから返ってくるテキスト生成レスポンスの構造体
// type TextResponse struct {
// 	Text  string `json:"text"`
// 	Error string `json:"error"`
// }

type AskCommand struct {
	Log interfaces.Logger
	AI  *ai.Client
}

func (c *AskCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "ask",
		Description: "Luna Assistantに質問します",
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionString, Name: "prompt", Description: "質問内容", Required: true},
		},
	}
}

// 内部の処理を、PythonサーバーへのHTTPリクエスト（ストリーミング対応）に変更
func (c *AskCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	prompt := i.ApplicationCommandData().Options[0].StringValue()

	// 「考え中...」と即時応答
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		c.Log.Error("Failed to send initial response", "error", err)
		return
	}

	// AIに役割を指示するシステムプロンプト（ペルソナ）を定義
	persona := "あなたは「Luna Assistant」という高性能で親切なAIアシスタントです。Googleによってトレーニングされた、という前置きは不要です。あなた自身の言葉で、ユーザーの質問に直接的かつ簡潔に回答してください。"

	// ユーザーの質問にペルソナを付け加える
	fullPrompt := fmt.Sprintf("システムインストラクション（あなたの役割）: %s\n\n[ユーザーからの質問]\n%s", persona, prompt)

	// AIクライアントを使用してテキストを生成
	responseText, err := c.AI.GenerateText(context.Background(), fullPrompt)

	// エラーハンドリング
	if err != nil {
		c.Log.Error("AIからの応答生成に失敗", "error", err)
		content := "エラー: AIからの応答の取得に失敗しました。"
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
			c.Log.Error("Failed to edit error response", "error", err)
		}
		return
	}

	// 最終的なメッセージを送信
	embed := &discordgo.MessageEmbed{
		Title:       "💬 Luna Assistantからの回答",
		Description: responseText,
		Color:       0x824ff1, // Gemini Purple
		Author: &discordgo.MessageEmbedAuthor{
			Name:    i.Member.User.String(),
			IconURL: i.Member.User.AvatarURL(""),
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Powered by Luna AI",
		},
	}

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	}); err != nil {
		c.Log.Error("Failed to edit final response", "error", err)
	}
}

func (c *AskCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *AskCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *AskCommand) GetComponentIDs() []string {
	return []string{}
}
func (c *AskCommand) GetCategory() string {
	return "AI"
}
