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
	Store *storage.DBStore
}

func (c *TicketCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:                     "ticket-setup",
		Description:              "チケットパネルをこのチャンネルに設置します (要: /config ticket)",
		DefaultMemberPermissions: int64Ptr(discordgo.PermissionManageGuild),
	}
}

func (c *TicketCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var config storage.TicketConfig
	if err := c.Store.GetConfig(i.GuildID, "ticket_config", &config); err != nil {
		logger.Error("チケット設定の取得に失敗", "error", err, "guildID", i.GuildID)
		return
	}
	if config.PanelChannelID == "" || config.CategoryID == "" || config.StaffRoleID == "" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: "チケット機能が完全に設定されていません。", Flags: discordgo.MessageFlagsEphemeral}})
		return
	}
	if config.PanelChannelID != i.ChannelID {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: fmt.Sprintf("このコマンドは <#%s> で実行してください。", config.PanelChannelID), Flags: discordgo.MessageFlagsEphemeral}})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{{Title: "サポートチケット", Description: "下のボタンを押してサポートチケットを作成してください。", Color: 0x5865F2}},
			Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.Button{Label: "チケットを作成", Style: discordgo.PrimaryButton, CustomID: CreateTicketButtonID, Emoji: &discordgo.ComponentEmoji{Name: "🎫"}}}}},
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
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral}})

	var config storage.TicketConfig
	if err := c.Store.GetConfig(i.GuildID, "ticket_config", &config); err != nil {
		logger.Error("チケット設定の取得に失敗", "error", err, "guildID", i.GuildID)
		return
	}
	if config.CategoryID == "" || config.StaffRoleID == "" {
		content := "❌ チケット機能が管理者によって設定されていません。"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
		return
	}

	config.Counter++
	if err := c.Store.SaveConfig(i.GuildID, "ticket_config", config); err != nil {
		logger.Error("チケットカウンターの更新に失敗", "error", err, "guildID", i.GuildID)
		return
	}

	ch, err := s.GuildChannelCreateComplex(i.GuildID, discordgo.GuildChannelCreateData{
		Name:     fmt.Sprintf("ticket-%04d", config.Counter),
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: config.CategoryID,
		PermissionOverwrites: []*discordgo.PermissionOverwrite{
			{ID: i.GuildID, Type: discordgo.PermissionOverwriteTypeRole, Deny: discordgo.PermissionViewChannel},
			{ID: i.Member.User.ID, Type: discordgo.PermissionOverwriteTypeMember, Allow: discordgo.PermissionViewChannel},
			{ID: config.StaffRoleID, Type: discordgo.PermissionOverwriteTypeRole, Allow: discordgo.PermissionViewChannel},
		},
	})
	if err != nil {
		logger.Error("チケットチャンネルの作成に失敗", "error", err, "guildID", i.GuildID)
		return
	}

	s.ChannelMessageSendComplex(ch.ID, &discordgo.MessageSend{
		Content:    fmt.Sprintf("ようこそ <@%s> さん！ <@&%s> が対応しますので、ご用件をお書きください。", i.Member.User.ID, config.StaffRoleID),
		Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.Button{Label: "チケットを閉じる", Style: discordgo.DangerButton, CustomID: CloseTicketButtonID, Emoji: &discordgo.ComponentEmoji{Name: "🔒"}}}}},
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
