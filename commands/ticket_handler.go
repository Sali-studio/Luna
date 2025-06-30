package commands

import (
	"fmt"
	"luna/logger"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// HandleTicketCreation はチケット作成ボタンが押されたときの処理を行います
func HandleTicketCreation(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := i.Member.User
	staffRoleID := ticketStaffRoleID[i.GuildID]

	permissionOverwrites := []*discordgo.PermissionOverwrite{
		{
			ID:   i.GuildID, // @everyone
			Type: discordgo.PermissionOverwriteTypeRole,
			Deny: discordgo.PermissionViewChannel,
		},
		{
			ID:    user.ID,
			Type:  discordgo.PermissionOverwriteTypeMember,
			Allow: discordgo.PermissionViewChannel | discordgo.PermissionSendMessages | discordgo.PermissionReadMessageHistory,
		},
		{
			ID:    staffRoleID,
			Type:  discordgo.PermissionOverwriteTypeRole,
			Allow: discordgo.PermissionViewChannel | discordgo.PermissionSendMessages | discordgo.PermissionReadMessageHistory,
		},
		{
			ID:    s.State.User.ID,
			Type:  discordgo.PermissionOverwriteTypeMember,
			Allow: discordgo.PermissionViewChannel | discordgo.PermissionSendMessages | discordgo.PermissionReadMessageHistory,
		},
	}

	ch, err := s.GuildChannelCreateComplex(i.GuildID, discordgo.GuildChannelCreateData{
		Name:                 fmt.Sprintf("ticket-%s", user.Username),
		Type:                 discordgo.ChannelTypeGuildText,
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

	closeButton := discordgo.Button{
		Label:    "チケットを閉じる",
		Style:    discordgo.DangerButton,
		Emoji:    &discordgo.ComponentEmoji{Name: "🔒"},
		CustomID: "close_ticket_button",
	}

	welcomeMessage := fmt.Sprintf("ようこそ <@%s> さん。\n<@&%s> が対応しますので、ご用件をお書きください。", user.ID, staffRoleID)

	s.ChannelMessageSendComplex(ch.ID, &discordgo.MessageSend{
		Content: welcomeMessage,
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{closeButton},
			},
		},
	})
}

// HandleTicketClose はチケットを閉じるボタンが押されたときの処理を行います
func HandleTicketClose(s *discordgo.Session, i *discordgo.InteractionCreate) {
	channel, err := s.Channel(i.ChannelID)
	if err != nil {
		logger.Error.Printf("Failed to get channel info: %v", err)
		return
	}

	ticketCreatorName := strings.TrimPrefix(channel.Name, "ticket-")

	var ticketCreator *discordgo.User
	for _, overwrite := range channel.PermissionOverwrites {
		if overwrite.Type == discordgo.PermissionOverwriteTypeMember {
			member, err := s.GuildMember(i.GuildID, overwrite.ID)
			if err != nil {
				continue
			}
			if strings.EqualFold(member.User.Username, ticketCreatorName) && member.User.ID != s.State.User.ID {
				ticketCreator = member.User
				break
			}
		}
	}

	if ticketCreator == nil {
		logger.Warning.Printf("Could not find ticket creator for channel %s", channel.Name)
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
		Name:                 fmt.Sprintf("closed-%s", ticketCreatorName),
		PermissionOverwrites: newOverwrites,
	})

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("🔒 <@%s> がチケットを閉じました。", i.Member.User.ID),
		},
	})
}
