package handlers

import (
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
	s.AddHandler(h.handleMessageCreate) // デバッグのため、このハンドラを最初に追加
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

// メッセージが作成された際に、それがキャッシュされたことをログに出力します。
func (h *EventHandler) handleMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// BOT自身のメッセージは無視
	if m.Author.ID == s.State.User.ID {
		return
	}
	// デバッグログ: メッセージがキャッシュされたことを確認
	logger.Info("MessageCreate event received and message should be cached", "guildID", m.GuildID, "channelID", m.ChannelID, "messageID", m.ID)

	// 元々のメンション時のAI応答機能
	if h.Gemini != nil {
		isMentioned := false
		for _, user := range m.Mentions {
			if user.ID == s.State.User.ID {
				isMentioned = true
				break
			}
		}
		if isMentioned {
			go h.HandleMention(s, m)
		}
	}
}

func (h *EventHandler) handleMessageUpdate(s *discordgo.Session, e *discordgo.MessageUpdate) {
	if e.Author == nil || e.Author.Bot {
		return
	}

	logger.Info("MessageUpdate event received", "guildID", e.GuildID, "channelID", e.ChannelID, "messageID", e.ID)

	// BeforeUpdateがnilの場合でも、キャッシュから直接取得を試みる
	var beforeContent string
	if e.BeforeUpdate != nil {
		beforeContent = e.BeforeUpdate.Content
		logger.Info("Found message in e.BeforeUpdate (cache)")
	} else {
		// BeforeUpdateがnilでも諦めずにStateから直接探す
		msg, err := s.State.Message(e.ChannelID, e.ID)
		if err == nil {
			beforeContent = msg.Content
			logger.Info("Found message in state cache directly")
		} else {
			logger.Warn("Could not find message in any cache", "error", err)
		}
	}

	// 編集後の内容と比較
	if e.Content == "" || e.Content == beforeContent {
		return
	}

	// ログを生成
	var embed *discordgo.MessageEmbed
	if beforeContent != "" {
		// 編集前後の内容が両方ある場合
		embed = &discordgo.MessageEmbed{
			Title:  "✏️ メッセージ編集",
			Color:  0x3498db, // Blue
			Author: &discordgo.MessageEmbedAuthor{Name: e.Author.String(), IconURL: e.Author.AvatarURL("")},
			Fields: []*discordgo.MessageEmbedField{
				{Name: "投稿者", Value: e.Author.Mention(), Inline: true},
				{Name: "チャンネル", Value: fmt.Sprintf("<#%s>", e.ChannelID), Inline: true},
				{Name: "メッセージ", Value: fmt.Sprintf("[リンク](https://discord.com/channels/%s/%s/%s)", e.GuildID, e.ChannelID, e.ID), Inline: true},
				{Name: "編集前", Value: "```\n" + beforeContent + "\n```", Inline: false},
				{Name: "編集後", Value: "```\n" + e.Content + "\n```", Inline: false},
			},
		}
	} else {
		// 編集前の内容が不明な場合
		embed = &discordgo.MessageEmbed{
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
	}
	h.sendLog(s, e.GuildID, embed)
}

func (h *EventHandler) HandleMention(s *discordgo.Session, m *discordgo.MessageCreate) {
	s.MessageReactionAdd(m.ChannelID, m.ID, "🤔")
	s.MessageReactionRemove(m.ChannelID, m.ID, "🤔", s.State.User.ID)
}

func (h *EventHandler) getExecutor(s *discordgo.Session, guildID string, targetID string, actionType discordgo.AuditLogAction) string {
	auditLog, err := s.GuildAuditLog(guildID, "", "", int(actionType), 5)
	if err != nil {
		return ""
	}
	for _, entry := range auditLog.AuditLogEntries {
		if entry.TargetID == targetID {
			return entry.UserID
		}
	}
	return ""
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

	executorID := h.getExecutor(s, e.GuildID, e.ID, discordgo.AuditLogActionChannelUpdate)
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
	executorID := h.getExecutor(s, e.GuildID, e.User.ID, discordgo.AuditLogActionMemberKick)
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

func (h *EventHandler) handleGuildUpdate(s *discordgo.Session, e *discordgo.GuildUpdate) { /* ... */ }
func (h *EventHandler) handleGuildMemberAdd(s *discordgo.Session, e *discordgo.GuildMemberAdd) { /* ... */
}
func (h *EventHandler) handleGuildMemberUpdate(s *discordgo.Session, e *discordgo.GuildMemberUpdate) { /* ... */
}
func (h *EventHandler) handleGuildBanAdd(s *discordgo.Session, e *discordgo.GuildBanAdd) { /* ... */ }
func (h *EventHandler) handleGuildBanRemove(s *discordgo.Session, e *discordgo.GuildBanRemove) { /* ... */
}
func (h *EventHandler) handleGuildRoleCreate(s *discordgo.Session, e *discordgo.GuildRoleCreate) { /* ... */
}
func (h *EventHandler) handleGuildRoleUpdate(s *discordgo.Session, e *discordgo.GuildRoleUpdate) { /* ... */
}
func (h *EventHandler) handleGuildRoleDelete(s *discordgo.Session, e *discordgo.GuildRoleDelete) { /* ... */
}
func (h *EventHandler) handleChannelCreate(s *discordgo.Session, e *discordgo.ChannelCreate) { /* ... */
}
func (h *EventHandler) handleChannelDelete(s *discordgo.Session, e *discordgo.ChannelDelete) { /* ... */
}
func (h *EventHandler) handleReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) { /* ... */
}
func (h *EventHandler) handleReactionRemove(s *discordgo.Session, r *discordgo.MessageReactionRemove) { /* ... */
}
func (h *EventHandler) handleVoiceStateUpdate(s *discordgo.Session, e *discordgo.VoiceStateUpdate) { /* ... */
}
