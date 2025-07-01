package commands

import (
	"fmt"
	"luna/gemini" // geminiパッケージをインポート
	"luna/logger"

	"github.com/bwmarrin/discordgo"
)

const (
	TranslateModalCustomID = "translate_modal"
)

type TranslateCommand struct {
	// 依存関係をGeminiクライアントに変更
	Gemini *gemini.Client
}

func (c *TranslateCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "translate",
		Description: "テキストをLuna Translatorを使って翻訳します",
	}
}

func (c *TranslateCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: TranslateModalCustomID,
			Title:    "翻訳",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "text", Label: "翻訳したいテキスト", Style: discordgo.TextInputParagraph, Required: true},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "lang", Label: "翻訳先の言語 (例: 英語, 日本語, 韓国語)", Style: discordgo.TextInputShort, Placeholder: "英語", Required: true},
				}},
			},
		},
	})
	if err != nil {
		logger.Error.Printf("Translateモーダルの表示に失敗: %v", err)
	}
}

func (c *TranslateCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if c.Gemini == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "❌ 翻訳機能は現在利用できません。", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	data := i.ModalSubmitData()
	text := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	lang := data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral}})

	// Geminiへの指示（プロンプト）を作成
	prompt := fmt.Sprintf("以下のテキストを %s に翻訳してください:\n\n---\n%s", lang, text)

	// Geminiに翻訳を実行させる
	translatedText, err := c.Gemini.GenerateContent(prompt)
	if err != nil {
		logger.Error.Printf("Luna Translatorからの翻訳応答取得に失敗: %v", err)
		content := "❌ 翻訳中にエラーが発生しました。"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
		return
	}

	embed := &discordgo.MessageEmbed{
		Title: "翻訳結果 (by Gemini)",
		Fields: []*discordgo.MessageEmbedField{
			{Name: "原文", Value: text},
			{Name: fmt.Sprintf("翻訳文 (%s)", lang), Value: translatedText},
		},
		Color: 0x4a8cf7, // Google Blue
	}
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Embeds: &[]*discordgo.MessageEmbed{embed}})
}

func (c *TranslateCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *TranslateCommand) GetComponentIDs() []string                                            { return []string{TranslateModalCustomID} }
