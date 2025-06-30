package commands

import (
	"fmt"
	"luna/logger"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func HandleOpenTicketModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "ticket_creation_modal",
			Title:    "新規サポートチケット",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "subject",
							Label:       "件名",
							Style:       discordgo.TextInputShort,
							Placeholder: "例: ユーザー間のトラブルについて",
							Required:    true,
							MaxLength:   100,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "details",
							Label:       "詳細",
							Style:       discordgo.TextInputParagraph,
							Placeholder: "いつ、どこで、誰が、何をしたかなど、できるだけ詳しくご記入ください。",
							Required:    true,
							MaxLength:   1000,
						},
					},
				},
			},
		},
	})
	if err != nil {
		logger.Error.Printf("Failed to show modal: %v", err)
	}
}

func HandleTicketCreation(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// commands.goで定義された共有変数を参照
	ticketCounter[i.GuildID]++
	currentTicketNumber := ticketCounter[i.GuildID]

	channelName := fmt.Sprintf("チケット-%03d", currentTicketNumber)

	data := i.ModalSubmitData()
	subject := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	details := data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	user := i.Member.User
	staffRoleID := ticketStaffRoleID[i.GuildID]
	categoryID := ticketCategoryID[i.GuildID]

	permissionOverwrites := []*discordgo.PermissionOverwrite{
		{ID: i.GuildID, Type: discordgo.PermissionOverwriteTypeRole, Deny: discordgo.PermissionViewChannel},
		{ID: user.ID, Type: discordgo.PermissionOverwriteTypeMember, Allow: discordgo.PermissionViewChannel | discordgo.PermissionSendMessages},
		{ID: staffRoleID, Type: discordgo.PermissionOverwriteTypeRole, Allow: discordgo.PermissionViewChannel | discordgo.PermissionSendMessages},
		{ID: s.State.User.ID, Type: discordgo.PermissionOverwriteTypeMember, Allow: discordgo.PermissionViewChannel | discordgo.PermissionSendMessages | discordgo.PermissionManageChannels},
	}

	ch, err := s.GuildChannelCreateComplex(i.GuildID, discordgo.GuildChannelCreateData{
		Name:                 channelName,
		Type:                 discordgo.ChannelTypeGuildText,
		ParentID:             categoryID,
		PermissionOverwrites: permissionOverwrites,
	})
	if err != nil {
		logger.Error.Printf("Failed to create ticket channel: %v", err)
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ チケットを作成しました: <#%s>", ch.ID),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	ticketEmbed := &discordgo.MessageEmbed{
		Author:      &discordgo.MessageEmbedAuthor{Name: "新規チケット作成", IconURL: user.AvatarURL("")},
		Title:       subject,
		Description: details,
		Color:       0x57F287,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "作成者", Value: user.Mention(), Inline: true},
			{Name: "対応担当", Value: fmt.Sprintf("<@&%s>", staffRoleID), Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("チケット番号: %d", currentTicketNumber)},
	}

	closeButton := discordgo.Button{
		Label:    "チケットを閉じる",
		Style:    discordgo.DangerButton,
		Emoji:    &discordgo.ComponentEmoji{Name: "🔒"},
		CustomID: "close_ticket_button",
	}

	s.ChannelMessageSendComplex(ch.ID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{ticketEmbed},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{closeButton},
			},
		},
	})
}

func HandleTicketClose(s *discordgo.Session, i *discordgo.InteractionCreate) {
	channel, err := s.Channel(i.ChannelID)
	if err != nil {
		logger.Error.Printf("Failed to get channel info: %v", err)
		return
	}

	closedName := strings.Replace(channel.Name, "チケット", "closed", 1)

	var ticketCreator *discordgo.User
	for _, overwrite := range channel.PermissionOverwrites {
		if overwrite.Type == discordgo.PermissionOverwriteTypeMember {
			member, err := s.GuildMember(i.GuildID, overwrite.ID)
			if err != nil {
				continue
			}
			if !member.User.Bot {
				ticketCreator = member.User
				break
			}
		}
	}

	if ticketCreator == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "チケットの作成者が見つからなかったため、チャンネルを削除します。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		s.ChannelDelete(i.ChannelID)
		return
	}

	newOverwrites := []*discordgo.PermissionOverwrite{}
	for _, overwrite := range channel.PermissionOverwrites {
		if overwrite.ID == ticketCreator.ID {
			newOverwrites = append(newOverwrites, &discordgo.PermissionOverwrite{
				ID:   ticketCreator.ID,
				Type: discordgo.PermissionOverwriteTypeMember,
				Deny: discordgo.PermissionViewChannel,
			})
		} else {
			newOverwrites = append(newOverwrites, overwrite)
		}
	}

	s.ChannelEditComplex(i.ChannelID, &discordgo.ChannelEdit{
		Name:                 closedName,
		PermissionOverwrites: newOverwrites,
	})

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("🔒 <@%s> がチケットを閉じました。", i.Member.User.ID),
		},
	})
}
