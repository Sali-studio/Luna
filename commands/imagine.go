// commands/imagine.go
package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"luna/interfaces"
	"net/http"
	"os"
	"path/filepath"
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

	// 1. まず「生成中です...」と即時応答する
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		c.Log.Error("Failed to send initial response", "error", err)
		return
	}

	// 2. プロンプトを強化する
	// 品質向上キーワード
	qualitySuffix := ", masterpiece, best quality, ultra-detailed, 8k, photorealistic"
	enhancedPrompt := prompt + qualitySuffix

	// ネガティブプロンプトの組み立て
	defaultNegativePrompt := "worst quality, low quality, normal quality, ugly, deformed, blurry, lowres, bad anatomy, bad hands, text, error, missing fingers, extra digit, fewer digits, cropped, jpeg artifacts, signature, watermark, username, bad feet"
	finalNegativePrompt := defaultNegativePrompt
	if userNegativePrompt != "" {
		finalNegativePrompt = strings.Join([]string{defaultNegativePrompt, userNegativePrompt}, ", ")
	}

	// 3. Pythonサーバーに送信するデータを作成
	reqData := ImagineRequest{
		Prompt:         enhancedPrompt,
		NegativePrompt: finalNegativePrompt,
	}
	reqJson, _ := json.Marshal(reqData)

	// 4. PythonサーバーにHTTP POSTリクエストを送信
	resp, err := http.Post("http://localhost:5001/generate-image", "application/json", bytes.NewBuffer(reqJson))
	if err != nil {
		c.Log.Error("画像生成サーバーへの接続に失敗", "error", err)
		content := "エラー: 画像生成サーバーに接続できませんでした。"
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
			c.Log.Error("Failed to edit error response", "error", err)
		}
		return
	}
	defer resp.Body.Close()

	// 5. Pythonサーバーからの応答を読み取る
	body, _ := io.ReadAll(resp.Body)
	var imagineResp struct {
		ImagePath string `json:"image_path"`
		Error     string `json:"error"`
	}
	if err := json.Unmarshal(body, &imagineResp); err != nil {
		c.Log.Error("Failed to unmarshal imagine response", "error", err)
		content := "エラー: サーバーからの応答を解析できませんでした。"
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
			c.Log.Error("Failed to edit error response", "error", err)
		}
		return
	}

	// 6. 応答に応じてメッセージを編集
	if imagineResp.Error != "" || resp.StatusCode != http.StatusOK {
		c.Log.Error("画像の生成に失敗", "error", imagineResp.Error, "status_code", resp.StatusCode)
		content := fmt.Sprintf("エラー: 画像の生成に失敗しました。\n`%s`", imagineResp.Error)
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
			c.Log.Error("Failed to edit error response", "error", err)
		}
		return
	}

	// 7. Pythonから教えられたパスの画像ファイルを開く
	file, err := os.Open(imagineResp.ImagePath)
	if err != nil {
		c.Log.Error("生成された画像ファイルを開けませんでした", "error", err, "path", imagineResp.ImagePath)
		content := "エラー: 生成された画像ファイルを開けませんでした。"
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
			c.Log.Error("Failed to edit error response", "error", err)
		}
		return
	}
	defer file.Close()

	// 8. ファイル名をパスから取得
	fileName := filepath.Base(imagineResp.ImagePath)

	// 9. 成功した場合、Embedとファイルを一緒に投稿
	description := fmt.Sprintf("**Prompt:**\n```\n%s\n```", prompt)
	if userNegativePrompt != "" {
		description += fmt.Sprintf("\n**Negative Prompt:**\n```\n%s\n```", userNegativePrompt)
	}

	embed := &discordgo.MessageEmbed{
		Title: "🎨 画像生成が完了しました",
		Author: &discordgo.MessageEmbedAuthor{
			Name:    i.Member.User.String(),
			IconURL: i.Member.User.AvatarURL(""),
		},
		Description: description,
		Image: &discordgo.MessageEmbedImage{
			URL: fmt.Sprintf("attachment://%s", fileName),
		},
		Color: 0x824ff1, // Gemini Purple
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Powered by Luna",
		},
	}

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
		Files: []*discordgo.File{
			{
				Name:   fileName,
				Reader: file,
			},
		},
	}); err != nil {
		c.Log.Error("Failed to edit final response", "error", err)
	}
}

func (c *ImagineCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *ImagineCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *ImagineCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *ImagineCommand) GetCategory() string                                                  { return "AI" }
