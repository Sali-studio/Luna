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
	s.AddHandler(h.handleMessageUpdate)
	s.AddHandler(h.handleMessageDelete)
	s.AddHandler(h.handleChannelCreate)
	s.AddHandler(h.handleChannelUpdate)
	s.AddHandler(h.handleChannelDelete)
	s.AddHandler(h.handleGuildRoleCreate)
	s.AddHandler(h.handleGuildRoleUpdate)
	s.AddHandler(h.handleGuildRoleDelete)
	s.AddHandler(h.handleGuildBanAdd)
	s.AddHandler(h.handleGuildMemberAdd)
	s.AddHandler(h.handleGuildMemberRemove)
	s.AddHandler(h.handleReactionAdd)
	s.AddHandler(h.handleReactionRemove)
	s.AddHandler(h.handleVoiceStateUpdate)
}

func (h *EventHandler) sendLog(s *discordgo.Session, guildID string, embed *discordgo.MessageEmbed) {
	var logConfig storage.LogConfig
	if err := h.Store.GetConfig(guildID, "log_config", &logConfig); err != nil {
		logger.Error("ログ設定の取得に失敗", "error", err, "guildID", guildID)
		return
	}
	if logConfig.ChannelID == "" {
		return
	}
	s.ChannelMessageSendEmbed(logConfig.ChannelID, embed)
}

func (h *EventHandler) handleMessageDelete(s *discordgo.Session, e *discordgo.MessageDelete) {
	if e.BeforeDelete == nil {
		embed := &discordgo.MessageEmbed{
			Title:       "メッセージ削除 (詳細不明)",
			Description: fmt.Sprintf("キャッシュにない古いメッセージが <#%s> で削除されました。\n**メッセージID:** `%s`", e.ChannelID, e.ID),
			Color:       0x99aab5,
			Timestamp:   time.Now().Format(time.RFC3339),
		}
		h.sendLog(s, e.GuildID, embed)
		return
	}
	if e.BeforeDelete.Author == nil || e.BeforeDelete.Author.Bot {
		return
	}
	auditLog, err := s.GuildAuditLog(e.GuildID, "", "", int(discordgo.AuditLogActionMessageDelete), 1)
	executor := "不明"
	if err == nil && len(auditLog.AuditLogEntries) > 0 {
		entry := auditLog.AuditLogEntries[0]
		if entry.TargetID == e.Author.ID {
			executor = entry.UserID
		}
	}
	executorStr := "不明"
	if executor == e.Author.ID {
		executorStr = "メッセージの作成者本人"
	} else if executor != "不明" {
		executorStr = fmt.Sprintf("<@%s>", executor)
	}
	embed := &discordgo.MessageEmbed{
		Title:     "🗑️ メッセージ削除",
		Color:     0xe74c3c,
		Timestamp: time.Now().Format(time.RFC3339),
		Author: &discordgo.MessageEmbedAuthor{
			Name:    e.Author.String(),
			IconURL: e.Author.AvatarURL(""),
		},
		Fields: []*discordgo.MessageEmbedField{
			{Name: "削除されたメッセージ", Value: fmt.Sprintf("```\n%s\n```", e.BeforeDelete.Content), Inline: false},
			{Name: "チャンネル", Value: fmt.Sprintf("<#%s>", e.ChannelID), Inline: true},
			{Name: "実行者", Value: executorStr, Inline: true},
		},
	}
	h.sendLog(s, e.GuildID, embed)
}

func (h *EventHandler) handleMessageUpdate(s *discordgo.Session, e *discordgo.MessageUpdate) {
	if e.Author == nil || e.Author.Bot || e.BeforeUpdate == nil || e.Content == e.BeforeUpdate.Content {
		return
	}
	embed := &discordgo.MessageEmbed{
		Title:     "✏️ メッセージ編集",
		Color:     0x3498db,
		Timestamp: time.Now().Format(time.RFC3339),
		Author: &discordgo.MessageEmbedAuthor{
			Name:    e.Author.String(),
			IconURL: e.Author.AvatarURL(""),
		},
		Fields: []*discordgo.MessageEmbedField{
			{Name: "編集前", Value: fmt.Sprintf("```\n%s\n```", e.BeforeUpdate.Content), Inline: false},
			{Name: "編集後", Value: fmt.Sprintf("```\n%s\n```", e.Content), Inline: false},
			{Name: "チャンネル", Value: fmt.Sprintf("<#%s>", e.ChannelID), Inline: true},
			{Name: "メッセージ", Value: fmt.Sprintf("[リンク](https://discord.com/channels/%s/%s/%s)", e.GuildID, e.ChannelID, e.ID), Inline: true},
		},
	}
	h.sendLog(s, e.GuildID, embed)
}

func (h *EventHandler) handleChannelCreate(s *discordgo.Session, e *discordgo.ChannelCreate) {
	embed := &discordgo.MessageEmbed{Title: "チャンネル作成", Description: fmt.Sprintf("新しいチャンネル **%s** が作成されました。", e.Name), Color: 0x2ecc71, Timestamp: time.Now().Format(time.RFC3339)}
	h.sendLog(s, e.GuildID, embed)
}

func (h *EventHandler) handleChannelDelete(s *discordgo.Session, e *discordgo.ChannelDelete) {
	embed := &discordgo.MessageEmbed{Title: "チャンネル削除", Description: fmt.Sprintf("チャンネル **%s** が削除されました。", e.Name), Color: 0xe74c3c, Timestamp: time.Now().Format(time.RFC3339)}
	h.sendLog(s, e.GuildID, embed)
}

func (h *EventHandler) handleChannelUpdate(s *discordgo.Session, e *discordgo.ChannelUpdate) {
	if e.BeforeUpdate == nil {
		return
	}
	var changes string
	if e.Name != e.BeforeUpdate.Name {
		changes += fmt.Sprintf("**名前:** `%s` → `%s`\n", e.BeforeUpdate.Name, e.Name)
	}
	if e.Topic != e.BeforeUpdate.Topic {
		changes += "**トピックが変更されました**\n"
	}
	if changes == "" {
		return
	}
	embed := &discordgo.MessageEmbed{Title: "チャンネル更新", Description: fmt.Sprintf("チャンネル <#%s> の設定が変更されました。\n\n%s", e.ID, changes), Color: 0x3498db, Timestamp: time.Now().Format(time.RFC3339)}
	h.sendLog(s, e.GuildID, embed)
}

func (h *EventHandler) handleGuildRoleCreate(s *discordgo.Session, e *discordgo.GuildRoleCreate) {
	embed := &discordgo.MessageEmbed{Title: "ロール作成", Description: fmt.Sprintf("新しいロール <@&%s> が作成されました。", e.Role.ID), Color: 0x2ecc71, Timestamp: time.Now().Format(time.RFC3339)}
	h.sendLog(s, e.GuildID, embed)
}

func (h *EventHandler) handleGuildRoleDelete(s *discordgo.Session, e *discordgo.GuildRoleDelete) {
	embed := &discordgo.MessageEmbed{Title: "ロール削除", Description: fmt.Sprintf("ロールID `%s` が削除されました。", e.RoleID), Color: 0xe74c3c, Timestamp: time.Now().Format(time.RFC3339)}
	h.sendLog(s, e.GuildID, embed)
}

func (h *EventHandler) handleGuildRoleUpdate(s *discordgo.Session, e *discordgo.GuildRoleUpdate) {
	embed := &discordgo.MessageEmbed{Title: "ロール更新", Description: fmt.Sprintf("ロール <@&%s> (`%s`) の設定が変更されました。", e.Role.ID, e.Role.Name), Color: 0x3498db, Timestamp: time.Now().Format(time.RFC3339)}
	h.sendLog(s, e.GuildID, embed)
}

func (h *EventHandler) handleGuildBanAdd(s *discordgo.Session, e *discordgo.GuildBanAdd) {
	embed := &discordgo.MessageEmbed{Title: "メンバーがBANされました", Description: fmt.Sprintf("**ユーザー:** %s (`%s`)", e.User.String(), e.User.ID), Color: 0xff0000, Timestamp: time.Now().Format(time.RFC3339)}
	h.sendLog(s, e.GuildID, embed)
}

func (h *EventHandler) handleGuildMemberAdd(s *discordgo.Session, e *discordgo.GuildMemberAdd) {
	embed := &discordgo.MessageEmbed{Title: "メンバー参加", Description: fmt.Sprintf("**ユーザー:** %s (`%s`)", e.User.String(), e.User.ID), Color: 0x00ff00, Timestamp: time.Now().Format(time.RFC3339)}
	h.sendLog(s, e.GuildID, embed)
}

func (h *EventHandler) handleGuildMemberRemove(s *discordgo.Session, e *discordgo.GuildMemberRemove) {
	embed := &discordgo.MessageEmbed{Title: "メンバー退出", Description: fmt.Sprintf("**ユーザー:** %s (`%s`)", e.User.String(), e.User.ID), Color: 0xaaaaaa, Timestamp: time.Now().Format(time.RFC3339)}
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

func (h *EventHandler) HandleMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if h.Gemini == nil {
		return
	}
	isMentioned := false
	for _, user := range m.Mentions {
		if user.ID == s.State.User.ID {
			isMentioned = true
			break
		}
	}
	if !isMentioned || m.Author.ID == s.State.User.ID {
		return
	}
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
