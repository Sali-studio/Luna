package commands

import (
	"context"
	"fmt"
	"luna/ai"
	"luna/interfaces"

	"github.com/bwmarrin/discordgo"
)

// OcrCommand は画像からの文字起こし（OCR）を実行します。
type OcrCommand struct {
	Log interfaces.Logger
	AI  *ai.Client
}

// Pythonサーバーに送る画像URLリクエストの構造体
type ImageUrlRequest struct {
	ImageUrl string `json:"image_url"`
}

func (c *OcrCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "ocr",
		Description: "画像から文字を抽出します (OCR)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionAttachment,
				Name:        "image",
				Description: "文字を抽出したい画像ファイル",
				Required:    true,
			},
		},
	}
}

func (c *OcrCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// 添付された画像を取得
	attachmentID := i.ApplicationCommandData().Options[0].Value.(string)
	attachment := i.ApplicationCommandData().Resolved.Attachments[attachmentID]

	// 1. まず「処理中です...」と即時応答
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		c.Log.Error("Failed to send initial OCR response", "error", err)
		return
	}

	// AIにOCRを依頼
	prompt := "この画像からテキストを正確に抽出してください。画像に写っているテキストだけを、他の余計な説明や前置きなしで書き出してください。"
	responseText, err := c.AI.GenerateTextFromImage(context.Background(), prompt, attachment.URL)

	// エラーハンドリング
	if err != nil {
		c.Log.Error("AIからの応答生成に失敗", "error", err)
		content := "エラー: AIからの応答の取得に失敗しました。"
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
			c.Log.Error("Failed to edit error response", "error", err)
		}
		return
	}

	// 5. 成功メッセージをEmbedで作成
	embed := &discordgo.MessageEmbed{
		Title:       "📝 文字起こし結果",
		Description: fmt.Sprintf("```\n%s\n```", responseText),
		Color:       0x824ff1, // Gemini Purple
		Author: &discordgo.MessageEmbedAuthor{
			Name:    i.Member.User.String(),
			IconURL: i.Member.User.AvatarURL(""),
		},
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: attachment.URL,
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Powered by Luna AI",
		},
	}

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	}); err != nil {
		c.Log.Error("Failed to edit final OCR response", "error", err)
	}
}

func (c *OcrCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *OcrCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *OcrCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *OcrCommand) GetCategory() string                                                  { return "AI" }
