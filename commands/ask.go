package commands

import (
	"luna/gemini"
	"luna/logger"

	"github.com/bwmarrin/discordgo"
)

type AskCommand struct {
	Gemini *gemini.Client
}

func (c *AskCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "ask",
		Description: "Luna Assistant AIに質問します",
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionString, Name: "prompt", Description: "Lunaへの質問内容", Required: true},
		},
	}
}

func (c *AskCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if c.Gemini == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "❌ このコマンドは現在利用できません。", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	prompt := i.ApplicationCommandData().Options[0].StringValue()
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredChannelMessageWithSource})
	responseContent, err := c.Gemini.GenerateContent(prompt)
	if err != nil {
		logger.Error.Printf("Luna APIからの応答取得に失敗: %v", err)
		content := "❌ Luna APIへの接続または応答の取得中にエラーが発生しました。"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
		return
	}
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &responseContent})
}

func (c *AskCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *AskCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *AskCommand) GetComponentIDs() []string                                            { return []string{} }
