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

	"github.com/bwmarrin/discordgo"
)

// Pythonサーバーに送るリクエストの構造体
type ImagineRequest struct {
	Prompt string `json:"prompt"`
}

type ImagineCommand struct{
	Log interfaces.Logger
}

func (c *ImagineCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "imagine",
		Description: "Luna Assistantで画像を生成します（実験的機能）",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "prompt",
				Description: "生成したい画像の説明 (必ず英語で指示を出してください。) (例: The Bear Swimming in Space)",
				Required:    true,
			},
		},
	}
}

func (c *ImagineCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// ユーザーが入力したプロンプトを取得
	prompt := i.ApplicationCommandData().Options[0].StringValue()

	// 1. まず「生成中です...」と即時応答する (時間のかかる処理のため)
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		return
	}

	// 2. Pythonサーバーに送信するデータを作成
	reqData := ImagineRequest{Prompt: prompt}
	reqJson, _ := json.Marshal(reqData)

	// 3. PythonサーバーにHTTP POSTリクエストを送信
	resp, err := http.Post("http://localhost:5001/generate-image", "application/json", bytes.NewBuffer(reqJson))
	if err != nil {
		c.Log.Error("画像生成サーバーへの接続に失敗", "error", err)
		// Pythonサーバーに接続できなかった場合
		content := "エラー: 画像生成サーバーに接続できませんでした。"
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
			c.Log.Error("Failed to edit error response", "error", err)
		}
		return
	}
	defer resp.Body.Close()

	// 4. Pythonサーバーからの応答を読み取る
	body, _ := io.ReadAll(resp.Body)
	var imagineResp struct {
		ImagePath string `json:"image_path"`
		Error     string `json:"error"`
	}
	if err := json.Unmarshal(body, &imagineResp); err != nil {
		c.Log.Error("Failed to unmarshal imagine response", "error", err)
	}

	// 5. 応答に応じてメッセージを編集
	if imagineResp.Error != "" || resp.StatusCode != http.StatusOK {
		c.Log.Error("画像の生成に失敗", "error", imagineResp.Error, "status_code", resp.StatusCode)
		// Pythonサーバー側でエラーが発生した場合
		content := fmt.Sprintf("エラー: 画像の生成に失敗しました。\n`%s`", imagineResp.Error)
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
			c.Log.Error("Failed to edit error response", "error", err)
		}
		return
	}

	// 6. Pythonから教えられたパスの画像ファイルを開く
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

	// 7. ファイル名をパスから取得
	fileName := filepath.Base(imagineResp.ImagePath)

	// 8. 成功した場合、Embedとファイルを一緒に投稿
	embed := &discordgo.MessageEmbed{
		Title: "🎨 画像生成が完了しました",
		Author: &discordgo.MessageEmbedAuthor{
			Name:    i.Member.User.String(),
			IconURL: i.Member.User.AvatarURL(""),
		},
		Description: fmt.Sprintf("**Prompt:**\n```\n%s\n```", prompt),
		Image: &discordgo.MessageEmbedImage{
			URL: fmt.Sprintf("attachment://%s", fileName),
		},
		Color: 0x824ff1, // Gemini Purple
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