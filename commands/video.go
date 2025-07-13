// commands/video.go
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
type VideoRequest struct {
	Prompt string `json:"prompt"`
}

type VideoCommand struct {
	Log interfaces.Logger
}

func (c *VideoCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "video",
		Description: "Luna Assistantで動画を生成します（実験的機能）",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "prompt",
				Description: "生成したい動画の説明 (必ず英語で指示を出してください。) (例: A majestic lion roaring on a rocky outcrop)",
				Required:    true,
			},
		},
	}
}

func (c *VideoCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// ユーザーが入力したプロンプトを取得
	prompt := i.ApplicationCommandData().Options[0].StringValue()

	// 1. まず「生成中です...」と即時応答する (時間のかかる処理のため)
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		return
	}

	// 2. Pythonサーバーに送信するデータを作成
	reqData := VideoRequest{Prompt: prompt}
	reqJson, _ := json.Marshal(reqData)

	// 3. PythonサーバーにHTTP POSTリクエストを送信
	resp, err := http.Post("http://localhost:5001/generate-video", "application/json", bytes.NewBuffer(reqJson))
	if err != nil {
		c.Log.Error("動画生成サーバーへの接続に失敗", "error", err)
		content := "エラー: 動画生成サーバーに接続できませんでした。"
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
			c.Log.Error("Failed to edit error response", "error", err)
		}
		return
	}
	defer resp.Body.Close()

	// 4. Pythonサーバーからの応答を読み取る
	body, _ := io.ReadAll(resp.Body)
	var videoResp struct {
		VideoPath string `json:"video_path"`
		Error     string `json:"error"`
	}
	if err := json.Unmarshal(body, &videoResp); err != nil {
		c.Log.Error("Failed to unmarshal video response", "error", err)
	}

	// 5. 応答に応じてメッセージを編集
	if videoResp.Error != "" || resp.StatusCode != http.StatusOK {
		c.Log.Error("動画の生成に失敗", "error", videoResp.Error, "status_code", resp.StatusCode)
		content := fmt.Sprintf("エラー: 動画の生成に失敗しました。\n`%s`", videoResp.Error)
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
			c.Log.Error("Failed to edit error response", "error", err)
		}
		return
	}

	// 6. Pythonから教えられたパスの動画ファイルを開く
	file, err := os.Open(videoResp.VideoPath)
	if err != nil {
		c.Log.Error("生成された動画ファイルを開けませんでした", "error", err, "path", videoResp.VideoPath)
		content := "エラー: 生成され動画ファイルを開けませんでした。"
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
			c.Log.Error("Failed to edit error response", "error", err)
		}
		return
	}
	defer file.Close()

	// 7. ファイル名をパスから取得
	fileName := filepath.Base(videoResp.VideoPath)

	// 8. 成功した場合、Embedとファイルを一緒に投稿
	embed := &discordgo.MessageEmbed{
		Title: "🎬 動画生成が完了しました",
		Author: &discordgo.MessageEmbedAuthor{
			Name:    i.Member.User.String(),
			IconURL: i.Member.User.AvatarURL(""),
		},
		Description: fmt.Sprintf("**Prompt:**\n```\n%s\n```", prompt),
		// DiscordのEmbedに直接ビデオは埋め込めないため、ファイルとして添付します。
		// 必要であれば、生成したビデオをどこかにアップロードし、そのURLをここに記載することも可能です。
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

func (c *VideoCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *VideoCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *VideoCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *VideoCommand) GetCategory() string                                                  { return "AI" }
