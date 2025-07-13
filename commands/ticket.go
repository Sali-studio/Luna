package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"luna/interfaces"
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
	Store interfaces.DataStore
	Log   interfaces.Logger
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
		c.Log.Error("チケット設定の保存に失敗", "error", err, "guildID", i.GuildID)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: "❌ 設定の保存に失敗しました。", Flags: discordgo.MessageFlagsEphemeral}})
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🎫 サポートチケット",
		Description: "お問い合わせやサポートが必要な場合は、下のボタンを押してチケットを作成してください。",
		Color:       0x5865F2,
	}
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.Button{Label: "チケットを作成", Style: discordgo.SuccessButton, CustomID: CreateTicketButtonID, Emoji: &discordgo.ComponentEmoji{Name: "✉️"}},
			}}},
		},
	}); err != nil {
		c.Log.Error("Failed to send ticket panel", "error", err)
	}
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
		c.Log.Error("チケットモーダルの表示に失敗", "error", err)
	}
}

func (c *TicketCommand) createTicket(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral}}); err != nil {
		c.Log.Error("Failed to send initial response", "error", err)
		return
	}

	data := i.ModalSubmitData()
	subject := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	details := data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	var config storage.TicketConfig
	if err := c.Store.GetConfig(i.GuildID, "ticket_config", &config); err != nil {
		c.Log.Error("チケット設定の取得に失敗", "error", err, "guildID", i.GuildID)
		return
	}

	counter, err := c.Store.GetNextTicketCounter(i.GuildID)
	if err != nil {
		c.Log.Error("チケット番号の取得に失敗", "error", err, "guildID", i.GuildID)
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
		c.Log.Error("チケットチャンネルの作成に失敗", "error", err)
		return
	}

	if err := c.Store.CreateTicketRecord(ch.ID, i.GuildID, i.Member.User.ID); err != nil {
		c.Log.Error("Failed to create ticket record", "error", err)
	}

	initialEmbed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("🎫 #%04d: %s", counter, subject),
		Description: fmt.Sprintf("**報告者:** <@%s>\n\n**詳細:**\n```\n%s\n```", i.Member.User.ID, details),
		Color:       0x5865F2,
		Footer:      &discordgo.MessageEmbedFooter{Text: "スタッフが対応しますので、しばらくお待ちください。"},
	}
	if _, err := s.ChannelMessageSendComplex(ch.ID, &discordgo.MessageSend{
		Content: fmt.Sprintf("<@%s>, <@&%s>", i.Member.User.ID, config.StaffRoleID),
		Embeds:  []*discordgo.MessageEmbed{initialEmbed},
		Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "チケットを閉じる", Style: discordgo.DangerButton, CustomID: CloseTicketButtonID, Emoji: &discordgo.ComponentEmoji{Name: "🔒"}},
		}}},
	}); err != nil {
		c.Log.Error("Failed to send initial ticket message", "error", err)
	}

	content := fmt.Sprintf("✅ チケットを作成しました: <#%s>", ch.ID)
	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
		c.Log.Error("Failed to edit final response", "error", err)
	}

	go func() {
		if err := s.ChannelTyping(ch.ID); err != nil {
			c.Log.Warn("Failed to send typing indicator", "error", err)
		}

		persona := `あなたは「Luna Assistant」という名前の、高性能なAIアシスタントです。ここはDiscordサーバーで、ユーザーからのサポートリクエストを受け付ける「チケット」チャンネルです。
あなたの役割は、ユーザーの問題報告に対して、人間のスタッフが対応する前に、考えられる解決策や、次に確認すべき情報（ログファイル、スクリーンショット、詳しい手順など）を提示し、問題解決の第一歩を手助けすることです。
常にユーザーに寄り添い、丁寧かつ簡潔な回答を心がけてください。ユーザーの報告内容に基づいて、必要な情報を引き出す質問を投げかけたり、問題の可能性を指摘したりします。
あなたはAIであり、感情や意識はありませんが、ユーザーにとって信頼できるサポートを提供することが求められます。人間のスタッフが後から対応することを念頭に置きつつ、できる限りの情報を提供してください。`

		// 報告者の名前をAIに伝え、メンションするように指示を追加
		prompt := fmt.Sprintf("システムインストラクション（あなたの役割）に従って、以下のユーザーからのサポートリクエストに回答してください。報告者の名前は「%s」です。回答の冒頭で「%sさん、ご報告ありがとうございます。」のように、報告者の名前を呼びかけるようにしてください。\n\n[ユーザーからの報告]\n件名: %s\n詳細: %s", i.Member.User.Username, i.Member.User.Username, subject, details)

		// Pythonサーバーにリクエストを送信
		reqData := TextRequest{Prompt: fmt.Sprintf("%s\n\n%s", persona, prompt)} // ペルソナとプロンプトを結合
		reqJson, _ := json.Marshal(reqData)
		resp, err := http.Post("http://localhost:5001/generate-text", "application/json", bytes.NewBuffer(reqJson))
		if err != nil {
			c.Log.Error("Luna Assistantからの応答取得に失敗 (サーバー接続不可)", "error", err)
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var textResp TextResponse
		if err := json.Unmarshal(body, &textResp); err != nil {
			c.Log.Error("Failed to unmarshal AI response", "error", err)
			return
		}

		if textResp.Error != "" || resp.StatusCode != http.StatusOK {
			c.Log.Error("Luna Assistantからの応答取得に失敗", "error", textResp.Error)
			return
		}

		aiEmbed := &discordgo.MessageEmbed{
			Author:      &discordgo.MessageEmbedAuthor{Name: "Luna Assistantによる一次回答", IconURL: s.State.User.AvatarURL("")},
			Description: textResp.Text,
			Color:       0x4a8cf7,
			Footer:      &discordgo.MessageEmbedFooter{Text: "これはLuna Assistantによる自動生成の回答です。問題が解決しない場合は、スタッフの対応をお待ちください。"},
		}
		if _, err := s.ChannelMessageSendEmbed(ch.ID, aiEmbed); err != nil {
			c.Log.Error("Failed to send AI response", "error", err)
		}
	}()
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
				discordgo.Button{Label: "アーカイブ", Style: discordgo.DangerButton, CustomID: ArchiveTicketButtonID},
			}}},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

func (c *TicketCommand) archiveTicket(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral}}); err != nil {
		c.Log.Error("Failed to send deferred response for archiving", "error", err)
		return
	}

	// A pointer to true is needed for the ChannelEdit struct.
	archive := true
	edit := &discordgo.ChannelEdit{
		Archived: &archive,
	}

	// Attempt to archive the channel.
	if _, err := s.ChannelEditComplex(i.ChannelID, edit); err != nil {
		c.Log.Error("チケットのアーカイブに失敗", "error", err, "channelID", i.ChannelID)
		content := "❌ アーカイブに失敗しました。BOTの権限が不足している可能性があります。"
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
			c.Log.Error("Failed to edit error response for archiving", "error", err)
		}
		return
	}

	// If archiving was successful, update the database record.
	if err := c.Store.CloseTicketRecord(i.ChannelID); err != nil {
		c.Log.Error("Failed to close ticket record in DB", "error", err)
		// Continue anyway, as the user-facing action is complete.
	}

	// Let the user know it's done and remove the buttons.
	content := "チケットはアーカイブされました。"
	var emptyComponents []discordgo.MessageComponent
	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content, Components: &emptyComponents}); err != nil {
		c.Log.Error("Failed to edit final response for archiving", "error", err)
	}
}

func (c *TicketCommand) GetComponentIDs() []string {
	return []string{CreateTicketButtonID, SubmitTicketModalID, CloseTicketButtonID, ArchiveTicketButtonID}
}

func (c *TicketCommand) GetCategory() string {
	return "管理"
}
