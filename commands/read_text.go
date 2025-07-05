package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/bwmarrin/discordgo"
)

type OcrResponse struct {
	Text string `json:"text"`
}

type ReadTextCommand struct{}

func (c *ReadTextCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "read-text",
		Description: "画像からテキストを抽出します (Powered by C# & Tesseract)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionAttachment,
				Name:        "image",
				Description: "テキストを読み取りたい画像",
				Required:    true,
			},
		},
	}
}

func (c *ReadTextCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	// ここからが正しいアタッチメントの取得方法です。
	options := i.ApplicationCommandData().Options
	attachmentID := options[0].Value.(string)
	attachment := i.ApplicationCommandData().Resolved.Attachments[attachmentID]

	// 画像をダウンロード
	resp, err := http.Get(attachment.URL)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &[]string{"エラー: 画像のダウンロードに失敗しました。"}[0]})
		return
	}
	defer resp.Body.Close()
	imgData, _ := io.ReadAll(resp.Body)

	// multipart/form-data 形式でC#サーバーに送信
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("image", attachment.Filename)
	io.Copy(part, bytes.NewReader(imgData))
	writer.Close()

	// C#サーバーにリクエストを送信 (ポート番号はProgram.csと合わせる)
	req, _ := http.NewRequest("POST", "http://localhost:5002/read-text", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	ocrResp, err := client.Do(req)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &[]string{"エラー: C#のOCRサーバーに接続できませんでした。"}[0]})
		return
	}
	defer ocrResp.Body.Close()

	// レスポンスを解析
	var ocrResult OcrResponse
	json.NewDecoder(ocrResp.Body).Decode(&ocrResult)

	if ocrResult.Text == "" {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &[]string{"画像からテキストを検出できませんでした。"}[0]})
		return
	}

	// 長文の場合は分割して送信
	content := fmt.Sprintf("```\n%s\n```", ocrResult.Text)
	if len(content) > 2000 {
		content = "抽出したテキストが長すぎるため、最初の2000文字まで表示します。\n" + content[:1950] + "..."
	}
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
}

func (c *ReadTextCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *ReadTextCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *ReadTextCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *ReadTextCommand) GetCategory() string                                                  { return "ツール" }
