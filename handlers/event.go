package handlers

import (
	"database/sql"
	"fmt"
	"luna/gemini"
	"luna/logger"
	"luna/storage"
	"time"

	"github.com/bwmarrin/discordgo"
)

type EventHandler struct {
	Store  *storage.DBStore
	Gemini *gemini.Client
}

func NewEventHandler(store *storage.DBStore, gemini *gemini.Client) *EventHandler {
	return &EventHandler{Store: store, Gemini: gemini}
}

func (h *EventHandler) RegisterAllHandlers(s *discordgo.Session) {
	s.AddHandler(h.HandleMessageCreate)
	s.AddHandler(h.handleMessageUpdate)
	s.AddHandler(h.handleMessageDelete)
	s.AddHandler(h.handleChannelUpdate)
	s.AddHandler(h.handleGuildMemberRemove)
	s.AddHandler(h.handleGuildUpdate)
	s.AddHandler(h.handleGuildMemberAdd)
	s.AddHandler(h.handleGuildMemberUpdate)
	s.AddHandler(h.handleGuildBanAdd)
	s.AddHandler(h.handleGuildBanRemove)
	s.AddHandler(h.handleGuildRoleCreate)
	s.AddHandler(h.handleGuildRoleUpdate)
	s.AddHandler(h.handleGuildRoleDelete)
	s.AddHandler(h.handleChannelCreate)
	s.AddHandler(h.handleChannelDelete)
	s.AddHandler(h.handleReactionAdd)
	s.AddHandler(h.handleReactionRemove)
	s.AddHandler(h.handleVoiceStateUpdate)
}

func (h *EventHandler) sendLog(s *discordgo.Session, guildID string, embed *discordgo.MessageEmbed) {
	var logConfig storage.LogConfig
	if err := h.Store.GetConfig(guildID, "log_config", &logConfig); err != nil {
		return
	}
	if logConfig.ChannelID == "" {
		return
	}
	guild, err := s.State.Guild(guildID)
	if err != nil {
		guild, _ = s.Guild(guildID)
	}
	if guild != nil {
		embed.Footer = &discordgo.MessageEmbedFooter{Text: guild.Name}
	}
	embed.Timestamp = time.Now().Format(time.RFC3339)
	s.ChannelMessageSendEmbed(logConfig.ChannelID, embed)
}

func getExecutor(s *discordgo.Session, guildID string, targetID string, action discordgo.AuditLogAction) string {
	auditLog, err := s.GuildAuditLog(guildID, "", "", int(action), 5)
	if err != nil {
		return ""
	}
	for _, entry := range auditLog.AuditLogEntries {
		if entry.TargetID == targetID {
			logTime, _ := discordgo.SnowflakeTimestamp(entry.ID)
			if time.Since(logTime) < 10*time.Second {
				return entry.UserID
			}
		}
	}
	return ""
}

func (h *EventHandler) HandleMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	logger.Info("MessageCreate event received, message cached.", "messageID", m.ID)
	if h.Gemini != nil {
		isMentioned := false
		for _, user := range m.Mentions {
			if user.ID == s.State.User.ID {
				isMentioned = true
				break
			}
		}
		if isMentioned {
			s.MessageReactionAdd(m.ChannelID, m.ID, "🤔")
			messages, err := s.ChannelMessages(m.ChannelID, 10, m.ID, "", "")
			if err != nil {
				logger.Error("会話履歴の取得に失敗", "error", err, "channelID", m.ChannelID)
				return
			}
			var history string
			for i := len(messages) - 1; i >= 0; i-- {
				msg := messages[i]
				history += fmt.Sprintf("%s: %s\n", msg.Author.Username, msg.Content)
			}
			history += fmt.Sprintf("%s: %s\n", m.Author.Username, m.Content)
			persona := "あなたは「Luna Assistant」という名前の、高性能で親切なAIアシスタントです。過去の会話の文脈を理解し、自然な対話を行ってください。一人称は「私」を使い、常にフレンドリーで丁寧な言葉遣いを心がけてください。"
			prompt := fmt.Sprintf("以下の会話履歴の続きとして、あなたの次の発言を生成してください。\n\n[会話履歴]\n%s\nLuna Assistant:", history)
			response, err := h.Gemini.GenerateContent(prompt, persona)
			if err != nil {
				logger.Error("Luna APIからの会話応答生成に失敗", "error", err)
				s.ChannelMessageSend(m.ChannelID, "すみません、少し調子が悪いようです…。")
			} else {
				s.ChannelMessageSend(m.ChannelID, response)
			}
			s.MessageReactionRemove(m.ChannelID, m.ID, "🤔", s.State.User.ID)
		}
	}
}

func (h *EventHandler) handleMessageUpdate(s *discordgo.Session, e *discordgo.MessageUpdate) {
	if e.Author == nil || e.Author.Bot {
		return
	}
	if e.BeforeUpdate == nil {
		embed := &discordgo.MessageEmbed{
			Title:  "✏️ メッセージ編集 (編集前は内容不明)",
			Color:  0x3498db,
			Author: &discordgo.MessageEmbedAuthor{Name: e.Author.String(), IconURL: e.Author.AvatarURL("")},
			Fields: []*discordgo.MessageEmbedField{
				{Name: "投稿者", Value: e.Author.Mention(), Inline: true},
				{Name: "チャンネル", Value: fmt.Sprintf("<#%s>", e.ChannelID), Inline: true},
				{Name: "メッセージ", Value: fmt.Sprintf("[リンク](https://discord.com/channels/%s/%s/%s)", e.GuildID, e.ChannelID, e.ID), Inline: true},
				{Name: "編集後", Value: "```\n" + e.Content + "\n```", Inline: false},
			},
		}
		h.sendLog(s, e.GuildID, embed)
		return
	}
	if e.Content == e.BeforeUpdate.Content {
		return
	}
	embed := &discordgo.MessageEmbed{
		Title:  "✏️ メッセージ編集",
		Color:  0x3498db,
		Author: &discordgo.MessageEmbedAuthor{Name: e.Author.String(), IconURL: e.Author.AvatarURL("")},
		Fields: []*discordgo.MessageEmbedField{
			{Name: "投稿者", Value: e.Author.Mention(), Inline: true},
			{Name: "チャンネル", Value: fmt.Sprintf("<#%s>", e.ChannelID), Inline: true},
			{Name: "メッセージ", Value: fmt.Sprintf("[リンク](https://discord.com/channels/%s/%s/%s)", e.GuildID, e.ChannelID, e.ID), Inline: true},
			{Name: "編集前", Value: "```\n" + e.BeforeUpdate.Content + "\n```", Inline: false},
			{Name: "編集後", Value: "```\n" + e.Content + "\n```", Inline: false},
		},
	}
	h.sendLog(s, e.GuildID, embed)
}

func (h *EventHandler) handleMessageDelete(s *discordgo.Session, e *discordgo.MessageDelete) {
	if e.BeforeDelete == nil {
		embed := &discordgo.MessageEmbed{
			Title:       "🗑️ メッセージ削除 (内容不明)",
			Description: fmt.Sprintf("<#%s> でメッセージが削除されました。", e.ChannelID),
			Color:       0x99aab5,
			Fields:      []*discordgo.MessageEmbedField{{Name: "メッセージID", Value: e.ID}},
		}
		h.sendLog(s, e.GuildID, embed)
		return
	}
	if e.BeforeDelete.Author == nil || e.BeforeDelete.Author.Bot {
		return
	}
	embed := &discordgo.MessageEmbed{
		Title:  "🗑️ メッセージ削除",
		Color:  0xf04747,
		Author: &discordgo.MessageEmbedAuthor{Name: e.BeforeDelete.Author.String(), IconURL: e.BeforeDelete.Author.AvatarURL("")},
		Fields: []*discordgo.MessageEmbedField{
			{Name: "投稿者", Value: e.BeforeDelete.Author.Mention(), Inline: true},
			{Name: "チャンネル", Value: fmt.Sprintf("<#%s>", e.ChannelID), Inline: true},
			{Name: "内容", Value: "```\n" + e.BeforeDelete.Content + "\n```", Inline: false},
		},
	}
	h.sendLog(s, e.GuildID, embed)
}

func (h *EventHandler) handleChannelUpdate(s *discordgo.Session, e *discordgo.ChannelUpdate) {
	if e.BeforeUpdate == nil {
		return
	}
	executorID := getExecutor(s, e.GuildID, e.ID, discordgo.AuditLogActionChannelUpdate)
	executorMention := "不明"
	if executorID != "" {
		executorMention = fmt.Sprintf("<@%s>", executorID)
	}
	var fields []*discordgo.MessageEmbedField
	if e.Name != e.BeforeUpdate.Name {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name: "名前の変更", Value: fmt.Sprintf("`%s` → `%s`", e.BeforeUpdate.Name, e.Name),
		})
	}
	if len(fields) == 0 {
		return
	}
	embed := &discordgo.MessageEmbed{
		Title:       "🔄 チャンネル更新",
		Description: fmt.Sprintf("**対象チャンネル:** <#%s>\n**実行者:** %s", e.ID, executorMention),
		Color:       0x3498db,
		Fields:      fields,
	}
	h.sendLog(s, e.GuildID, embed)
}

func (h *EventHandler) handleGuildMemberRemove(s *discordgo.Session, e *discordgo.GuildMemberRemove) {
	executorID := getExecutor(s, e.GuildID, e.User.ID, discordgo.AuditLogActionMemberKick)
	if executorID != "" {
		auditLog, _ := s.GuildAuditLog(e.GuildID, "", "", int(discordgo.AuditLogActionMemberKick), 1)
		reason := "理由なし"
		if len(auditLog.AuditLogEntries) > 0 && auditLog.AuditLogEntries[0].Reason != "" {
			reason = auditLog.AuditLogEntries[0].Reason
		}
		embed := &discordgo.MessageEmbed{
			Title: "👢 Kick", Color: 0xdd5f53,
			Author: &discordgo.MessageEmbedAuthor{Name: e.User.String(), IconURL: e.User.AvatarURL("")},
			Fields: []*discordgo.MessageEmbedField{
				{Name: "対象", Value: e.User.String(), Inline: false},
				{Name: "実行者", Value: fmt.Sprintf("<@%s>", executorID), Inline: true},
				{Name: "理由", Value: reason, Inline: true},
			},
		}
		h.sendLog(s, e.GuildID, embed)
	} else {
		embed := &discordgo.MessageEmbed{
			Title: "🚪 メンバー退出", Color: 0x99aab5,
			Author:      &discordgo.MessageEmbedAuthor{Name: e.User.String(), IconURL: e.User.AvatarURL("")},
			Description: fmt.Sprintf("**<@%s>** がサーバーから退出しました。", e.User.ID),
		}
		h.sendLog(s, e.GuildID, embed)
	}
}

func (h *EventHandler) handleGuildUpdate(s *discordgo.Session, e *discordgo.GuildUpdate) {
	executorID := getExecutor(s, e.Guild.ID, e.Guild.ID, discordgo.AuditLogActionGuildUpdate)
	executorMention := "不明"
	if executorID != "" {
		executorMention = fmt.Sprintf("<@%s>", executorID)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "⚙️ サーバー設定更新",
		Description: fmt.Sprintf("**実行者:** %s", executorMention),
		Color:       0x3498db,
	}
	h.sendLog(s, e.Guild.ID, embed)
}

func (h *EventHandler) handleGuildMemberAdd(s *discordgo.Session, e *discordgo.GuildMemberAdd) {
	embed := &discordgo.MessageEmbed{
		Title:       "✅ メンバー参加",
		Description: fmt.Sprintf("**<@%s>** がサーバーに参加しました。", e.User.ID),
		Author:      &discordgo.MessageEmbedAuthor{Name: e.User.String(), IconURL: e.User.AvatarURL("")},
		Color:       0x2ecc71,
	}
	h.sendLog(s, e.GuildID, embed)
}

func (h *EventHandler) handleGuildMemberUpdate(s *discordgo.Session, e *discordgo.GuildMemberUpdate) {
	if e.BeforeUpdate == nil {
		return
	}
	executorID := getExecutor(s, e.GuildID, e.User.ID, discordgo.AuditLogActionMemberUpdate)
	executorMention := "不明"
	if executorID != "" {
		executorMention = fmt.Sprintf("<@%s>", executorID)
	}

	isTimeoutAdded := e.CommunicationDisabledUntil != nil && (e.BeforeUpdate.CommunicationDisabledUntil == nil || e.CommunicationDisabledUntil.After(*e.BeforeUpdate.CommunicationDisabledUntil))
	isTimeoutRemoved := e.CommunicationDisabledUntil == nil && e.BeforeUpdate.CommunicationDisabledUntil != nil
	if isTimeoutAdded {
		embed := &discordgo.MessageEmbed{
			Title:  "🔇 メンバータイムアウト",
			Color:  0xe67e22,
			Author: &discordgo.MessageEmbedAuthor{Name: e.User.String(), IconURL: e.User.AvatarURL("")},
			Fields: []*discordgo.MessageEmbedField{
				{Name: "対象", Value: e.User.Mention(), Inline: true},
				{Name: "実行者", Value: executorMention, Inline: true},
				{Name: "解除日時", Value: fmt.Sprintf("<t:%d:F>", e.CommunicationDisabledUntil.Unix()), Inline: false},
			},
		}
		h.sendLog(s, e.GuildID, embed)
	} else if isTimeoutRemoved {
		embed := &discordgo.MessageEmbed{
			Title:  "🔈 タイムアウト解除",
			Color:  0x5cb85c,
			Author: &discordgo.MessageEmbedAuthor{Name: e.User.String(), IconURL: e.User.AvatarURL("")},
			Fields: []*discordgo.MessageEmbedField{
				{Name: "対象", Value: e.User.Mention(), Inline: true},
				{Name: "実行者", Value: executorMention, Inline: true},
			},
		}
		h.sendLog(s, e.GuildID, embed)
	}
}

func (h *EventHandler) handleGuildBanAdd(s *discordgo.Session, e *discordgo.GuildBanAdd) {
	embed := &discordgo.MessageEmbed{
		Title:  "🔨 BAN",
		Color:  0xff0000,
		Author: &discordgo.MessageEmbedAuthor{Name: e.User.String(), IconURL: e.User.AvatarURL("")},
		Fields: []*discordgo.MessageEmbedField{
			{Name: "対象", Value: fmt.Sprintf("%s (`%s`)", e.User.String(), e.User.ID)},
		},
	}
	h.sendLog(s, e.GuildID, embed)
}

func (h *EventHandler) handleGuildBanRemove(s *discordgo.Session, e *discordgo.GuildBanRemove) {
	embed := &discordgo.MessageEmbed{
		Title:  "🕊️ BAN解除",
		Color:  0x58d68d,
		Author: &discordgo.MessageEmbedAuthor{Name: e.User.String(), IconURL: e.User.AvatarURL("")},
		Fields: []*discordgo.MessageEmbedField{
			{Name: "対象", Value: fmt.Sprintf("%s (`%s`)", e.User.String(), e.User.ID)},
		},
	}
	h.sendLog(s, e.GuildID, embed)
}

func (h *EventHandler) handleGuildRoleCreate(s *discordgo.Session, e *discordgo.GuildRoleCreate) {
	embed := &discordgo.MessageEmbed{
		Title:       "➕ ロール作成",
		Description: fmt.Sprintf("新しいロール <@&%s> が作成されました。", e.Role.ID),
		Color:       0x2ecc71,
	}
	h.sendLog(s, e.GuildID, embed)
}

func (h *EventHandler) handleGuildRoleUpdate(s *discordgo.Session, e *discordgo.GuildRoleUpdate) {
	embed := &discordgo.MessageEmbed{
		Title:       "🔄 ロール更新",
		Description: fmt.Sprintf("ロール <@&%s> (`%s`) の設定が変更されました。", e.Role.ID, e.Role.Name),
		Color:       0x3498db,
	}
	h.sendLog(s, e.GuildID, embed)
}

func (h *EventHandler) handleGuildRoleDelete(s *discordgo.Session, e *discordgo.GuildRoleDelete) {
	embed := &discordgo.MessageEmbed{
		Title:       "➖ ロール削除",
		Description: fmt.Sprintf("ロール `%s` が削除されました。", e.RoleID),
		Color:       0xf04747,
	}
	h.sendLog(s, e.GuildID, embed)
}

func (h *EventHandler) handleChannelCreate(s *discordgo.Session, e *discordgo.ChannelCreate) {
	embed := &discordgo.MessageEmbed{
		Title:       "➕ チャンネル作成",
		Description: fmt.Sprintf("チャンネル **<#%s>** (`%s`) が作成されました。", e.ID, e.Name),
		Color:       0x2ecc71,
	}
	h.sendLog(s, e.GuildID, embed)
}

func (h *EventHandler) handleChannelDelete(s *discordgo.Session, e *discordgo.ChannelDelete) {
	embed := &discordgo.MessageEmbed{
		Title:       "➖ チャンネル削除",
		Description: fmt.Sprintf("チャンネル **%s** が削除されました。", e.Name),
		Color:       0xf04747,
	}
	h.sendLog(s, e.GuildID, embed)
}

func (h *EventHandler) handleReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	if r.UserID == s.State.User.ID {
		return
	}
	rr, err := h.Store.GetReactionRole(r.GuildID, r.MessageID, r.Emoji.APIName())
	if err != nil {
		if err != sql.ErrNoRows {
			logger.Error("リアクションロールのDB取得に失敗", "error", err, "guildID", r.GuildID)
		}
		return
	}
	err = s.GuildMemberRoleAdd(r.GuildID, r.UserID, rr.RoleID)
	if err != nil {
		logger.Error("ロールの付与に失敗", "error", err, "userID", r.UserID, "roleID", rr.RoleID)
	}
}

func (h *EventHandler) handleReactionRemove(s *discordgo.Session, r *discordgo.MessageReactionRemove) {
	if r.UserID == s.State.User.ID {
		return
	}
	rr, err := h.Store.GetReactionRole(r.GuildID, r.MessageID, r.Emoji.APIName())
	if err != nil {
		if err != sql.ErrNoRows {
			logger.Error("リアクションロールのDB取得に失敗", "error", err, "guildID", r.GuildID)
		}
		return
	}
	err = s.GuildMemberRoleRemove(r.GuildID, r.UserID, rr.RoleID)
	if err != nil {
		logger.Error("ロールの削除に失敗", "error", err, "userID", r.UserID, "roleID", rr.RoleID)
	}
}

func (h *EventHandler) handleVoiceStateUpdate(s *discordgo.Session, e *discordgo.VoiceStateUpdate) {
	var vcConfig storage.TempVCConfig
	if err := h.Store.GetConfig(e.GuildID, "temp_vc_config", &vcConfig); err != nil || vcConfig.LobbyID == "" {
		return
	}
	if e.ChannelID == vcConfig.LobbyID {
		member, err := s.State.Member(e.GuildID, e.UserID)
		if err != nil {
			member, err = s.GuildMember(e.GuildID, e.UserID)
			if err != nil {
				return
			}
		}
		newChannel, err := s.GuildChannelCreateComplex(e.GuildID, discordgo.GuildChannelCreateData{
			Name:     fmt.Sprintf("%sの部屋", member.User.Username),
			Type:     discordgo.ChannelTypeGuildVoice,
			ParentID: vcConfig.CategoryID,
		})
		if err != nil {
			logger.Error("一時VCの作成に失敗", "error", err)
			return
		}
		s.GuildMemberMove(e.GuildID, e.UserID, &newChannel.ID)
	}
	if e.BeforeUpdate != nil && e.BeforeUpdate.ChannelID != "" && e.BeforeUpdate.ChannelID != vcConfig.LobbyID {
		oldChannel, err := s.Channel(e.BeforeUpdate.ChannelID)
		if err != nil {
			return
		}
		if oldChannel.ParentID == vcConfig.CategoryID {
			guild, err := s.State.Guild(e.GuildID)
			if err != nil {
				return
			}
			found := false
			for _, vs := range guild.VoiceStates {
				if vs.ChannelID == oldChannel.ID {
					found = true
					break
				}
			}
			if !found {
				if _, err := s.ChannelDelete(oldChannel.ID); err != nil {
					logger.Error("一時VCの削除に失敗", "error", err)
				}
			}
		}
	}
}
