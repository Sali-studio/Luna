// commands/imagine.go
package commands

import (
	"context"
	"fmt"
	"luna/ai"
	"luna/interfaces"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Pythonサーバーに送るリクエストの構造体
type ImagineRequest struct {
	Prompt         string `json:"prompt"`
	NegativePrompt string `json:"negative_prompt,omitempty"`
}

type ImagineCommand struct {
	Log interfaces.Logger
	AI  *ai.Client
}

func (c *ImagineCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "imagine",
		Description: "Luna Assistantで画像を生成します",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "prompt",
				Description: "生成したい画像の説明（例: 宇宙を泳ぐクマ）",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "negative_prompt",
				Description: "生成してほしくない要素（例: 低品質, ぼやけ）",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        "no_enhancements",
				Description: "プロンプトの自動補完を無効化します (デフォルト: false)",
				Required:    false,
			},
		},
	}
}

func (c *ImagineCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// オプションをマップに変換して簡単にアクセスできるようにする
	options := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(i.ApplicationCommandData().Options))
	for _, opt := range i.ApplicationCommandData().Options {
		options[opt.Name] = opt
	}

	// ユーザーが入力したプロンプトを取得
	prompt := options["prompt"].StringValue()
	userNegativePrompt := ""
	if opt, ok := options["negative_prompt"]; ok {
		userNegativePrompt = opt.StringValue()
	}
	noEnhancements := false
	if opt, ok := options["no_enhancements"]; ok {
		noEnhancements = opt.BoolValue()
	}

	// 1. まず「生成中です...」と即時応答する
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		c.Log.Error("Failed to send initial response", "error", err)
		return
	}

	// プロンプトを組み立て
	finalPrompt := prompt
	finalNegativePrompt := userNegativePrompt
	if !noEnhancements {
		qualitySuffix := ", masterpiece, best quality, ultra-detailed, 8k, photorealistic"
		finalPrompt = prompt + qualitySuffix
		defaultNegativePrompt := "worst quality, low quality, normal quality, ugly, deformed, blurry, lowres, bad anatomy, bad hands, text, error, missing fingers, extra digit, fewer digits, cropped, jpeg artifacts, signature, watermark, username, bad feet"
		if userNegativePrompt != "" {
			finalNegativePrompt = strings.Join([]string{defaultNegativePrompt, userNegativePrompt}, ", ")
		} else {
			finalNegativePrompt = defaultNegativePrompt
		}
	}

	// 2. プロンプト補完後のメッセージを送信
	generatingEmbed := &discordgo.MessageEmbed{
		Title: "🎨 画像生成中...",
		Description: fmt.Sprintf("**Prompt:**\n```\n%s\n```\n**Negative Prompt:**\n```\n%s\n```", finalPrompt, finalNegativePrompt),
		Color:       0x824ff1, // Gemini Purple
		Author: &discordgo.MessageEmbedAuthor{
			Name:    i.Member.User.String(),
			IconURL: i.Member.User.AvatarURL(""),
		},
		Image: &discordgo.MessageEmbedImage{
			URL: "https://i.gifer.com/ZNeT.gif", // 生成中GIF
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Powered by Luna | 生成中...",
		},
	}
	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{generatingEmbed},
	}); err != nil {
		c.Log.Error("Failed to edit generating response", "error", err)
	}

	// AIに画像生成を依頼
	imageURL, err := c.AI.GenerateImage(context.Background(), finalPrompt)

	// 6. 応答に応じてメッセージを編集
	if err != nil {
		c.Log.Error("画像の生成に失敗", "error", err)
		content := fmt.Sprintf("エラー: 画像の生成に失敗しました。\n`%s`", err.Error())
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
			c.Log.Error("Failed to edit error response", "error", err)
		}
		return
	}

	// 7. 成功した場合、Embedを更新
	description := fmt.Sprintf("**Prompt:**\n```\n%s\n```", prompt)
	if userNegativePrompt != "" {
		description += fmt.Sprintf("\n**Negative Prompt:**\n```\n%s\n```", userNegativePrompt)
	}
	if noEnhancements {
		description += "\n*プロンプトの自動補完は無効化されています。*"
	}

	embed := &discordgo.MessageEmbed{
		Title: "🎨 画像生成が完了しました",
		Author: &discordgo.MessageEmbedAuthor{
			Name:    i.Member.User.String(),
			IconURL: i.Member.User.AvatarURL(""),
		},
		Description: description,
		Image: &discordgo.MessageEmbedImage{
			URL: imageURL,
		},
		Color: 0x824ff1, // Gemini Purple
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

func (c *ImagineCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *ImagineCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *ImagineCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *ImagineCommand) GetCategory() string                                                  { return "AI" }

