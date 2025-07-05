package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/bwmarrin/discordgo"
)

type TranslateCommand struct{}

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

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	prompt := fmt.Sprintf("以下のテキストを「%s」に翻訳してください。翻訳結果のテキストだけを返してください。\n\n[翻訳元テキスト]\n%s", targetLang, text)

	reqData := TextRequest{Prompt: prompt}
	reqJson, _ := json.Marshal(reqData)

	resp, err := http.Post("http://localhost:5001/generate-text", "application/json", bytes.NewBuffer(reqJson))
	if err != nil {
		content := "エラー: Luna Assistantサーバーへの接続に失敗しました。"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
		return
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var textResp TextResponse
	json.Unmarshal(body, &textResp)

	if textResp.Error != "" || resp.StatusCode != http.StatusOK {
		content := fmt.Sprintf("エラー: 翻訳に失敗しました。\n`%s`", textResp.Error)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
		return
	}

	embed := &discordgo.MessageEmbed{
		Title: "🌐 翻訳結果",
		Color: 0x4CAF50,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "翻訳元", Value: "```\n" + text + "\n```"},
			{Name: "翻訳先 (" + targetLang + ")", Value: "```\n" + textResp.Text + "\n```"},
		},
	}
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
}

func (c *TranslateCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *TranslateCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *TranslateCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *TranslateCommand) GetCategory() string                                                  { return "AI" }
