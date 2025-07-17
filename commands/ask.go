package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"luna/interfaces"

	"github.com/bwmarrin/discordgo"
)



type AskCommand struct {
	Log interfaces.Logger
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

// 内部の処理を、PythonサーバーへのHTTPリクエストに変更
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

	// Pythonサーバーに送信するデータを作成
	reqData := TextRequest{Prompt: fullPrompt} // 修正：ペルソナ付きのプロンプトを送信
	reqJson, _ := json.Marshal(reqData)

	// Pythonサーバーのテキスト生成エンドポイントにリクエストを送信
	resp, err := http.Post("http://localhost:5001/generate-text", "application/json", bytes.NewBuffer(reqJson))

	// エラーハンドリング
	if err != nil {
		c.Log.Error("AIサーバーへの接続に失敗", "error", err)
		content := "エラー: AIサーバーへの接続に失敗しました。"
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
			c.Log.Error("Failed to edit error response", "error", err)
		}
		return
	}
	defer resp.Body.Close()

	// レスポンスを読み取りJSONをパース
	body, _ := io.ReadAll(resp.Body)
	var textResp TextResponse
	if err := json.Unmarshal(body, &textResp); err != nil {
		c.Log.Error("Failed to unmarshal AI response", "error", err)
		return
	}

	if textResp.Error != "" || resp.StatusCode != http.StatusOK {
		c.Log.Error("Luna Assistantからの応答取得に失敗", "error", textResp.Error, "status_code", resp.StatusCode)
		content := fmt.Sprintf("エラー: Luna Assistantからの応答取得に失敗しました。\n`%s`", textResp.Error)
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
			c.Log.Error("Failed to edit error response", "error", err)
		}
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "💬 Luna Assistantからの回答",
		Description: textResp.Text,
		Color:       0x824ff1, // Gemini Purple
		Author: &discordgo.MessageEmbedAuthor{
			Name:    i.Member.User.String(),
			IconURL: i.Member.User.AvatarURL(""),
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Powered by Luna",
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
