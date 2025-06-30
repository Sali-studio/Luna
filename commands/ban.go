package commands

import (
	"fmt"
	"luna/logger"

	"github.com/bwmarrin/discordgo"
)

func init() {
	cmd := &discordgo.ApplicationCommand{
		Name:                     "ban",
		Description:              "指定したユーザーをサーバーからBANします。",
		DefaultMemberPermissions: int64Ptr(discordgo.PermissionBanMembers), // BAN権限を持つ人のみ実行可能
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "BANするユーザー",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "reason",
				Description: "BANする理由",
				Required:    false,
			},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		logger.Info.Println("ban command received")

		options := i.ApplicationCommandData().Options
		optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
		for _, opt := range options {
			optionMap[opt.Name] = opt
		}

		targetUser := optionMap["user"].UserValue(s)
		var reason string
		if opt, ok := optionMap["reason"]; ok {
			reason = opt.StringValue()
		}

		// BANを実行（メッセージ履歴は削除しない設定: 0日分）
		err := s.GuildBanCreateWithReason(i.GuildID, targetUser.ID, reason, 0)
		if err != nil {
			logger.Error.Printf("Failed to ban user: %v", err)
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("❌ %s のBANに失敗しました。", targetUser.Username),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		response := fmt.Sprintf("✅ **%s** を理由「%s」でサーバーからBANしました。", targetUser.Username, reason)
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
