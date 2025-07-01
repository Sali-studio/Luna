package commands

import (
	"fmt"
	"luna/gemini"
	"luna/logger"
	"os"

	"github.com/bwmarrin/discordgo"
)

// SendTicketPanel はチケット作成パネルを送信するヘルパー関数
// 他のファイルからも呼び出せるように、名前の先頭を大文字にする
func SendTicketPanel(s *discordgo.Session, channelID string) {
	embed := &discordgo.MessageEmbed{
		Title:       "サポート & お問い合わせ",
		Description: "下のボタンを押して、サポートチケットを作成してください。",
		Color:       0x5865F2,
	}
	button := discordgo.Button{
		Label: "チケットを作成", Style: discordgo.PrimaryButton, Emoji: &discordgo.ComponentEmoji{Name: "✉️"}, CustomID: "open_ticket_modal",
	}
	s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{button}}},
	})
}

func HandleOpenTicketModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "ticket_creation_modal",
			Title:    "新規サポートチケット",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "subject", Label: "件名", Style: discordgo.TextInputShort, Placeholder: "例: ユーザー間のトラブルについて", Required: true, MaxLength: 100},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "details", Label: "詳細", Style: discordgo.TextInputParagraph, Placeholder: "いつ、どこで、誰が、何をしたかなど、できるだけ詳しくご記入ください。", Required: true, MaxLength: 1000},
				}},
			},
		},
	})
	if err != nil {
		logger.Error.Printf("Failed to show modal: %v", err)
	}
}

func HandleTicketCreation(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	})

	data := i.ModalSubmitData()
	subject := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	details := data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	user := i.Member.User

	config := Config.GetGuildConfig(i.GuildID)
	staffRoleID := config.Ticket.StaffRoleID
	categoryID := config.Ticket.CategoryID

	var aiResponse string
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey != "" {
		response, err := gemini.GenerateTicketResponse(apiKey, subject, details)
		if err != nil {
			logger.Error.Printf("Failed to get response from Gemini: %v", err)
			aiResponse = "AIによる一次回答の生成中にエラーが発生しました。"
		} else {
			aiResponse = response
		}
	} else {
		aiResponse = "AIによる一次回答機能は現在無効です。"
	}

	config.Ticket.Counter++
	currentTicketNumber := config.Ticket.Counter
	Config.SaveGuildConfig(i.GuildID, config)

	channelName := fmt.Sprintf("チケット-%03d", currentTicketNumber)

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

	content := fmt.Sprintf("✅ チケットを作成しました: <#%s>", ch.ID)
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
	})

	ticketEmbed := &discordgo.MessageEmbed{
		Author:      &discordgo.MessageEmbedAuthor{Name: user.Username, IconURL: user.AvatarURL("")},
		Title:       subject,
		Description: details,
		Color:       0x5865F2,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "作成者", Value: user.Mention(), Inline: true},
			{Name: "対応担当", Value: fmt.Sprintf("<@&%s>", staffRoleID), Inline: true},
			{Name: "🤖 AIによる一次回答", Value: aiResponse},
		},
		Footer: &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("チケット番号: %d", currentTicketNumber)},
	}

	closeButton := discordgo.Button{
		Label: "チケットを閉じる", Style: discordgo.DangerButton, Emoji: &discordgo.ComponentEmoji{Name: "🔒"}, CustomID: "close_ticket_button",
	}

	s.ChannelMessageSendComplex(ch.ID, &discordgo.MessageSend{
		Content:    fmt.Sprintf("ようこそ <@%s> さん。まずはAIからの回答をご確認ください。", user.ID),
		Embeds:     []*discordgo.MessageEmbed{ticketEmbed},
		Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{closeButton}}},
	})
}

func HandleTicketClose(s *discordgo.Session, i *discordgo.InteractionCreate) {
	channel, _ := s.Channel(i.ChannelID)
	config := Config.GetGuildConfig(i.GuildID)
	closedName := fmt.Sprintf("closed-%03d", config.Ticket.Counter)

	var ticketCreator *discordgo.User
	for _, overwrite := range channel.PermissionOverwrites {
		if overwrite.Type == discordgo.PermissionOverwriteTypeMember {
			member, err := s.GuildMember(i.GuildID, overwrite.ID)
			if err != nil || member.User.Bot {
				continue
			}
			ticketCreator = member.User
			break
		}
	}

	if ticketCreator == nil {
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
