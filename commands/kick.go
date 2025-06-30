package commands

import (
	"fmt"
	"luna/logger"

	"github.com/bwmarrin/discordgo"
)

func init() {
	cmd := &discordgo.ApplicationCommand{
		Name:                     "kick",
		Description:              "指定したユーザーをサーバーから追放します。",
		DefaultMemberPermissions: int64Ptr(discordgo.PermissionKickMembers), // Kick権限を持つ人のみ実行可能
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "追放するユーザー",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "reason",
				Description: "追放する理由",
				Required:    false, // 理由は任意
			},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		logger.Info.Println("kick command received")

		options := i.ApplicationCommandData().Options
		optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
		for _, opt := range options {
			optionMap[opt.Name] = opt
		}

		targetUser := optionMap["user"].UserValue(s)
		var reason string
		if opt, ok := optionMap["reason"]; ok {
			reason = opt.StringValue()
		} else {
			reason = "理由が指定されていません"
		}

		// 実行者が対象者より上位のロールを持っているかなどのチェックを追加すると、より安全になります

		// 実際にKickを実行
		err := s.GuildMemberDeleteWithReason(i.GuildID, targetUser.ID, reason)
		if err != nil {
			logger.Error.Printf("Failed to kick user: %v", err)
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("❌ %s の追放に失敗しました。", targetUser.Username),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		// 成功メッセージ
		response := fmt.Sprintf("✅ **%s** を理由「%s」でサーバーから追放しました。", targetUser.Username, reason)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: response,
			},
		})
	}

	Commands = append(Commands, cmd)
	CommandHandlers[cmd.Name] = handler
}
