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
			{Type: discordgo.ApplicationCommandOptionString, Name: "prompt", Description: "質問内容", Required: true},
		},
	}
}

func (c *AskCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if c.Gemini == nil {
		logger.Warn("Luna Assistantが設定されていないため、askコマンドが実行されましたが処理を中断しました。")
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "❌ このコマンドは現在、管理者によって無効化されています。", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	prompt := i.ApplicationCommandData().Options[0].StringValue()

	persona := "あなたは「Luna Assistant」という名前の、高性能で親切なAIアシスタントです。穏やかで、知的で、常にユーザーに寄り添い、丁寧な言葉遣いで回答してください。一人称は「私」を使ってください。"

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredChannelMessageWithSource})

	responseContent, err := c.Gemini.GenerateContent(prompt, persona)

	if err != nil {
		logger.Error("Geminiからの応答取得に失敗", "error", err, "prompt", prompt)
		content := "❌ AIへの接続または応答の取得中にエラーが発生しました。"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
		return
	}
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &responseContent})
}

func (c *AskCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *AskCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *AskCommand) GetComponentIDs() []string {
	return []string{}
}
func (c *AskCommand) GetCategory() string {
	return "AI"
}
