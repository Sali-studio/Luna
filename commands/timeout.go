package commands

import (
	"fmt"
	"luna/logger"
	"time"

	"github.com/bwmarrin/discordgo"
)

func init() {
	cmd := &discordgo.ApplicationCommand{
		Name:                     "timeout",
		Description:              "指定したユーザーをタイムアウトさせます。",
		DefaultMemberPermissions: int64Ptr(discordgo.PermissionModerateMembers), // メンバーを隔離する権限
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "タイムアウトさせるユーザー",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "duration",
				Description: "期間 (例: 5m, 1h, 3d)",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "reason",
				Description: "タイムアウトさせる理由",
				Required:    false,
			},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		logger.Info.Println("timeout command received")

		options := i.ApplicationCommandData().Options
		optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
		for _, opt := range options {
			optionMap[opt.Name] = opt
		}

		targetUser := optionMap["user"].UserValue(s)
		durationStr := optionMap["duration"].StringValue()
		var reason string
		if opt, ok := optionMap["reason"]; ok {
			reason = opt.StringValue()
		}

		duration, err := time.ParseDuration(durationStr)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ 無効な期間です。`5m` (5分), `1h` (1時間), `3d` (3日) のように指定してください。",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		until := time.Now().Add(duration)

		// --- ↓↓↓ ここからが修正箇所です ↓↓↓ ---
		// タイムアウトを実行
		// 理由がある場合は、WithAuditLogReasonオプションを追加する
		var auditLogOption discordgo.RequestOption
		if reason != "" {
			auditLogOption = discordgo.WithAuditLogReason(reason)
		}

		// GuildMemberEditは2つの値を返すので、不要な方は _ で無視する
		_, err = s.GuildMemberEdit(i.GuildID, targetUser.ID, &discordgo.GuildMemberParams{
			CommunicationDisabledUntil: &until,
		}, auditLogOption) // auditLogOption を引数に追加
		// --- ↑↑↑ ここまでが修正箇所です ↑↑↑

		if err != nil {
			logger.Error.Printf("Failed to timeout user: %v", err)
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("❌ %s のタイムアウトに失敗しました。", targetUser.Username),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		response := fmt.Sprintf("✅ **%s** を **%s** の期間タイムアウトさせました。", targetUser.Username, durationStr)
		if reason != "" {
			response += fmt.Sprintf(" (理由: %s)", reason)
		}

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
