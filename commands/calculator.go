package commands

import (
	"errors"
	"fmt"
	"luna/interfaces"
	"math"

	"github.com/Knetic/govaluate"
	"github.com/bwmarrin/discordgo"
)

type CalculatorCommand struct {
	Log interfaces.Logger
}

func (c *CalculatorCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "calc",
		Description: "数式を計算します（数学関数も利用可能）",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "expression",
				Description: "計算したい数式 (例: sin(pi/2) * (2^3))",
				Required:    true,
			},
		},
	}
}

func (c *CalculatorCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	expressionStr := i.ApplicationCommandData().Options[0].StringValue()

	functions := map[string]govaluate.ExpressionFunction{
		"sin": func(args ...interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, errors.New("引数は1つ必要です")
			}
			return math.Sin(args[0].(float64)), nil
		},
		"cos": func(args ...interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, errors.New("引数は1つ必要です")
			}
			return math.Cos(args[0].(float64)), nil
		},
		"tan": func(args ...interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, errors.New("引数は1つ必要です")
			}
			return math.Tan(args[0].(float64)), nil
		},
		"log": func(args ...interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, errors.New("引数は1つ必要です")
			}
			return math.Log(args[0].(float64)), nil
		},
		"log10": func(args ...interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, errors.New("引数は1つ必要です")
			}
			return math.Log10(args[0].(float64)), nil
		},
		"sqrt": func(args ...interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, errors.New("引数は1つ必要です")
			}
			return math.Sqrt(args[0].(float64)), nil
		},
		"pow": func(args ...interface{}) (interface{}, error) {
			if len(args) != 2 {
				return nil, errors.New("引数は2つ必要です")
			}
			return math.Pow(args[0].(float64), args[1].(float64)), nil
		},
	}

	parameters := make(map[string]interface{}, 8)
	parameters["pi"] = math.Pi
	parameters["e"] = math.E

	expression, err := govaluate.NewEvaluableExpressionWithFunctions(expressionStr, functions)
	if err != nil {
		c.Log.Error("数式の解析に失敗", "error", err, "expression", expressionStr)
		errorMessage := fmt.Sprintf("❌ 無効な数式です: `%s`\n**エラー:** `%v`", expressionStr, err)
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: errorMessage, Flags: discordgo.MessageFlagsEphemeral}}); err != nil {
			c.Log.Error("Failed to send error response", "error", err)
		}
		return
	}

	result, err := expression.Evaluate(parameters)
	if err != nil {
		c.Log.Error("数式の計算に失敗", "error", err, "expression", expressionStr)
		errorMessage := fmt.Sprintf("❌ 数式の計算中にエラーが発生しました: `%s`\n**エラー:** `%v`", expressionStr, err)
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: errorMessage, Flags: discordgo.MessageFlagsEphemeral}}); err != nil {
			c.Log.Error("Failed to send error response", "error", err)
		}
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

		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}}}); err != nil {
		c.Log.Error("Failed to send response", "error", err)
	}
}

func (c *CalculatorCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *CalculatorCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *CalculatorCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *CalculatorCommand) GetCategory() string {
	return "ユーティリティ"
}