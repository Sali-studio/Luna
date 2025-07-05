// commands/imagine.go
package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/bwmarrin/discordgo"
)

// Pythonサーバーに送るリクエストの構造体
type ImagineRequest struct {
	Prompt string `json:"prompt"`
}

// Pythonサーバーから返ってくるレスポンスの構造体
type ImagineResponse struct {
	ImageURL string `json:"image_url"`
	Error    string `json:"error"`
}

type ImagineCommand struct{}

func (c *ImagineCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "imagine",
		Description: "Luna Assistantで画像を生成します",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "prompt",
				Description: "生成したい画像の説明 (例: 宇宙を泳ぐ猫)",
				Required:    true,
			},
		},
	}
}

func (c *ImagineCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// ユーザーが入力したプロンプトを取得
	prompt := i.ApplicationCommandData().Options[0].StringValue()

	// 1. まず「生成中です...」と即時応答する (時間のかかる処理のため)
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		return
	}

	// 2. Pythonサーバーに送信するデータを作成
	reqData := ImagineRequest{Prompt: prompt}
	reqJson, _ := json.Marshal(reqData)

	// 3. PythonサーバーにHTTP POSTリクエストを送信
	resp, err := http.Post("http://localhost:5001/generate-image", "application/json", bytes.NewBuffer(reqJson))
	if err != nil {
		// Pythonサーバーに接続できなかった場合
		content := "エラー: 画像生成サーバーに接続できませんでした。"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
		return
	}
	defer resp.Body.Close()

	// 4. Pythonサーバーからの応答を読み取る
	body, _ := ioutil.ReadAll(resp.Body)
	var imagineResp ImagineResponse
	json.Unmarshal(body, &imagineResp)

	// 5. 応答に応じてメッセージを編集
	if imagineResp.Error != "" || resp.StatusCode != http.StatusOK {
		// Pythonサーバー側でエラーが発生した場合
		content := fmt.Sprintf("エラー: 画像の生成に失敗しました。\n`%s`", imagineResp.Error)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
		return
	}

	// 成功した場合、Embedを作成して画像を投稿
	embed := &discordgo.MessageEmbed{
		Title: "🎨 画像生成が完了しました",
		Author: &discordgo.MessageEmbedAuthor{
			Name:    i.Member.User.String(),
			IconURL: i.Member.User.AvatarURL(""),
		},
		Description: fmt.Sprintf("**Prompt:**\n```\n%s\n```", prompt),
		Image: &discordgo.MessageEmbedImage{
			URL: imagineResp.ImageURL,
		},
		Color: 0x824ff1, // Gemini Purple
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
}

func (c *ImagineCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *ImagineCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *ImagineCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *ImagineCommand) GetCategory() string                                                  { return "AI" }
