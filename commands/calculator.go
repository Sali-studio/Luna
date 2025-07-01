package commands

import (
	"fmt"
	"luna/logger"
	"strings"

	"github.com/Knetic/govaluate"
	"github.com/bwmarrin/discordgo"
)

type CalculatorCommand struct{}

func (c *CalculatorCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "calc",
		Description: "数式を計算します",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "expression",
				Description: "計算したい数式 (例: (2 + 3) * 4)",
				Required:    true,
			},
		},
	}
}

func (c *CalculatorCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	expressionStr := i.ApplicationCommandData().Options[0].StringValue()

	if strings.ContainsAny(expressionStr, "abcdefghijklmnopqrstuvwxyz") {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ 無効な文字が含まれています。数値と演算子のみ使用できます。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	expression, err := govaluate.NewEvaluableExpression(expressionStr)
	if err != nil {
		logger.Error.Printf("数式の解析に失敗: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ 無効な数式です: `%s`", expressionStr),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	result, err := expression.Evaluate(nil)
	if err != nil {
		logger.Error.Printf("数式の計算に失敗: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ 数式の計算中にエラーが発生しました: `%s`", expressionStr),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	embed := &discordgo.MessageEmbed{
		Title: "🧮 計算結果",
		Fields: []*discordgo.MessageEmbedField{
			{Name: "数式", Value: fmt.Sprintf("```\n%s\n```", expressionStr)},
			{Name: "結果", Value: fmt.Sprintf("```\n%v\n```", result)},
		},
		Color: 0x57F287,
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func (c *CalculatorCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *CalculatorCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *CalculatorCommand) GetComponentIDs() []string                                            { return []string{} }
