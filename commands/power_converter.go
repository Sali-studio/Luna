package commands

import (
	"fmt"
	"luna/logger"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// 各電力単位のRF/FEに対するレート
// 基準: 1 RF/FE = X 他の単位
var powerConversionRates = map[string]float64{
	"rf": 1.0,
	"fe": 1.0,
	"eu": 0.25,
	"mj": 0.1,
	"if": 1.0,
	"ae": 0.5,
	"j":  2.5,
}

// 単位の正式名称
var powerUnitFullName = map[string]string{
	"rf": "Redstone Flux",
	"fe": "Forge Energy",
	"eu": "Energy Unit (IndustrialCraft 2)",
	"mj": "Minecraft Joules (BuildCraft)",
	"if": "Industrial Foregoing",
	"ae": "AE/t (Applied Energistics 2)",
	"j":  "Joules (Mekanism)",
}

// 表示する単位の順番
var unitDisplayOrder = []string{"rf", "fe", "j", "eu", "mj", "ae", "if"}

func init() {
	var unitChoices []*discordgo.ApplicationCommandOptionChoice
	for _, key := range unitDisplayOrder {
		unitChoices = append(unitChoices, &discordgo.ApplicationCommandOptionChoice{
			Name:  fmt.Sprintf("%s (%s)", strings.ToUpper(key), powerUnitFullName[key]),
			Value: key,
		})
	}

	cmd := &discordgo.ApplicationCommand{
		Name:        "convert-power",
		Description: "Minecraft工業MODの電力単位を一覧に変換します。",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionNumber,
				Name:        "amount",
				Description: "変換したい数値",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "from",
				Description: "元の単位",
				Required:    true,
				Choices:     unitChoices,
			},
			// 「to」オプションを削除
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		logger.Info.Println("convert-power command received")

		options := i.ApplicationCommandData().Options
		optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
		for _, opt := range options {
			optionMap[opt.Name] = opt
		}

		amount := optionMap["amount"].FloatValue()
		fromUnit := optionMap["from"].StringValue()

		// --- ここからが新しい計算ロジック ---
		// 1. 元の単位を基準単位(RF/FE)に変換
		amountInRF := amount / powerConversionRates[fromUnit]

		// 2. 結果を表示するためのEmbedフィールドを生成
		var fields []*discordgo.MessageEmbedField
		for _, unitKey := range unitDisplayOrder {
			// 基準値から各単位へ変換
			convertedAmount := amountInRF * powerConversionRates[unitKey]

			// 元の単位と同じ場合はスキップせず、太字で表示する
			fieldName := fmt.Sprintf("%s (%s)", strings.ToUpper(unitKey), powerUnitFullName[unitKey])
			fieldValue := fmt.Sprintf("`%.2f`", convertedAmount)

			if unitKey == fromUnit {
				fieldValue = fmt.Sprintf("**`%.2f`**", convertedAmount)
			}

			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   fieldName,
				Value:  fieldValue,
				Inline: true,
			})
		}
		// --- ここまで ---

		// 結果表示用のEmbedを作成
		embed := &discordgo.MessageEmbed{
			Author: &discordgo.MessageEmbedAuthor{
				Name:    fmt.Sprintf("⚡ %.2f %s の変換結果", amount, strings.ToUpper(fromUnit)),
				IconURL: "https://cdn.discordapp.com/emojis/995772399329759242.png", // MODアイコンなど
			},
			Color:  0xFEE75C, // 黄色
			Fields: fields,
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
