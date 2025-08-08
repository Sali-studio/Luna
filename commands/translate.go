package commands

import (
	"context"
	"fmt"
	"luna/ai"
	"luna/interfaces"

	"github.com/bwmarrin/discordgo"
)

type TranslateCommand struct {
	Log interfaces.Logger
	AI  *ai.Client
}

func (c *TranslateCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "translate",
		Description: "Luna Assistantを使用し、テキストを指定された言語に翻訳します",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "text",
				Description: "翻訳したいテキスト",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "target_language",
				Description: "翻訳先の言語 (例: 英語, 日本語, 韓国語、ヘブライ語)",
				Required:    true,
			},
		},
	}
}

func (c *TranslateCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	text := options[0].StringValue()
	targetLang := options[1].StringValue()

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		c.Log.Error("Failed to send deferred response", "error", err)
		return
	}

	prompt := fmt.Sprintf("以下のテキストを「%s」に翻訳してください。翻訳結果のテキストだけを返してください。\n\n[翻訳元テキスト]\n%s", targetLang, text)

	translatedText, err := c.AI.GenerateText(context.Background(), prompt)
	if err != nil {
		c.Log.Error("翻訳に失敗", "error", err)
		content := fmt.Sprintf("エラー: 翻訳に失敗しました。\n`%s`", err.Error())
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
			c.Log.Error("Failed to edit error response", "error", err)
		}
		return
	}

	embed := &discordgo.MessageEmbed{
		Title: "🌐 翻訳結果",
		Color: 0x4CAF50,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "翻訳元", Value: "```\n" + text + "\n```"},
			{Name: "翻訳先 (" + targetLang + ")", Value: "```\n" + translatedText + "\n```"},
		},
	}
	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	}); err != nil {
		c.Log.Error("Failed to edit final response", "error", err)
	}
}

func (c *TranslateCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *TranslateCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *TranslateCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *TranslateCommand) GetCategory() string                                                  { return "AI" }