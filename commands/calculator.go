package commands

import (
	"fmt"
	"luna/logger"
	"math"
	"strconv"

	"github.com/Knetic/govaluate"
	"github.com/bwmarrin/discordgo"
)

var functions = map[string]govaluate.ExpressionFunction{
	// --- 三角関数 ---
	"sin": func(args ...interface{}) (interface{}, error) {
		return math.Sin(args[0].(float64)), nil
	},
	"cos": func(args ...interface{}) (interface{}, error) {
		return math.Cos(args[0].(float64)), nil
	},
	"tan": func(args ...interface{}) (interface{}, error) {
		return math.Tan(args[0].(float64)), nil
	},
	// --- 基本的な数学関数 ---
	"sqrt": func(args ...interface{}) (interface{}, error) {
		return math.Sqrt(args[0].(float64)), nil
	},
	"abs": func(args ...interface{}) (interface{}, error) {
		return math.Abs(args[0].(float64)), nil
	},
	"log": func(args ...interface{}) (interface{}, error) {
		return math.Log(args[0].(float64)), nil
	},
	"log10": func(args ...interface{}) (interface{}, error) {
		return math.Log10(args[0].(float64)), nil
	},
	"ceil": func(args ...interface{}) (interface{}, error) {
		return math.Ceil(args[0].(float64)), nil
	},
	"floor": func(args ...interface{}) (interface{}, error) {
		return math.Floor(args[0].(float64)), nil
	},
	"round": func(args ...interface{}) (interface{}, error) {
		return math.Round(args[0].(float64)), nil
	},
	"max": func(args ...interface{}) (interface{}, error) {
		return math.Max(args[0].(float64), args[1].(float64)), nil
	},
	"min": func(args ...interface{}) (interface{}, error) {
		return math.Min(args[0].(float64), args[1].(float64)), nil
	},
}

func init() {
	cmd := &discordgo.ApplicationCommand{
		Name:        "calc",
		Description: "数式を計算します。",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "expression",
				Description: "計算したい数式 (例: (10 + 20) * 3 / 2)",
				Required:    true,
			},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		logger.Info.Println("calc command received")
		expressionStr := i.ApplicationCommandData().Options[0].StringValue()

		expression, err := govaluate.NewEvaluableExpressionWithFunctions(expressionStr, functions)
		if err != nil {
			logger.Error.Printf("Failed to parse expression: %v", err)
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ 無効な数式です。もう一度確認してください。",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		parameters := make(map[string]interface{}, 8)
		parameters["pi"] = math.Pi

		result, err := expression.Evaluate(parameters)
		if err != nil {
			logger.Error.Printf("Failed to evaluate expression: %v", err)
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ 計算中にエラーが発生しました。",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		resultStr := strconv.FormatFloat(result.(float64), 'f', -1, 64)

		embed := &discordgo.MessageEmbed{
			Author: &discordgo.MessageEmbedAuthor{
				Name:    i.Member.User.Username,
				IconURL: i.Member.User.AvatarURL(""),
			},
			Color: 0x2ECC71,
			Fields: []*discordgo.MessageEmbedField{
				{Name: "問題", Value: fmt.Sprintf("```%s```", expressionStr)},
				{Name: "答え", Value: fmt.Sprintf("```%s```", resultStr)},
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
