// commands/describe_image.go
package commands

import (
	"context"
	"fmt"
	"luna/ai"
	"luna/interfaces"

	"github.com/bwmarrin/discordgo"
)

// Pythonサーバーに送る画像認識リクエストの構造体
type DescribeImageRequest struct {
	ImageURL string `json:"image_url"`
}

type DescribeImageCommand struct {
	Log interfaces.Logger
	AI  *ai.Client
}

func (c *DescribeImageCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name: "Luna Assistantで画像を説明",
		Type: discordgo.MessageApplicationCommand, // メッセージコマンドとして定義
	}
}

func (c *DescribeImageCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// 対象のメッセージを取得
	targetMessage := i.ApplicationCommandData().Resolved.Messages[i.ApplicationCommandData().TargetID]

	// メッセージに画像が含まれているかチェック
	var imageURL string
	if len(targetMessage.Attachments) > 0 && len(targetMessage.Attachments[0].ContentType) > 5 && targetMessage.Attachments[0].ContentType[0:5] == "image" {
		imageURL = targetMessage.Attachments[0].URL
	} else if len(targetMessage.Embeds) > 0 && targetMessage.Embeds[0].Image != nil {
		imageURL = targetMessage.Embeds[0].Image.URL
	} else {
		// 画像が見つからない場合
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

	// 「考え中...」と即時応答
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		c.Log.Error("Failed to send initial response", "error", err)
		return
	}

	// AIに画像の説明を依頼
	prompt := "この画像を詳細に説明してください。"
	responseText, err := c.AI.GenerateTextFromImage(context.Background(), prompt, imageURL)

	// エラーハンドリング
	if err != nil {
		c.Log.Error("AIからの応答生成に失敗", "error", err)
		content := "エラー: AIからの応答の取得に失敗しました。"
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
			c.Log.Error("Failed to edit error response", "error", err)
		}
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🖼️ 画像の説明",
		Description: responseText,
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
