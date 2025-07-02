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
	s.AddHandler(h.handleMessageDelete)
	s.AddHandler(h.handleGuildBanAdd)
	s.AddHandler(h.handleGuildMemberAdd)
	s.AddHandler(h.handleGuildMemberRemove)
	s.AddHandler(h.handleVoiceStateUpdate)
	s.AddHandler(h.handleReactionAdd)
	s.AddHandler(h.handleReactionRemove)
	// メッセージ作成イベントはmain.goで直接処理するため、ここには含めない
}

func (h *EventHandler) logEvent(s *discordgo.Session, guildID string, embed *discordgo.MessageEmbed) {
	var logConfig storage.LogConfig
	if err := h.Store.GetConfig(guildID, "log_config", &logConfig); err != nil {
		// エラーログは出すが、ここで処理は中断しない
		logger.Error("ログ設定の取得に失敗", "error", err, "guildID", guildID)
		return
	}
	if logConfig.ChannelID == "" {
		return
	}
	s.ChannelMessageSendEmbed(logConfig.ChannelID, embed)
}

func (h *EventHandler) handleMessageDelete(s *discordgo.Session, e *discordgo.MessageDelete) {
	embed := &discordgo.MessageEmbed{
		Title:       "メッセージ削除",
		Description: fmt.Sprintf("メッセージが削除されました。\n**チャンネル:** <#%s>", e.ChannelID),
		Color:       0xffa500, // Orange
		Timestamp:   time.Now().Format(time.RFC3339),
	}
	h.logEvent(s, e.GuildID, embed)
}

func (h *EventHandler) handleGuildBanAdd(s *discordgo.Session, e *discordgo.GuildBanAdd) {
	embed := &discordgo.MessageEmbed{
		Title:       "ユーザーがBANされました",
		Description: fmt.Sprintf("**ユーザー:** %s (`%s`)", e.User.String(), e.User.ID),
		Color:       0xff0000, // Red
		Timestamp:   time.Now().Format(time.RFC3339),
	}
	h.logEvent(s, e.GuildID, embed)
}

func (h *EventHandler) handleGuildMemberAdd(s *discordgo.Session, e *discordgo.GuildMemberAdd) {
	embed := &discordgo.MessageEmbed{
		Title:       "メンバー参加",
		Description: fmt.Sprintf("**ユーザー:** %s (`%s`)", e.User.String(), e.User.ID),
		Color:       0x00ff00, // Green
		Timestamp:   time.Now().Format(time.RFC3339),
	}
	h.logEvent(s, e.GuildID, embed)
}

func (h *EventHandler) handleGuildMemberRemove(s *discordgo.Session, e *discordgo.GuildMemberRemove) {
	embed := &discordgo.MessageEmbed{
		Title:       "メンバー退出",
		Description: fmt.Sprintf("**ユーザー:** %s (`%s`)", e.User.String(), e.User.ID),
		Color:       0xaaaaaa, // Grey
		Timestamp:   time.Now().Format(time.RFC3339),
	}
	h.logEvent(s, e.GuildID, embed)
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
		logger.Error("Geminiからの会話応答生成に失敗", "error", err)
		s.ChannelMessageSend(m.ChannelID, "すみません、少し調子が悪いようです…")
	} else {
		s.ChannelMessageSend(m.ChannelID, response)
	}

	s.MessageReactionRemove(m.ChannelID, m.ID, "🤔", s.State.User.ID)
}
