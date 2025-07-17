// commands/describe_image.go
package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"luna/interfaces"
	"net/http"

	"github.com/bwmarrin/discordgo"
)

// Pythonサーバーに送る画像認識リクエストの構造体
type DescribeImageRequest struct {
	ImageURL string `json:"image_url"`
	Prompt   string `json:"prompt"`
}

type DescribeImageCommand struct {
	Log interfaces.Logger
}

func (c *DescribeImageCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "describe-image",
		Description: "AIが画像を説明します",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionAttachment,
				Name:        "image",
				Description: "説明してほしい画像",
				Required:    true,
			},
		},
	}
}

func (c *DescribeImageCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// オプションから画像を取得
	attachmentID := i.ApplicationCommandData().Options[0].Value.(string)
	attachment := i.ApplicationCommandData().Resolved.Attachments[attachmentID]
	imageURL := attachment.URL

	// AIに画像を説明させる
	SendDescribeRequest(s, i, imageURL, c.Log)
}

// SendDescribeRequest は画像URLを受け取り、AIサーバーに説明をリクエストして結果をDiscordに送信します。
// この関数はコンテキストメニューコマンドからも利用されます。
func SendDescribeRequest(s *discordgo.Session, i *discordgo.InteractionCreate, imageURL string, log interfaces.Logger) {
	// AIに渡すプロンプトを定義（文字起こしメイン）
	prompt := "この画像に文字が書かれている場合は、その内容を正確に書き出してください。文字がない、または読み取れない場合は、画像の内容を簡潔に説明してください。"

	// 「考え中...」と即時応答
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		log.Error("Failed to send initial response", "error", err)
		return
	}

	// Pythonサーバーに送信するデータを作成
	reqData := DescribeImageRequest{ImageURL: imageURL, Prompt: prompt} // プロンプトを追加
	reqJson, _ := json.Marshal(reqData)

	// Pythonサーバーの画像認識エンドポイントにリクエストを送信
	resp, err := http.Post("http://localhost:5001/describe-image", "application/json", bytes.NewBuffer(reqJson))

	// エラーハンドリング
	if err != nil {
		log.Error("AIサーバーへの接続に失敗", "error", err)
		content := "エラー: AIサーバーへの接続に失敗しました。"
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
			log.Error("Failed to edit error response", "error", err)
		}
		return
	}
	defer resp.Body.Close()

	// レスポンスを読み取りJSONをパース (TextResponseを再利用)
	body, _ := io.ReadAll(resp.Body)
	var textResp TextResponse
	if err := json.Unmarshal(body, &textResp); err != nil {
		log.Error("Failed to unmarshal AI response", "error", err)
		return
	}

	if textResp.Error != "" || resp.StatusCode != http.StatusOK {
		log.Error("Luna Assistantからの応答取得に失敗", "error", textResp.Error, "status_code", resp.StatusCode)
		content := fmt.Sprintf("エラー: Luna Assistantからの応答取得に失敗しました。\n`%s`", textResp.Error)
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
			log.Error("Failed to edit error response", "error", err)
		}
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🖼️ 画像の説明",
		Description: textResp.Text,
		Color:       0x824ff1, // Gemini Purple
		Author: &discordgo.MessageEmbedAuthor{
			Name:    i.Member.User.String(),
			IconURL: i.Member.User.AvatarURL(""),
		},
		Image: &discordgo.MessageEmbedImage{
			URL: imageURL,
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Powered by Gemini",
		},
	}

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	}); err != nil {
		log.Error("Failed to edit final response", "error", err)
	}
}

func (c *DescribeImageCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *DescribeImageCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *DescribeImageCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *DescribeImageCommand) GetCategory() string                                                  { return "AI" }