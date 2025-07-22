package commands

import (
	"fmt"
	"luna/interfaces"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// WTBRCommand は、War ThunderのBRをランダムに選択するコマンドです。
type WTBRCommand struct {
	Log interfaces.Logger
}

// BRデータ定義
var (
	airBRs = []float64{
		1.0, 1.3, 1.7, 2.0, 2.3, 2.7, 3.0, 3.3, 3.7, 4.0, 4.3, 4.7, 5.0, 5.3, 5.7, 6.0, 6.3, 6.7, 7.0, 7.3, 7.7, 8.0, 8.3, 8.7, 9.0, 9.3, 9.7, 10.0, 10.3, 10.7, 11.0, 11.3, 11.7, 12.0, 12.3, 12.7, 13.0, 13.3, 13.7, 14.0,
	}
	groundBRs = []float64{
		1.0, 1.3, 1.7, 2.0, 2.3, 2.7, 3.0, 3.3, 3.7, 4.0, 4.3, 4.7, 5.0, 5.3, 5.7, 6.0, 6.3, 6.7, 7.0, 7.3, 7.7, 8.0, 8.3, 8.7, 9.0, 9.3, 9.7, 10.0, 10.3, 10.7, 11.0, 11.3, 11.7, 12.0,
	}
	navalBRs = []float64{
		1.0, 1.3, 1.7, 2.0, 2.3, 2.7, 3.0, 3.3, 3.7, 4.0, 4.3, 4.7, 5.0, 5.3, 5.7, 6.0, 6.3, 6.7, 7.0, 7.3, 7.7, 8.0, 8.3, 8.7,
	}
)

func (c *WTBRCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "wtbr",
		Description: "War ThunderのBRをランダムに選択します。",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "mode",
				Description: "ルーレットを行うゲームモードを選択します。",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "空", Value: "air"},
					{Name: "陸", Value: "ground"},
					{Name: "海", Value: "naval"},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "exclude_brs",
				Description: "ルーレットから除外したいBRをカンマ区切りで入力してください。(例: 2.7,3.7,4.7)",
				Required:    false,
			},
		},
	}
}

func (c *WTBRCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// 遅延応答を送信
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		c.Log.Error("Failed to send deferred response", "error", err)
		return
	}

	// アニメーションEmbedを表示
	thinkingEmbed := &discordgo.MessageEmbed{
		Title:       "BRルーレット回転中...",
		Description: "最適なBRを探索中...",
		Color:       0x3498db, // Blue
		Image: &discordgo.MessageEmbedImage{
			URL: "https://i.gifer.com/ZNeT.gif", // 回転するGIF
		},
	}
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{thinkingEmbed},
	})

	options := i.ApplicationCommandData().Options
	mode := options[0].StringValue()
	excludeBRsStr := ""
	if len(options) > 1 {
		excludeBRsStr = options[1].StringValue()
	}

	var availableBRs []float64
	switch mode {
	case "air":
		availableBRs = airBRs
	case "ground":
		availableBRs = groundBRs
	case "naval":
		availableBRs = navalBRs
	default:
		content := "無効なゲームモードです。'air', 'ground', 'naval' のいずれかを指定してください。"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
		return
	}

	// 除外BRの処理
	excludeMap := make(map[float64]bool)
	if excludeBRsStr != "" {
		for _, brStr := range strings.Split(excludeBRsStr, ",") {
			br, parseErr := strconv.ParseFloat(strings.TrimSpace(brStr), 64)
			if parseErr != nil {
				content := fmt.Sprintf("除外BRの形式が不正です: %s", brStr)
				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: &content,
				})
				return
			}
			excludeMap[br] = true
		}
	}

	filteredBRs := []float64{}
	for _, br := range availableBRs {
		if !excludeMap[br] {
			filteredBRs = append(filteredBRs, br)
		}
	}

	if len(filteredBRs) == 0 {
		content := "選択可能なBRがありません。除外BRの設定を見直してください。"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
		return
	}

	// 乱数のシードを初期化
	rand.Seed(time.Now().UnixNano())
	// ランダムに結果を選択
	chosenBR := filteredBRs[rand.Intn(len(filteredBRs))]

	// 2秒待って結果を表示（アニメーションを見せるため）
	time.Sleep(2 * time.Second)

	// 結果表示用のEmbed
	resultEmbed := &discordgo.MessageEmbed{
		Title:       "🎉 BRルーレット結果！",
		Description: fmt.Sprintf("## **選択されたBR: %.1f**", chosenBR),
		Color:       0x2ecc71, // Green
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "ゲームモード",
				Value: mode,
				Inline: true,
			},
			{
				Name:  "除外BR",
				Value: func() string {
					if excludeBRsStr == "" {
						return "なし"
					}
					return excludeBRsStr
				}(),
				Inline: true,
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

func (c *WTBRCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *WTBRCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *WTBRCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *WTBRCommand) GetCategory() string                                                  { return "War Thunder" }
