package commands

import (
	"fmt"
	"luna/logger"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// 各電力単位の基準となる値（この場合はRF/FEを基準とする）
var powerConversionRates = map[string]float64{
	"rf": 1.0,
	"fe": 1.0,
	"eu": 4.0,
	"mj": 10.0,
	"if": 1.0, // Industrial Foregoingの電力単位 (FEと同じ)
	"ae": 2.0, // Applied Energistics 2の電力単位 (AE/t)
}

// 単位の正式名称
var powerUnitFullName = map[string]string{
	"rf": "Redstone Flux",
	"fe": "Forge Energy",
	"eu": "Energy Unit (IndustrialCraft 2)",
	"mj": "Minecraft Joules (BuildCraft)",
	"if": "Industrial Foregoing",
	"ae": "AE/t (Applied Energistics 2)",
}

func init() {
	// 各単位を選択肢として動的に生成
	var unitChoices []*discordgo.ApplicationCommandOptionChoice
	for key, name := range powerUnitFullName {
		unitChoices = append(unitChoices, &discordgo.ApplicationCommandOptionChoice{
			Name:  fmt.Sprintf("%s (%s)", strings.ToUpper(key), name),
			Value: key,
		})
	}

	cmd := &discordgo.ApplicationCommand{
		Name:        "convert-power",
		Description: "Minecraft工業MODの電力単位を相互に変換します。",
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
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "to",
				Description: "変換先の単位",
				Required:    true,
				Choices:     unitChoices,
			},
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
		toUnit := optionMap["to"].StringValue()

		// 計算ロジック
		// 1. 元の単位を基準単位(RF/FE)に変換
		amountInRF := amount * powerConversionRates[fromUnit]
		// 2. 基準単位から変換先の単位に変換
		convertedAmount := amountInRF / powerConversionRates[toUnit]

		// 結果表示用のEmbedを作成
		embed := &discordgo.MessageEmbed{
			Title: "⚡ 電力単位変換結果",
			Color: 0xFEE75C, // 黄色
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "変換前",
					Value:  fmt.Sprintf("`%.2f %s`", amount, strings.ToUpper(fromUnit)),
					Inline: true,
				},
				{
					Name:   "変換後",
					Value:  fmt.Sprintf("`%.2f %s`", convertedAmount, strings.ToUpper(toUnit)),
					Inline: true,
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: fmt.Sprintf("%s ⇄ %s", powerUnitFullName[fromUnit], powerUnitFullName[toUnit]),
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
