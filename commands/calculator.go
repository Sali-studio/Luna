package commands

import (
	"errors"
	"fmt"
	"luna/logger"
	"math"

	"github.com/Knetic/govaluate"
	"github.com/bwmarrin/discordgo"
)

type CalculatorCommand struct{}

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

	// 利用可能な数学関数を定義します
	functions := map[string]govaluate.ExpressionFunction{
		// 三角関数
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
		// 対数関数
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
		// 指数・平方根
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

	// 定数も定義できます
	parameters := make(map[string]interface{}, 8)
	parameters["pi"] = math.Pi
	parameters["e"] = math.E

	// 新しく定義した関数を使って数式を評価する準備
	expression, err := govaluate.NewEvaluableExpressionWithFunctions(expressionStr, functions)
	if err != nil {
		logger.Error.Printf("数式の解析に失敗: %v", err)
		errorMessage := fmt.Sprintf("❌ 無効な数式です: `%s`\n**エラー:** `%v`", expressionStr, err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: errorMessage, Flags: discordgo.MessageFlagsEphemeral}})
		return
	}

	result, err := expression.Evaluate(parameters) // 定数を渡す

	if err != nil {
		logger.Error.Printf("数式の計算に失敗: %v", err)
		errorMessage := fmt.Sprintf("❌ 数式の計算中にエラーが発生しました: `%s`\n**エラー:** `%v`", expressionStr, err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: errorMessage, Flags: discordgo.MessageFlagsEphemeral}})
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

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}}})
}

func (c *CalculatorCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *CalculatorCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *CalculatorCommand) GetComponentIDs() []string                                            { return []string{} }
