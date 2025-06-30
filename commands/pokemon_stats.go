package commands

import (
	"fmt"
	"luna/logger"
	"math"

	"github.com/bwmarrin/discordgo"
)

// 性格補正の倍率マップ
var natureMultipliers = map[string]float64{
	"up":      1.1,
	"neutral": 1.0,
	"down":    0.9,
}

func init() {
	cmd := &discordgo.ApplicationCommand{
		Name:        "calc-stats",
		Description: "ポケモンの実数値を計算します。",
		Options: []*discordgo.ApplicationCommandOption{
			// ★★★ ここからオプションの順番を修正 ★★★
			// --- 必須オプション ---
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "base_stat",
				Description: "計算したいステータスの種族値",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "stat_type",
				Description: "ステータスの種類 (HPかそれ以外か)",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "HP", Value: "hp"},
					{Name: "こうげき / ぼうぎょ / とくこう / とくぼう / すばやさ", Value: "other"},
				},
			},
			// --- 任意オプション ---
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "level",
				Description: "ポケモンのレベル (デフォルト: 50)",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "iv",
				Description: "個体値 (0-31, デフォルト: 31)",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "ev",
				Description: "努力値 (0-252, デフォルト: 0)",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "nature",
				Description: "性格補正 (デフォルト: 補正なし)",
				Required:    false,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "⬆️ 上昇補正 (1.1倍)", Value: "up"},
					{Name: "補正なし (1.0倍)", Value: "neutral"},
					{Name: "⬇️ 下降補正 (0.9倍)", Value: "down"},
				},
			},
			// ★★★ ここまで修正 ★★★
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		logger.Info.Println("calc-stats command received")

		options := i.ApplicationCommandData().Options
		optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
		for _, opt := range options {
			optionMap[opt.Name] = opt
		}

		baseStat := float64(optionMap["base_stat"].IntValue())
		statType := optionMap["stat_type"].StringValue()

		level := float64(50)
		if opt, ok := optionMap["level"]; ok {
			level = float64(opt.IntValue())
		}
		iv := float64(31)
		if opt, ok := optionMap["iv"]; ok {
			iv = float64(opt.IntValue())
		}
		ev := float64(0)
		if opt, ok := optionMap["ev"]; ok {
			ev = float64(opt.IntValue())
		}
		natureKey := "neutral"
		if opt, ok := optionMap["nature"]; ok {
			natureKey = opt.StringValue()
		}

		var result float64
		if statType == "hp" {
			result = math.Floor((baseStat*2+iv+math.Floor(ev/4))*level/100) + level + 10
		} else {
			base := math.Floor((baseStat*2+iv+math.Floor(ev/4))*level/100) + 5
			multiplier := natureMultipliers[natureKey]
			result = math.Floor(base * multiplier)
		}

		embed := &discordgo.MessageEmbed{
			Title: "📊 ポケモン実数値計算結果",
			Color: 0x3498DB,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "入力情報",
					Value: fmt.Sprintf("レベル: `%.0f`\n種族値: `%.0f`\n個体値: `%.0f`\n努力値: `%.0f`", level, baseStat, iv, ev),
				},
				{
					Name:  "計算結果",
					Value: fmt.Sprintf("▶️ **実数値: `%.0f`**", result),
				},
			},
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{embed},
			},
		})
	}

	Commands = append(Commands, cmd)
	CommandHandlers[cmd.Name] = handler
}
