package commands

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/bwmarrin/discordgo"
)

type MandelbrotCommand struct{}

func (c *MandelbrotCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "mandelbrot",
		Description: "Juliaの高速計算でマンデルブロ集合を生成します",
	}
}

func (c *MandelbrotCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: "⏳ Juliaサーバーで画像を生成中です..."},
	})

	// Juliaサーバーにリクエストを送信 (ポート番号はmandelbrot_server.jlと合わせる)
	resp, err := http.Get("http://localhost:8001/mandelbrot")
	if err != nil {
		content := "エラー: Julia計算サーバーへの接続に失敗しました。"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		content := fmt.Sprintf("エラー: 計算サーバーがエラーを返しました (ステータス: %d)", resp.StatusCode)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
		return
	}

	// レスポンスから画像データを読み込む
	imageData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		content := "エラー: 画像データの読み込みに失敗しました。"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
		return
	}

	// Discordに画像ファイルを送信
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &[]string{"✅ Juliaによるマンデルブロ集合の生成が完了しました！"}[0],
		Files: []*discordgo.File{
			{
				Name:        "mandelbrot.png",
				ContentType: "image/png",
				Reader:      bytes.NewReader(imageData),
			},
		},
	})
}

func (c *MandelbrotCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *MandelbrotCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *MandelbrotCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *MandelbrotCommand) GetCategory() string                                                  { return "ツール" }
