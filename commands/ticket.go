package commands

import (
	"fmt"
	"luna/logger"
	"luna/storage"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	CreateTicketButtonID = "create_ticket_button"
	CloseTicketButtonID  = "close_ticket_button"
)

type TicketCommand struct {
	Store *storage.ConfigStore
}

func (c *TicketCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:                     "ticket-setup",
		Description:              "チケットパネルをこのチャンネルに設置します (要: /config ticket)",
		DefaultMemberPermissions: int64Ptr(discordgo.PermissionManageGuild),
	}
}

func (c *TicketCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	config := c.Store.GetGuildConfig(i.GuildID)
	if config.Ticket.PanelChannelID == "" || config.Ticket.CategoryID == "" || config.Ticket.StaffRoleID == "" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "チケット機能が完全に設定されていません。`/config ticket`で設定してください。", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	if config.Ticket.PanelChannelID != i.ChannelID {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: fmt.Sprintf("このコマンドは設定されたパネルチャンネル <#%s> で実行する必要があります。", config.Ticket.PanelChannelID), Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{{Title: "サポートチケット", Description: "下のボタンを押してサポートチケットを作成してください。", Color: 0x5865F2}},
			Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.Button{Label: "チケットを作成", Style: discordgo.PrimaryButton, CustomID: CreateTicketButtonID, Emoji: &discordgo.ComponentEmoji{Name: "🎫"}},
			}}},
		},
	})
}

func (c *TicketCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.MessageComponentData().CustomID {
	case CreateTicketButtonID:
		c.createTicket(s, i)
	case CloseTicketButtonID:
		c.closeTicket(s, i)
	}
}

func (c *TicketCommand) createTicket(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// 「考え中...」と即時応答
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral}})

	config := c.Store.GetGuildConfig(i.GuildID)

	// ★★★ ここからが改善点 ★★★
	// 設定がされているか事前にチェック
	if config.Ticket.CategoryID == "" || config.Ticket.StaffRoleID == "" {
		content := "❌ チケット機能がまだ管理者によって設定されていません。サーバーの管理者に連絡してください。"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
		return
	}
	// ★★★ ここまで ★★★

	config.Ticket.Counter++
	c.Store.Save()

	ch, err := s.GuildChannelCreateComplex(i.GuildID, discordgo.GuildChannelCreateData{
		Name:     fmt.Sprintf("ticket-%04d", config.Ticket.Counter),
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: config.Ticket.CategoryID,
		PermissionOverwrites: []*discordgo.PermissionOverwrite{
			{ID: i.GuildID, Type: discordgo.PermissionOverwriteTypeRole, Deny: discordgo.PermissionViewChannel},
			{ID: i.Member.User.ID, Type: discordgo.PermissionOverwriteTypeMember, Allow: discordgo.PermissionViewChannel},
			{ID: config.Ticket.StaffRoleID, Type: discordgo.PermissionOverwriteTypeRole, Allow: discordgo.PermissionViewChannel},
		},
	})
	if err != nil {
		logger.Error.Printf("チケットチャンネルの作成に失敗: %v", err)
		content := "❌ チケットチャンネルの作成に失敗しました。BOTの権限が不足している可能性があります。"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
		return
	}

	s.ChannelMessageSendComplex(ch.ID, &discordgo.MessageSend{
		Content: fmt.Sprintf("ようこそ <@%s> さん！ <@&%s> が対応しますので、ご用件をお書きください。", i.Member.User.ID, config.Ticket.StaffRoleID),
		Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "チケットを閉じる", Style: discordgo.DangerButton, CustomID: CloseTicketButtonID, Emoji: &discordgo.ComponentEmoji{Name: "🔒"}},
		}}},
	})

	content := fmt.Sprintf("✅ チケットを作成しました: <#%s>", ch.ID)
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
}

func (c *TicketCommand) closeTicket(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: "このチケットを5秒後に削除します..."}})
	time.AfterFunc(5*time.Second, func() {
		s.ChannelDelete(i.ChannelID)
	})
}

func (c *TicketCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *TicketCommand) GetComponentIDs() []string {
	return []string{CreateTicketButtonID, CloseTicketButtonID}
}
