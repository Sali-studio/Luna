package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

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

	// Pythonサーバーに送信するデータを作成
	reqData := TextRequest{Prompt: fullPrompt}
	reqJson, _ := json.Marshal(reqData)

	// Pythonサーバーのストリーミングエンドポイントにリクエストを送信
	resp, err := http.Post("http://localhost:5001/generate-text-stream", "application/json", bytes.NewBuffer(reqJson))

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

	// ストリーミングでレスポンスを処理
	var responseText strings.Builder
	var lastUpdateTime time.Time
	buffer := make([]byte, 1024) // チャンクを読み込むためのバッファ

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			responseText.WriteString(string(buffer[:n]))

			// Discord APIのレート制限を避けるため、一定間隔でメッセージを更新
			if time.Since(lastUpdateTime) > 1500*time.Millisecond {
				embed := &discordgo.MessageEmbed{
					Title:       "💬 Luna Assistantからの回答",
					Description: responseText.String() + "...", // 生成中であることを示す
					Color:       0x824ff1, // Gemini Purple
					Author: &discordgo.MessageEmbedAuthor{
						Name:    i.Member.User.String(),
						IconURL: i.Member.User.AvatarURL(""),
					},
					Footer: &discordgo.MessageEmbedFooter{
						Text: "Powered by Luna | 生成中...",
					},
				}
				if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Embeds: &[]*discordgo.MessageEmbed{embed},
				}); err != nil {
					c.Log.Error("Failed to edit streaming response", "error", err)
				}
				lastUpdateTime = time.Now()
			}
		}
		if err == io.EOF {
			break // ストリームの終端
		}
		if err != nil {
			c.Log.Error("ストリームの読み込みに失敗", "error", err)
			break
		}
	}

	// 最終的なメッセージを送信
	embed := &discordgo.MessageEmbed{
		Title:       "💬 Luna Assistantからの回答",
		Description: responseText.String(),
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
