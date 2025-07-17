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
		Name: "この画像を説明して", // メッセージコンテキストメニューに表示される名前
		Type: discordgo.MessageApplicationCommand,
	}
}

func (c *DescribeImageCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	targetMessage := i.ApplicationCommandData().Resolved.Messages[i.ApplicationCommandData().TargetID]

	var imageURL string
	if len(targetMessage.Attachments) > 0 && len(targetMessage.Attachments[0].ContentType) > 5 && targetMessage.Attachments[0].ContentType[0:5] == "image" {
		imageURL = targetMessage.Attachments[0].URL
	} else if len(targetMessage.Embeds) > 0 && targetMessage.Embeds[0].Image != nil {
		imageURL = targetMessage.Embeds[0].Image.URL
	} else {
		content := "エラー: 対象のメッセージに画像が見つかりませんでした。"
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: content,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		}); err != nil {
			c.Log.Error("Failed to send error response", "error", err)
		}
		return
	}

	// AIに渡すプロンプトを定義（文字起こしメイン）
	prompt := "この画像に文字が書かれている場合は、その内容を正確に書き出してください。文字がない、または読み取れない場合は、画像の内容を簡潔に説明してください。"

	// 「考え中...」と即時応答
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		c.Log.Error("Failed to send initial response", "error", err)
		return
	}

	// Pythonサーバーに送信するデータを作成
	reqData := DescribeImageRequest{ImageURL: imageURL, Prompt: prompt}
	reqJson, _ := json.Marshal(reqData)

	// Pythonサーバーの画像認識エンドポイントにリクエストを送信
	resp, err := http.Post("http://localhost:5001/describe-image", "application/json", bytes.NewBuffer(reqJson))

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

	// レスポンスを読み取りJSONをパース (TextResponseを再利用)
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
		Title:       "🖼️ 画像の文字起こし・説明",
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
			Text: "Powered by Luna Assistant",
		},
	}

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	}); err != nil {
		c.Log.Error("Failed to edit final response", "error", err)
	}
}

func (c *DescribeImageCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
}
func (c *DescribeImageCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *DescribeImageCommand) GetComponentIDs() []string                                        { return []string{} }
func (c *DescribeImageCommand) GetCategory() string                                              { return "AI" }
