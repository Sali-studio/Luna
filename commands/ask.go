// commands/ask.go
package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/bwmarrin/discordgo"
)

// Pythonサーバーに送るテキスト生成リクエストの構造体
type TextRequest struct {
	Prompt string `json:"prompt"`
}

// Pythonサーバーから返ってくるテキスト生成レスポンスの構造体
type TextResponse struct {
	Text  string `json:"text"`
	Error string `json:"error"`
}

type AskCommand struct{}

func (c *AskCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "ask",
		Description: "Luna Assistantに質問します",
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionString, Name: "prompt", Description: "質問内容", Required: true},
		},
	}
}

// 内部の処理を、PythonサーバーへのHTTPリクエストに変更
func (c *AskCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	prompt := i.ApplicationCommandData().Options[0].StringValue()

	// 「考え中...」と即時応答
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	// Pythonサーバーに送信するデータを作成
	reqData := TextRequest{Prompt: prompt}
	reqJson, _ := json.Marshal(reqData)

	// Pythonサーバーのテキスト生成エンドポイントにリクエストを送信
	resp, err := http.Post("http://localhost:5001/generate-text", "application/json", bytes.NewBuffer(reqJson))
	if err != nil {
		content := "エラー: AIサーバーへの接続に失敗しました。"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
		return
	}
	defer resp.Body.Close()

	// レスポンスを読み取りJSONをパース
	body, _ := ioutil.ReadAll(resp.Body)
	var textResp TextResponse
	json.Unmarshal(body, &textResp)

	if textResp.Error != "" || resp.StatusCode != http.StatusOK {
		content := fmt.Sprintf("エラー: Luna Assistantからの応答取得に失敗しました。\n`%s`", textResp.Error)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
		return
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &textResp.Text,
	})
}

func (c *AskCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *AskCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *AskCommand) GetComponentIDs() []string {
	return []string{}
}
func (c *AskCommand) GetCategory() string {
	return "AI"
}
