package commands

import (
	"luna/logger"

	"github.com/bwmarrin/discordgo"
)

func init() {
	cmd := &discordgo.ApplicationCommand{
		Name:                     "ticket-setup",
		Description:              "チケット作成パネルを設置します。",
		DefaultMemberPermissions: int64Ptr(discordgo.PermissionManageChannels),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionChannel,
				Name:         "channel",
				Description:  "パネルを設置するテキストチャンネル",
				ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildText},
				Required:     true,
			},
			{
				Type:         discordgo.ApplicationCommandOptionChannel,
				Name:         "category",
				Description:  "作成されたチケットを格納するカテゴリ",
				ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildCategory},
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

		targetChannelID := optionMap["channel"].Value.(string)
		categoryID := optionMap["category"].Value.(string)
		staffRoleIDValue := optionMap["staff-role"].Value.(string)

		// commands.goで定義された共有変数に値を設定
		ticketStaffRoleID[i.GuildID] = staffRoleIDValue
		ticketCategoryID[i.GuildID] = categoryID

		embed := &discordgo.MessageEmbed{
			Title:       "サポート & お問い合わせ",
			Description: "サーバーに関するご質問や、ユーザー間のトラブル報告など、お気軽にお問い合わせください。\n\n下のボタンを押して、チケットを作成してください。",
			Color:       0x5865F2,
			Thumbnail: &discordgo.MessageEmbedThumbnail{
				URL: "https://cdn.discordapp.com/emojis/864921522055741440.png",
			},
		}

		button := discordgo.Button{
			Label:    "チケットを作成",
			Style:    discordgo.PrimaryButton,
			Emoji:    &discordgo.ComponentEmoji{Name: "✉️"},
			CustomID: "open_ticket_modal",
		}

		_, err := s.ChannelMessageSendComplex(targetChannelID, &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{button},
				},
			},
		})

		if err != nil {
			logger.Error.Printf("Failed to send ticket panel message: %v", err)
			return
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "✅ チケット作成パネルを設置しました。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	Commands = append(Commands, cmd)
	CommandHandlers[cmd.Name] = handler
}
