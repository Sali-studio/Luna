// handlers/event.go
package handlers

import (
	"fmt"
	"luna/gemini"
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
	s.AddHandler(h.handleGuildUpdate)
	s.AddHandler(h.handleGuildMemberAdd)
	s.AddHandler(h.handleGuildMemberRemove)
	s.AddHandler(h.handleGuildMemberUpdate)
	s.AddHandler(h.handleGuildBanAdd)
	s.AddHandler(h.handleGuildBanRemove)
	s.AddHandler(h.handleGuildRoleCreate)
	s.AddHandler(h.handleGuildRoleUpdate)
	s.AddHandler(h.handleGuildRoleDelete)
	s.AddHandler(h.handleChannelCreate)
	s.AddHandler(h.handleChannelUpdate)
	s.AddHandler(h.handleChannelDelete)
	s.AddHandler(h.handleMessageDelete)
	s.AddHandler(h.handleMessageUpdate)
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

func (h *EventHandler) handleGuildUpdate(s *discordgo.Session, e *discordgo.GuildUpdate) { /* 実装済み */
}
func (h *EventHandler) handleGuildMemberAdd(s *discordgo.Session, e *discordgo.GuildMemberAdd) { /* 実装済み */
}
func (h *EventHandler) handleGuildMemberUpdate(s *discordgo.Session, e *discordgo.GuildMemberUpdate) { /* 実装済み */
}
func (h *EventHandler) handleGuildBanAdd(s *discordgo.Session, e *discordgo.GuildBanAdd) { /* 実装済み */
}
func (h *EventHandler) handleGuildBanRemove(s *discordgo.Session, e *discordgo.GuildBanRemove) { /* 実装済み */
}
func (h *EventHandler) handleGuildRoleCreate(s *discordgo.Session, e *discordgo.GuildRoleCreate) { /* 実装済み */
}
func (h *EventHandler) handleGuildRoleUpdate(s *discordgo.Session, e *discordgo.GuildRoleUpdate) { /* 実装済み */
}
func (h *EventHandler) handleGuildRoleDelete(s *discordgo.Session, e *discordgo.GuildRoleDelete) { /* 実装済み */
}
func (h *EventHandler) handleChannelCreate(s *discordgo.Session, e *discordgo.ChannelCreate) { /* 実装済み */
}
func (h *EventHandler) handleChannelDelete(s *discordgo.Session, e *discordgo.ChannelDelete) { /* 実装済み */
}
func (h *EventHandler) handleReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) { /* 実装済み */
}
func (h *EventHandler) handleReactionRemove(s *discordgo.Session, r *discordgo.MessageReactionRemove) { /* 実装済み */
}
func (h *EventHandler) handleVoiceStateUpdate(s *discordgo.Session, e *discordgo.VoiceStateUpdate) { /* 実装済み */
}
func (h *EventHandler) HandleMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) { /* 実装済み */
}
