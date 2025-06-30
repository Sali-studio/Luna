package commands

import (
	"fmt"
	"luna/logger"
	"strconv"

	"github.com/Knetic/govaluate"
	"github.com/bwmarrin/discordgo"
)

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

		// ユーザーが入力した数式を取得
		expressionStr := i.ApplicationCommandData().Options[0].StringValue()

		// govaluateを使って数式を評価
		expression, err := govaluate.NewEvaluableExpression(expressionStr)
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

		result, err := expression.Evaluate(nil)
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

		// 結果をきれいにフォーマット
		// govaluateはfloat64で結果を返すので、整数なら小数点以下を消す
		resultStr := strconv.FormatFloat(result.(float64), 'f', -1, 64)

		// 結果表示用のEmbedを作成
		embed := &discordgo.MessageEmbed{
			Author: &discordgo.MessageEmbedAuthor{
				Name:    i.Member.User.Username,
				IconURL: i.Member.User.AvatarURL(""),
			},
			Color: 0x2ECC71, // 緑色
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "問題",
					Value: fmt.Sprintf("```%s```", expressionStr),
				},
				{
					Name:  "答え",
					Value: fmt.Sprintf("```%s```", resultStr),
				},
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
