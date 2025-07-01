package commands

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type PowerConverterCommand struct{}

func (c *PowerConverterCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "power",
		Description: "電力・電圧・電流を計算します",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "values",
				Description: "2つの値をカンマ区切りで入力 (例: 100V,15A や 1500W,100V)",
				Required:    true,
			},
		},
	}
}

func (c *PowerConverterCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	input := i.ApplicationCommandData().Options[0].StringValue()
	parts := strings.Split(input, ",")
	if len(parts) != 2 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "値を2つ、カンマ区切りで入力してください。", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	var w, v, a float64
	var err error

	for _, part := range parts {
		part = strings.TrimSpace(strings.ToUpper(part))
		if strings.HasSuffix(part, "W") {
			w, err = strconv.ParseFloat(strings.TrimSuffix(part, "W"), 64)
		} else if strings.HasSuffix(part, "V") {
			v, err = strconv.ParseFloat(strings.TrimSuffix(part, "V"), 64)
		} else if strings.HasSuffix(part, "A") {
			a, err = strconv.ParseFloat(strings.TrimSuffix(part, "A"), 64)
		}
		if err != nil { /* エラー処理 */
			return
		}
	}

	var resultEmbed *discordgo.MessageEmbed

	if w != 0 && v != 0 { // 電流を計算
		a = w / v
		resultEmbed = c.createEmbed(w, v, a)
	} else if w != 0 && a != 0 { // 電圧を計算
		v = w / a
		resultEmbed = c.createEmbed(w, v, a)
	} else if v != 0 && a != 0 { // 電力を計算
		w = v * a
		resultEmbed = c.createEmbed(w, v, a)
	} else {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "W, V, A のうち、2種類の値を正しく入力してください。", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{resultEmbed}},
	})
}

func (c *PowerConverterCommand) createEmbed(w, v, a float64) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title: "電力計算結果",
		Color: 0xFFFF00, // Yellow
		Fields: []*discordgo.MessageEmbedField{
			{Name: "電力 (W)", Value: fmt.Sprintf("%.2f W", math.Abs(w)), Inline: true},
			{Name: "電圧 (V)", Value: fmt.Sprintf("%.2f V", math.Abs(v)), Inline: true},
			{Name: "電流 (A)", Value: fmt.Sprintf("%.2f A", math.Abs(a)), Inline: true},
		},
	}
}

func (c *PowerConverterCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
}
func (c *PowerConverterCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *PowerConverterCommand) GetComponentIDs() []string                                        { return []string{} }
