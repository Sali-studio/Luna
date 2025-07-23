package events

import (
	"fmt"
	"strings"
	"time"

	"luna/interfaces"
	"luna/storage"

	"github.com/bwmarrin/discordgo"
)



type MemberHandler struct {
	Log   interfaces.Logger
	Store interfaces.DataStore
}

func NewMemberHandler(log interfaces.Logger, store interfaces.DataStore) *MemberHandler {
	return &MemberHandler{Log: log, Store: store}
}

func (h *MemberHandler) Register(s *discordgo.Session) {
	s.AddHandler(h.onGuildMemberAdd)
	s.AddHandler(h.onGuildMemberRemove)
	s.AddHandler(h.onGuildMemberUpdate)
}

func (h *MemberHandler) onGuildMemberAdd(s *discordgo.Session, e *discordgo.GuildMemberAdd) {
	// 自動ロール機能の処理
	var autoRoleConfig storage.AutoRoleConfig
	if err := h.Store.GetConfig(e.GuildID, "autorole_config", &autoRoleConfig); err != nil {
		h.Log.Error("Failed to get autorole config from DB", "error", err, "guildID", e.GuildID)
	} else if autoRoleConfig.Enabled && autoRoleConfig.RoleID != "" {
		// ロール付与
		err := s.GuildMemberRoleAdd(e.GuildID, e.Member.User.ID, autoRoleConfig.RoleID)
		if err != nil {
			h.Log.Error("Failed to add autorole to new member", "error", err, "guildID", e.GuildID, "userID", e.Member.User.ID, "roleID", autoRoleConfig.RoleID)
		}
	}

	createdAt, _ := discordgo.SnowflakeTimestamp(e.User.ID)
	embed := &discordgo.MessageEmbed{
		Title: "✅ メンバー参加",
		Author: &discordgo.MessageEmbedAuthor{
			Name:    e.User.String(),
			IconURL: e.User.AvatarURL(""),
		},
		Description: fmt.Sprintf("**<@%s>** がサーバーに参加しました。", e.User.ID),
		Fields: []*discordgo.MessageEmbedField{
			{Name: "アカウント作成日", Value: fmt.Sprintf("<t:%d:F>", createdAt.Unix())},
		},
		Color: ColorGreen,
	}
	SendLog(s, e.GuildID, h.Store, h.Log, embed)
}

func (h *MemberHandler) onGuildMemberRemove(s *discordgo.Session, e *discordgo.GuildMemberRemove) {
	executorID := GetExecutor(s, e.GuildID, e.User.ID, discordgo.AuditLogActionMemberKick, h.Log)
	if executorID != "" {
		// Kick event
		auditLog, _ := s.GuildAuditLog(e.GuildID, "", "", int(discordgo.AuditLogActionMemberKick), 1)
		reason := "理由なし"
		if len(auditLog.AuditLogEntries) > 0 && auditLog.AuditLogEntries[0].Reason != "" {
			reason = auditLog.AuditLogEntries[0].Reason
		}
		embed := &discordgo.MessageEmbed{
			Title:  "👢 Kick",
			Color:  ColorRed,
			Author: &discordgo.MessageEmbedAuthor{Name: e.User.String(), IconURL: e.User.AvatarURL("")},
			Fields: []*discordgo.MessageEmbedField{
				{Name: "対象", Value: e.User.String(), Inline: false},
				{Name: "実行者", Value: fmt.Sprintf("<@%s>", executorID), Inline: true},
				{Name: "理由", Value: reason, Inline: true},
			},
		}
		SendLog(s, e.GuildID, h.Store, h.Log, embed)
	} else {
		// Leave event
		roles := "不明"
		if e.Member != nil && len(e.Member.Roles) > 0 {
			roleMentions := []string{}
			for _, roleID := range e.Member.Roles {
				roleMentions = append(roleMentions, fmt.Sprintf("<@&%s>", roleID))
			}
			roles = strings.Join(roleMentions, " ")
		}

		embed := &discordgo.MessageEmbed{
			Title:       "🚪 メンバー退出",
			Color:       handlers.ColorGray,
			Author:      &discordgo.MessageEmbedAuthor{Name: e.User.String(), IconURL: e.User.AvatarURL("")},
			Description: fmt.Sprintf("**<@%s>** がサーバーから退出しました。", e.User.ID),
			Fields: []*discordgo.MessageEmbedField{
				{Name: "保有していたロール", Value: roles},
			},
		}
		SendLog(s, e.GuildID, h.Store, h.Log, embed)
	}
}

func (h *MemberHandler) onGuildMemberUpdate(s *discordgo.Session, e *discordgo.GuildMemberUpdate) {
	if e.BeforeUpdate == nil {
		return
	}
	executorID := h.getExecutor(s, e.GuildID, e.User.ID, discordgo.AuditLogActionMemberUpdate)
	executorMention := "不明"
	if executorID != "" {
		executorMention = fmt.Sprintf("<@%s>", executorID)
	}
	isTimeoutAdded := e.CommunicationDisabledUntil != nil && (e.BeforeUpdate.CommunicationDisabledUntil == nil || e.CommunicationDisabledUntil.After(*e.BeforeUpdate.CommunicationDisabledUntil))
	isTimeoutRemoved := e.CommunicationDisabledUntil == nil && e.BeforeUpdate.CommunicationDisabledUntil != nil
	if isTimeoutAdded {
		embed := &discordgo.MessageEmbed{
			Title:  "🔇 メンバータイムアウト",
			Color:  ColorOrange,
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
			Color:  ColorTeal,
			Author: &discordgo.MessageEmbedAuthor{Name: e.User.String(), IconURL: e.User.AvatarURL("")},
			Fields: []*discordgo.MessageEmbedField{
				{Name: "対象", Value: e.User.Mention(), Inline: true},
				{Name: "実行者", Value: executorMention, Inline: true},
			},
		}
		h.sendLog(s, e.GuildID, embed)
	}
}

func (h *MemberHandler) sendLog(s *discordgo.Session, guildID string, embed *discordgo.MessageEmbed) {
	var logConfig storage.LogConfig
	if err := h.Store.GetConfig(guildID, ConfigKeyLog, &logConfig); err != nil {
		h.Log.Error("Failed to get log config from DB", "error", err, "guildID", guildID)
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
	if _, err := s.ChannelMessageSendEmbed(logConfig.ChannelID, embed); err != nil {
		h.Log.Error("Failed to send log embed", "error", err, "channelID", logConfig.ChannelID)
	}
}

func (h *MemberHandler) getExecutor(s *discordgo.Session, guildID string, targetID string, action discordgo.AuditLogAction) string {
	auditLog, err := s.GuildAuditLog(guildID, "", "", int(action), 5)
	if err != nil {
		h.Log.Error("Failed to get audit log", "error", err, "guildID", guildID, "action", action)
		return ""
	}
	for _, entry := range auditLog.AuditLogEntries {
		if entry.TargetID == targetID {
			logTime, _ := discordgo.SnowflakeTimestamp(entry.ID)
			if time.Since(logTime) < handlers.AuditLogTimeWindow {
				return entry.UserID
			}
		}
	}
	return ""
}
