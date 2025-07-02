package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type PowerConverterCommand struct{}

func (c *PowerConverterCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "power-converter",
		Description: "Minecraftの工業MODのエネルギー単位を変換します",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "from",
				Description: "変換元の値と単位 (例: 100EU/t)",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "to",
				Description: "変換先の単位",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "EU/t (IndustrialCraft)", Value: "eu"},
					{Name: "RF/t (Thermal Expansion, etc.)", Value: "rf"},
					{Name: "J/t (Mekanism)", Value: "j"},
				},
			},
		},
	}
}

func (c *PowerConverterCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	fromInput := i.ApplicationCommandData().Options[0].StringValue()
	toUnit := i.ApplicationCommandData().Options[1].StringValue()

	value, fromUnit, err := c.parseInput(fromInput)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: fmt.Sprintf("❌ 入力形式が無効です: %v", err), Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	var valueInRF float64
	switch fromUnit {
	case "eu":
		valueInRF = value * 4.0
	case "rf":
		valueInRF = value
	case "j":
		valueInRF = value * 0.4
	}

	var resultValue float64
	var resultUnitStr string
	switch toUnit {
	case "eu":
		resultValue = valueInRF / 4.0
		resultUnitStr = "EU/t"
	case "rf":
		resultValue = valueInRF
		resultUnitStr = "RF/t"
	case "j":
		resultValue = valueInRF * 2.5
		resultUnitStr = "J/t" // 1 RF = 2.5 J
	}

	embed := &discordgo.MessageEmbed{
		Title:       "⚡ エネルギー単位変換結果",
		Description: fmt.Sprintf("**`%s`** は **`%.2f %s`** です。", fromInput, resultValue, resultUnitStr),
		Color:       0xFFC300,
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}}})
}

func (c *PowerConverterCommand) parseInput(input string) (float64, string, error) {
	input = strings.ToLower(strings.TrimSpace(input))
	var unit string
	var valueStr string

	units := []string{"eu/t", "rf/t", "j/t"}
	for _, u := range units {
		if strings.HasSuffix(input, u) {
			unit = strings.TrimSuffix(u, "/t")
			valueStr = strings.TrimSuffix(input, u)
			goto found
		}
	}
	return 0, "", fmt.Errorf("単位(EU/t, RF/t, J/t)が見つかりません")

found:
	value, err := strconv.ParseFloat(strings.TrimSpace(valueStr), 64)
	if err != nil {
		return 0, "", fmt.Errorf("数値の解析に失敗しました")
	}
	return value, unit, nil
}

func (c *PowerConverterCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
}
func (c *PowerConverterCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *PowerConverterCommand) GetComponentIDs() []string                                        { return []string{} }
