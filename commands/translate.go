package commands

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"luna/logger"
	"net/http"
	"net/url"
	"os"

	"github.com/bwmarrin/discordgo"
)

const (
	TranslateModalCustomID = "translate_modal"
	TranslateTextID        = "translate_text"
	TranslateTargetLangID  = "translate_target_lang"
)

// Google Apps Script APIのレスポンス
type gasResponse struct {
	Code int    `json:"code"`
	Text string `json:"text"`
}

type TranslateCommand struct{}

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
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID: TranslateTextID,
							Label:    "翻訳したいテキスト",
							Style:    discordgo.TextInputParagraph,
							Required: true,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    TranslateTargetLangID,
							Label:       "翻訳先の言語コード (例: en, ja, ko)",
							Style:       discordgo.TextInputShort,
							Placeholder: "en",
							Required:    true,
							MinLength:   2,
							MaxLength:   2,
						},
					},
				},
			},
		},
	})
	if err != nil {
		logger.Error.Printf("Translateモーダルの表示に失敗: %v", err)
	}
}

func (c *TranslateCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	if data.CustomID != TranslateModalCustomID {
		return
	}

	text := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	lang := data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	// ここにGoogle Apps ScriptのエンドポイントURLを設定してください
	gasURL := os.Getenv("GOOGLE_TRANSLATE_API_URL")
	if gasURL == "" {
		logger.Error.Println("環境変数 'GOOGLE_TRANSLATE_API_URL' が未設定です。")
		// ... エラーレスポンス ...
		return
	}

	// APIリクエスト
	resp, err := http.Get(fmt.Sprintf("%s?text=%s&target=%s", gasURL, url.QueryEscape(text), lang))
	if err != nil {
		// ... エラーレスポンス ...
		return
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var result gasResponse
	json.Unmarshal(body, &result)

	// Embedを作成
	embed := &discordgo.MessageEmbed{
		Title: "翻訳結果",
		Fields: []*discordgo.MessageEmbedField{
			{Name: "原文", Value: text},
			{Name: "翻訳文 (" + lang + ")", Value: result.Text},
		},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func (c *TranslateCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
