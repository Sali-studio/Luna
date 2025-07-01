package commands

import (
	"fmt"
	"math"

	"github.com/bwmarrin/discordgo"
)

type PokemonCalculatorCommand struct{}

func (c *PokemonCalculatorCommand) GetCommandDef() *discordgo.ApplicationCommand {
	float64Ptr := func(f float64) *float64 { return &f }
	return &discordgo.ApplicationCommand{
		Name:        "calc-pokemon",
		Description: "ポケモンの各種数値を計算します。",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "stats",
				Description: "ポケモンのステータス実数値を計算します",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionInteger, Name: "base_stat", Description: "種族値", Required: true},
					{Type: discordgo.ApplicationCommandOptionInteger, Name: "iv", Description: "個体値 (0-31)", Required: true, MinValue: float64Ptr(0), MaxValue: 31},
					{Type: discordgo.ApplicationCommandOptionInteger, Name: "ev", Description: "努力値 (0-252)", Required: true, MinValue: float64Ptr(0), MaxValue: 252},
					{Type: discordgo.ApplicationCommandOptionInteger, Name: "level", Description: "レベル (1-100)", Required: true, MinValue: float64Ptr(1), MaxValue: 100},
					{Type: discordgo.ApplicationCommandOptionString, Name: "stat_name", Description: "ステータス名 (HP or その他)", Required: true, Choices: []*discordgo.ApplicationCommandOptionChoice{{Name: "HP", Value: "hp"}, {Name: "その他(攻撃/防御/特攻/特防/素早さ)", Value: "other"}}},
					{Type: discordgo.ApplicationCommandOptionNumber, Name: "nature_correction", Description: "性格補正 (1.1, 1.0, 0.9)", Required: true, Choices: []*discordgo.ApplicationCommandOptionChoice{{Name: "上昇(1.1)", Value: 1.1}, {Name: "無補正(1.0)", Value: 1.0}, {Name: "下降(0.9)", Value: 0.9}}},
				},
			},
		},
	}
}

func (c *PokemonCalculatorCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options[0].Options
	if i.ApplicationCommandData().Options[0].Name == "stats" {
		c.handleStatsCalc(s, i, options)
	}
}

func (c *PokemonCalculatorCommand) handleStatsCalc(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var baseStat, iv, ev, level int64
	var statName string
	var natureCorrection float64

	for _, opt := range options {
		switch opt.Name {
		case "base_stat":
			baseStat = opt.IntValue()
		case "iv":
			iv = opt.IntValue()
		case "ev":
			ev = opt.IntValue()
		case "level":
			level = opt.IntValue()
		case "stat_name":
			statName = opt.StringValue()
		case "nature_correction":
			natureCorrection = opt.FloatValue()
		}
	}

	var result float64
	if statName == "hp" {
		result = math.Floor((float64(baseStat*2+iv)+math.Floor(float64(ev)/4))*float64(level)/100) + 10 + float64(level)
	} else {
		result = math.Floor((math.Floor((float64(baseStat*2+iv)+math.Floor(float64(ev)/4))*float64(level)/100) + 5) * natureCorrection)
	}

	embed := &discordgo.MessageEmbed{
		Title: "ポケモン ステータス実数値 計算結果",
		Color: 0xFF0000,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "入力値", Value: fmt.Sprintf("種族値: %d, 個体値: %d, 努力値: %d, Lv: %d, 性格補正: %.1f", baseStat, iv, ev, level, natureCorrection)},
			{Name: "計算結果", Value: fmt.Sprintf("実数値: **%d**", int(result))},
		},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
	})
}

func (c *PokemonCalculatorCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
}
func (c *PokemonCalculatorCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
}
func (c *PokemonCalculatorCommand) GetComponentIDs() []string { return []string{} }
