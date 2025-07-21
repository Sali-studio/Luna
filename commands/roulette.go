package commands

import (
	"fmt"
	"luna/interfaces"
	"math/rand"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// RouletteCommand は、選択肢の中からランダムに1つを選ぶコマンドです。
type RouletteCommand struct {
	Log interfaces.Logger
}

func (c *RouletteCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "roulette",
		Description: "指定された選択肢の中からランダムで1つを選びます。",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "choices",
				Description: "選択肢をスペース区切りで入力してください。 (例: ラーメン ピザ 寿司)",
				Required:    true,
			},
		},
	}
}

func (c *RouletteCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// 最初に遅延応答を送信し、「考え中...」のような状態を示す
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		c.Log.Error("Failed to send deferred response", "error", err)
		return
	}

	options := i.ApplicationCommandData().Options
	choicesStr := options[0].StringValue()
	choicesRaw := strings.Split(choicesStr, " ") // スペースで選択肢を分割
	var choices []string
	for _, choice := range choicesRaw {
		trimmedChoice := strings.TrimSpace(choice)
		if trimmedChoice != "" {
			choices = append(choices, trimmedChoice)
		}
	}

	if len(choices) < 2 {
		content := "選択肢は2つ以上入力してください。"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
		return
	}

	// ルーレットのアニメーション風Embed
	thinkingEmbed := &discordgo.MessageEmbed{
		Title: "룰렛หมุน...", // "ルーレット回転..." を多言語で
		Description: "さて、どれにしようかな...",
		Color: 0x3498db, // Blue
		Image: &discordgo.MessageEmbedImage{
			URL: "https://i.gifer.com/ZNeT.gif", // 回転するGIF
		},
	}
	// "考え中" のメッセージをアニメーションEmbedに更新
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{thinkingEmbed},
	})

	// 乱数のシードを初期化
	rand.Seed(time.Now().UnixNano())
	// ランダムに結果を選択
	winner := choices[rand.Intn(len(choices))]

	// 2秒待って結果を表示（アニメーションを見せるため）
	time.Sleep(2 * time.Second)

	// 結果表示用のEmbed
	resultEmbed := &discordgo.MessageEmbed{
		Title: "🎉 結果は...!",
		Description: fmt.Sprintf("\n## **%s**\n\nに決定しました！", winner),
		Color: 0x2ecc71, // Green
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "すべての選択肢",
				Value: fmt.Sprintf("`%s`", strings.Join(choices, "`, `")),
			},
		},
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: "https://i.imgur.com/kCnD41z.png", // 当たりっぽいアイコン
		},
	}

	// アニメーションEmbedを最終結果に更新
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{resultEmbed},
	})
}

func (c *RouletteCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *RouletteCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *RouletteCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *RouletteCommand) GetCategory() string                                                  { return "Fun" }
