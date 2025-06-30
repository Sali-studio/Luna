package commands

import (
	"fmt"
	"luna/logger"

	"github.com/bwmarrin/discordgo"
)

// チケットを作成するスタッフのロールIDをサーバーごとに保存
var ticketStaffRoleID = make(map[string]string)

func init() {
	cmd := &discordgo.ApplicationCommand{
		Name:                     "ticket-setup",
		Description:              "チケット作成パネルを設置します。",
		DefaultMemberPermissions: int64Ptr(discordgo.PermissionManageChannels),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionChannel,
				Name:         "channel",
				Description:  "パネルを設置するチャンネル",
				ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildText},
				Required:     true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionRole,
				Name:        "staff-role",
				Description: "チケットに対応するスタッフのロール",
				Required:    true,
			},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		logger.Info.Println("ticket-setup command received")

		options := i.ApplicationCommandData().Options
		optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
		for _, opt := range options {
			optionMap[opt.Name] = opt
		}

		channelID, ok := optionMap["channel"].Value.(string)
		if !ok {
			logger.Error.Println("Could not get channel ID from options")
			return
		}
		roleID, ok := optionMap["staff-role"].Value.(string)
		if !ok {
			logger.Error.Println("Could not get role ID from options")
			return
		}

		targetChannel, err := s.Channel(channelID)
		if err != nil {
			logger.Error.Printf("Could not get channel object: %v", err)
			return
		}

		ticketStaffRoleID[i.GuildID] = roleID

		embed := &discordgo.MessageEmbed{
			Title:       "サポートチケット",
			Description: "下のボタンを押して、サポートチケットを作成してください。\nスタッフが順次対応します。",
			Color:       0x5865F2,
		}

		button := discordgo.Button{
			Label:    "チケットを作成",
			Style:    discordgo.SuccessButton,
			Emoji:    &discordgo.ComponentEmoji{Name: "🎫"},
			CustomID: "create_ticket_button",
		}

		_, err = s.ChannelMessageSendComplex(targetChannel.ID, &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{button},
				},
			},
		})

		if err != nil {
			logger.Error.Printf("Failed to send ticket panel message: %v", err)
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "エラー: パネルの送信に失敗しました。",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("✅ チケット作成パネルを <#%s> に設置しました。", targetChannel.ID),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	Commands = append(Commands, cmd)
	CommandHandlers[cmd.Name] = handler
}

func int64Ptr(i int64) *int64 {
	return &i
}
