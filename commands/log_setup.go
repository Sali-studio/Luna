package commands

import (
	"fmt"
	"luna/logger"

	"github.com/bwmarrin/discordgo"
)

// サーバーごとにログを送信するチャンネルのIDを保存
var logChannelID = make(map[string]string)

func init() {
	cmd := &discordgo.ApplicationCommand{
		Name:                     "log-setup",
		Description:              "ログを送信するチャンネルを設定します。",
		DefaultMemberPermissions: int64Ptr(discordgo.PermissionManageGuild), // サーバー管理権限
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionChannel,
				Name:         "channel",
				Description:  "ログを送信するテキストチャンネル",
				ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildText},
				Required:     true,
			},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		logger.Info.Println("log-setup command received")

		options := i.ApplicationCommandData().Options

		// 型アサーションを使って、安全にチャンネルIDを取得
		channelID, ok := options[0].Value.(string)
		if !ok {
			logger.Error.Println("Could not get channel ID from log-setup options")
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ チャンネルの取得に失敗しました。",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		// マップにチャンネルIDを保存
		logChannelID[i.GuildID] = channelID

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("✅ ログチャンネルを <#%s> に設定しました。", channelID),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	Commands = append(Commands, cmd)
	CommandHandlers[cmd.Name] = handler
}
