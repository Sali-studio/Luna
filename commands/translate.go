package commands

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"luna/logger"
	"net/http"
	"net/url"

	"github.comcom/bwmarrin/discordgo"
)

const (
	TranslateModalCustomID = "translate_modal"
)

type gasResponse struct {
	Code int    `json:"code"`
	Text string `json:"text"`
}

type TranslateCommand struct {
	APIURL string
}

func (c *TranslateCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "translate",
		Description: "テキストを翻訳します",
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
					discordgo.TextInput{CustomID: "lang", Label: "翻訳先の言語コード (例: en, ja, ko)", Style: discordgo.TextInputShort, Placeholder: "en", Required: true, MinLength: 2, MaxLength: 7},
				}},
			},
		},
	})
	if err != nil {
		logger.Error.Printf("Translateモーダルの表示に失敗: %v", err)
	}
}

func (c *TranslateCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if c.APIURL == "" {
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

	resp, err := http.Get(fmt.Sprintf("%s?text=%s&target=%s", c.APIURL, url.QueryEscape(text), lang))
	if err != nil { /* ... エラー処理 ... */
		return
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var result gasResponse
	json.Unmarshal(body, &result)

	embed := &discordgo.MessageEmbed{
		Title: "翻訳結果",
		Fields: []*discordgo.MessageEmbedField{
			{Name: "原文", Value: text},
			{Name: "翻訳文 (" + lang + ")", Value: result.Text},
		},
	}
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Embeds: &[]*discordgo.MessageEmbed{embed}})
}

func (c *TranslateCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *TranslateCommand) GetComponentIDs() []string                                            { return []string{TranslateModalCustomID} }
