package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"luna/interfaces"

	"github.com/bwmarrin/discordgo"
)

// OcrCommand は画像からの文字起こし（OCR）を実行します。
type OcrCommand struct {
	Log interfaces.Logger
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

	// 2. Pythonサーバーに送信するデータを作成
	reqData := ImageUrlRequest{ImageUrl: attachment.URL}
	reqJson, _ := json.Marshal(reqData)

	// 3. PythonサーバーのOCRエンドポイントにリクエストを送信
	resp, err := http.Post("http://localhost:5001/ocr", "application/json", bytes.NewBuffer(reqJson))
	if err != nil {
		c.Log.Error("OCRサーバーへの接続に失敗", "error", err)
		content := "エラー: OCRサーバーへの接続に失敗しました。"
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
			c.Log.Error("Failed to edit OCR error response", "error", err)
		}
		return
	}
	defer resp.Body.Close()

	// 4. レスポンスを読み取りJSONをパース
	body, _ := io.ReadAll(resp.Body)
	var ocrResp struct {
		Text  string `json:"text"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &ocrResp); err != nil {
		c.Log.Error("Failed to unmarshal OCR response", "error", err)
		return
	}

	if ocrResp.Error != "" || resp.StatusCode != http.StatusOK {
		c.Log.Error("文字の抽出に失敗", "error", ocrResp.Error, "status_code", resp.StatusCode)
		content := fmt.Sprintf("エラー: 文字の抽出に失敗しました。\n`%s`", ocrResp.Error)
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
			c.Log.Error("Failed to edit OCR error response", "error", err)
		}
		return
	}

	// 5. 成功メッセージをEmbedで作成
	embed := &discordgo.MessageEmbed{
		Title:       "📝 文字起こし結果",
		Description: fmt.Sprintf("```\n%s\n```", ocrResp.Text),
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
