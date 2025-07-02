package commands

import (
	"fmt"
	"luna/gemini"
	"luna/logger"
	"luna/storage"

	"github.com/bwmarrin/discordgo"
)

const (
	CreateTicketButtonID  = "create_ticket_button"
	SubmitTicketModalID   = "submit_ticket_modal"
	CloseTicketButtonID   = "close_ticket_button"
	ArchiveTicketButtonID = "archive_ticket_button"
)

type TicketCommand struct {
	Store  *storage.DBStore
	Gemini *gemini.Client
}

func (c *TicketCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:                     "ticket-setup",
		Description:              "チケット作成パネルをこのチャンネルに設置します",
		DefaultMemberPermissions: int64Ptr(discordgo.PermissionManageGuild),
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionChannel, Name: "category", Description: "チケットが作成されるカテゴリ", ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildCategory}, Required: true},
			{Type: discordgo.ApplicationCommandOptionRole, Name: "staff_role", Description: "チケットに対応するスタッフのロール", Required: true},
		},
	}
}

func (c *TicketCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	categoryID := i.ApplicationCommandData().Options[0].ChannelValue(s).ID
	staffRoleID := i.ApplicationCommandData().Options[1].RoleValue(s, i.GuildID).ID

	config := storage.TicketConfig{
		PanelChannelID: i.ChannelID,
		CategoryID:     categoryID,
		StaffRoleID:    staffRoleID,
	}
	if err := c.Store.SaveConfig(i.GuildID, "ticket_config", config); err != nil {
		logger.Error("チケット設定の保存に失敗", "error", err, "guildID", i.GuildID)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: "❌ 設定の保存に失敗しました。", Flags: discordgo.MessageFlagsEphemeral}})
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🎫 サポートチケット",
		Description: "お問い合わせやサポートが必要な場合は、下のボタンを押してチケットを作成してください。",
		Color:       0x5865F2,
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.Button{Label: "チケットを作成", Style: discordgo.SuccessButton, CustomID: CreateTicketButtonID, Emoji: &discordgo.ComponentEmoji{Name: "✉️"}},
			}}},
		},
	})
}

func (c *TicketCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	switch customID {
	case CreateTicketButtonID:
		c.showTicketModal(s, i)
	case CloseTicketButtonID:
		c.confirmCloseTicket(s, i)
	case ArchiveTicketButtonID:
		c.archiveTicket(s, i)
	}
}

func (c *TicketCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.ModalSubmitData().CustomID == SubmitTicketModalID {
		c.createTicket(s, i)
	}
}

func (c *TicketCommand) showTicketModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: SubmitTicketModalID,
			Title:    "チケット作成",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "subject", Label: "件名", Style: discordgo.TextInputShort, Placeholder: "どのようなご用件ですか？", Required: true, MaxLength: 100},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "details", Label: "詳細", Style: discordgo.TextInputParagraph, Placeholder: "問題の詳細や質問内容をできるだけ詳しくご記入ください。", Required: true, MaxLength: 2000},
				}},
			},
		},
	})
	if err != nil {
		logger.Error("チケットモーダルの表示に失敗", "error", err)
	}
}

func (c *TicketCommand) createTicket(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral}})

	data := i.ModalSubmitData()
	subject := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	details := data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	var config storage.TicketConfig
	if err := c.Store.GetConfig(i.GuildID, "ticket_config", &config); err != nil {
		logger.Error("チケット設定の取得に失敗", "error", err, "guildID", i.GuildID)
		return
	}

	counter, err := c.Store.GetNextTicketCounter(i.GuildID)
	if err != nil {
		logger.Error("チケット番号の取得に失敗", "error", err, "guildID", i.GuildID)
		return
	}

	ch, err := s.GuildChannelCreateComplex(i.GuildID, discordgo.GuildChannelCreateData{
		Name:     fmt.Sprintf("ticket-%04d-%s", counter, i.Member.User.Username),
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: config.CategoryID,
		PermissionOverwrites: []*discordgo.PermissionOverwrite{
			{ID: i.GuildID, Type: discordgo.PermissionOverwriteTypeRole, Deny: discordgo.PermissionViewChannel},
			{ID: i.Member.User.ID, Type: discordgo.PermissionOverwriteTypeMember, Allow: discordgo.PermissionViewChannel},
			{ID: config.StaffRoleID, Type: discordgo.PermissionOverwriteTypeRole, Allow: discordgo.PermissionViewChannel},
		},
	})
	if err != nil {
		logger.Error("チケットチャンネルの作成に失敗", "error", err)
		return
	}

	c.Store.CreateTicketRecord(ch.ID, i.GuildID, i.Member.User.ID)

	initialEmbed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("🎫 #%04d: %s", counter, subject),
		Description: fmt.Sprintf("**報告者:** <@%s>\n\n**詳細:**\n```\n%s\n```", i.Member.User.ID, details),
		Color:       0x5865F2,
		Footer:      &discordgo.MessageEmbedFooter{Text: "スタッフが対応しますので、しばらくお待ちください。"},
	}
	s.ChannelMessageSendComplex(ch.ID, &discordgo.MessageSend{
		Content: fmt.Sprintf("<@%s>, <@&%s>", i.Member.User.ID, config.StaffRoleID),
		Embeds:  []*discordgo.MessageEmbed{initialEmbed},
		Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "チケットを閉じる", Style: discordgo.DangerButton, CustomID: CloseTicketButtonID, Emoji: &discordgo.ComponentEmoji{Name: "🔒"}},
		}}},
	})

	content := fmt.Sprintf("✅ チケットを作成しました: <#%s>", ch.ID)
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})

	if c.Gemini != nil {
		go func() {
			s.ChannelTyping(ch.ID)
			prompt := fmt.Sprintf("以下のユーザーからのサポートリクエストに対して、考えられる解決策や次に確認すべきことを、サポート担当者のように簡潔に回答してください。\n\n件名: %s\n詳細: %s", subject, details)
			// 第2引数に空の文字列を渡し、役割設定なしで呼び出す
			aiResponse, err := c.Gemini.GenerateContent(prompt, "")
			if err != nil {
				logger.Error("Geminiによる一次回答の生成に失敗", "error", err)
				return
			}
			aiEmbed := &discordgo.MessageEmbed{
				Author:      &discordgo.MessageEmbedAuthor{Name: "AIによる一次回答", IconURL: s.State.User.AvatarURL("")},
				Description: aiResponse,
				Color:       0x4a8cf7,
				Footer:      &discordgo.MessageEmbedFooter{Text: "これはAIによる自動生成の回答です。問題が解決しない場合は、スタッフの対応をお待ちください。"},
			}
			s.ChannelMessageSendEmbed(ch.ID, aiEmbed)
		}()
	}
}

func (c *TicketCommand) confirmCloseTicket(s *discordgo.Session, i *discordgo.InteractionCreate) {
	embed := &discordgo.MessageEmbed{
		Title:       "チケットをクローズしますか？",
		Description: "このチケットをアーカイブ（書き込み禁止）します。この操作は元に戻せません。",
		Color:       0xfee75c,
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.Button{Label: "はい、アーカイブします", Style: discordgo.DangerButton, CustomID: ArchiveTicketButtonID},
			}}},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

func (c *TicketCommand) archiveTicket(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral}})
	if err != nil {
		return
	}

	// アーカイブ処理のロジックを修正
	edit := &discordgo.ChannelEdit{
		Archived: &[]bool{true}[0],
	}
	_, err = s.ChannelEditComplex(i.ChannelID, edit)

	if err != nil {
		logger.Error("チケットのアーカイブに失敗", "error", err, "channelID", i.ChannelID)
		content := "❌ アーカイブに失敗しました。BOTの権限が不足している可能性があります。"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
		return
	}

	c.Store.CloseTicketRecord(i.ChannelID)
	content := "チケットはアーカイブされました。"
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content, Components: &[]discordgo.MessageComponent{}})
}

func (c *TicketCommand) GetComponentIDs() []string {
	return []string{CreateTicketButtonID, SubmitTicketModalID, CloseTicketButtonID, ArchiveTicketButtonID}
}
