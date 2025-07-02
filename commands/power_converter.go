package commands

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// 各エネルギー単位からRFへの変換レートを定義
var conversionRatesToRF = map[string]float64{
	"eu": 1.0 / 4.0, // 1 RF = 0.25 EU
	"rf": 1.0,
	"fe": 1.0,        // FEはRFと等価
	"j":  1.0 / 2.5,  // 1 RF = 2.5 J
	"mj": 1.0 / 10.0, // 1 RF = 0.1 MJ
	"ae": 1.0 / 2.0,  // 1 RF = 0.5 AE
	"if": 1.0,        // IFはRFと等価
}

type PowerConverterCommand struct{}

func (c *PowerConverterCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "power-converter",
		Description: "Minecraftの工業MODのエネルギー単位を相互変換します",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionNumber,
				Name:        "value",
				Description: "変換したい数値",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "unit",
				Description: "入力した数値の単位",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "EU (IndustrialCraft)", Value: "eu"},
					{Name: "RF (Thermal Expansion, etc.)", Value: "rf"},
					{Name: "FE (Forge Energy)", Value: "fe"},
					{Name: "J (Mekanism)", Value: "j"},
					{Name: "MJ (BuildCraft)", Value: "mj"},
					{Name: "AE (Applied Energistics 2)", Value: "ae"},
					{Name: "IF (Industrial Foregoing)", Value: "if"},
				},
			},
		},
	}
}

func (c *PowerConverterCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	value := options[0].FloatValue()
	fromUnit := options[1].StringValue()

	// 基準となるRFに一度変換する
	rfPerUnit, ok := conversionRatesToRF[fromUnit]
	if !ok {
		// このエラーは通常発生しないはず
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "❌ 不明な単位です。", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	valueInRF := value / rfPerUnit

	// 各単位へ換算
	euValue := valueInRF * conversionRatesToRF["eu"]
	rfValue := valueInRF // RFとFE、IFは等価
	jValue := valueInRF / conversionRatesToRF["j"]
	mjValue := valueInRF * conversionRatesToRF["mj"]
	aeValue := valueInRF * conversionRatesToRF["ae"]

	embed := &discordgo.MessageEmbed{
		Title:       "⚡ エネルギー単位 相互変換結果",
		Description: fmt.Sprintf("入力値: **`%.2f %s/t`**", value, strings.ToUpper(fromUnit)),
		Color:       0xFFC300, // Yellow
		Fields: []*discordgo.MessageEmbedField{
			{Name: "EU (IC2)", Value: fmt.Sprintf("```%.2f EU/t```", euValue), Inline: true},
			{Name: "RF/FE/IF", Value: fmt.Sprintf("```%.2f RF/t```", rfValue), Inline: true},
			{Name: "J (Mekanism)", Value: fmt.Sprintf("```%.2f J/t```", jValue), Inline: true},
			{Name: "MJ (BuildCraft)", Value: fmt.Sprintf("```%.2f MJ/t```", mjValue), Inline: true},
			{Name: "AE (AE2)", Value: fmt.Sprintf("```%.2f AE/t```", aeValue), Inline: true},
		},
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}}})
}

func (c *PowerConverterCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
}
func (c *PowerConverterCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *PowerConverterCommand) GetComponentIDs() []string                                        { return []string{} }
